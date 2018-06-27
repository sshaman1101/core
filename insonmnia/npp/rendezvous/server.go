// Rendezvous protocol implementation. The other name - Bidirectional Locator.
//
// Rendezvous is a protocol aimed to achieve mutual address resolution for both
// client and server. This is especially useful where there is no guarantee
// that both peers are reachable directly, i.e. behind a NAT for example.
// The protocol allows for servers to publish their private network addresses
// while resolving their real remove address. This information is saved under
// some ID until a connection held and heartbeat frames transmitted.
// When a client wants to perform a connection it goes to this server, informs
// about its own private network addresses and starting from this point a
// rendezvous can be achieved.
// Both client and server are informed about remote peers public and private
// addresses and they may perform several attempts to create an actual p2p
// connection.
//
// By trying to connect to private addresses they can reach each other if and
// only if they're both in the same LAN.
// If a peer doesn't have private address, the connection is almost guaranteed
// to be established directly from the private peer to the public one.
// If both of them are located under different NATs, a TCP punching attempt can
// be performed.
// At last, when there is no hope, a special relay server can be used to
// forward the traffic.
//
// Servers should publish all of possibly reachable endpoints for all protocols
// they support.
// Clients should specify the desired protocol and ID for resolution.
//
// Currently only TCP endpoints exchanging is supported.
// TODO: When resolving it's necessary to track also IP version. For example to be able not to return IPv6 when connecting socket is IPv4.

package rendezvous

import (
	"errors"
	"math/rand"
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/memberlist"
	"github.com/sonm-io/core/insonmnia/auth"
	"github.com/sonm-io/core/insonmnia/npp/cluster"
	"github.com/sonm-io/core/insonmnia/npp/nppc"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util/netutil"
	"github.com/sonm-io/core/util/xgrpc"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type deleter func()

type peerCandidate struct {
	Peer
	C chan<- peerOrError
}

type peerOrError struct {
	Peer Peer
	Err  error
}

type meetingRoom struct {
	mu sync.Mutex
	// We allow multiple clients to be waited for servers.
	clients map[PeerID]peerCandidate
	// Also we allow the opposite: multiple servers can be registered for
	// fault tolerance.
	servers map[PeerID]peerCandidate
}

func newMeeting() *meetingRoom {
	return &meetingRoom{
		servers: map[PeerID]peerCandidate{},
		clients: map[PeerID]peerCandidate{},
	}
}

func (m *meetingRoom) addServer(peer Peer, c chan<- peerOrError) {
	m.servers[peer.ID] = peerCandidate{Peer: peer, C: c}
}

func (m *meetingRoom) addClient(peer Peer, c chan<- peerOrError) {
	m.clients[peer.ID] = peerCandidate{Peer: peer, C: c}
}

func (m *meetingRoom) randomServer() *peerCandidate {
	return randomPeerCandidate(m.servers)
}

func (m *meetingRoom) randomClient() *peerCandidate {
	return randomPeerCandidate(m.clients)
}

func randomPeerCandidate(candidates map[PeerID]peerCandidate) *peerCandidate {
	if len(candidates) == 0 {
		return nil
	}
	var keys []PeerID
	for key := range candidates {
		keys = append(keys, key)
	}

	v := candidates[keys[rand.Intn(len(keys))]]
	return &v
}

// Server represents a rendezvous server.
//
// This server is responsible for tracking servers and clients to make them
// meet each other.
type Server struct {
	cfg      ServerConfig
	log      *zap.Logger
	server   *grpc.Server
	resolver resolver

	mu        sync.Mutex
	rv        map[nppc.ResourceID]*meetingRoom
	continuum *cluster.Continuum
	cluster   *memberlist.Memberlist
	port      netutil.Port
}

// NewServer constructs a new rendezvous server using specified config and
// options.
//
// The server supports TLS by passing transport credentials using
// WithCredentials option.
// Also it is possible to activate logging system by passing a logger using
// WithLogger function as an option.
func NewServer(cfg ServerConfig, options ...Option) (*Server, error) {
	opts := newOptions()
	for _, option := range options {
		option(opts)
	}

	port, err := netutil.ExtractPort(cfg.Addr.String())
	if err != nil {
		return nil, err
	}

	m := &Server{
		cfg: cfg,
		log: opts.log,
		server: xgrpc.NewServer(
			opts.log,
			xgrpc.Credentials(opts.credentials),
			xgrpc.DefaultTraceInterceptor(),
			xgrpc.VerifyInterceptor(),
		),
		resolver:  newExternalResolver(""),
		rv:        map[nppc.ResourceID]*meetingRoom{},
		continuum: cluster.NewContinuum(),
		port:      port,
	}

	if m.cluster, err = cluster.NewCluster(cfg.Cluster, m, m.log); err != nil {
		return nil, err
	}

	m.log.Debug("configured authentication settings", zap.Any("credentials", opts.credentials.Info()))

	sonm.RegisterRendezvousServer(m.server, m)
	m.log.Debug("registered gRPC server")

	return m, nil
}

func (m *Server) Discover(ctx context.Context, in *sonm.HandshakeRequest) (*sonm.DiscoverResponse, error) {
	targetAddr, ok := m.continuum.Get(common.BytesToAddress(in.Addr))
	if !ok {
		targetAddr = m.cfg.Addr.String()
	}
	m.log.Debug("redirecting", zap.String("target", targetAddr))
	return &sonm.DiscoverResponse{Addr: targetAddr}, nil
}

func (m *Server) NotifyJoin(node *memberlist.Node) {
	m.log.Info("node has joined to the cluster", zap.String("node_name", node.Name),
		zap.String("node_addr", node.Address()))
	discardedAddrs := m.continuum.Add(m.formatEndpoint(node.Addr), 1)
	m.dropDiscardedPeers(discardedAddrs)
}

func (m *Server) NotifyLeave(node *memberlist.Node) {
	m.log.Info("node has left from the cluster", zap.String("node_name", node.Name),
		zap.String("node_addr", node.Address()))
	m.continuum.Remove(m.formatEndpoint(node.Addr))
	discardedAddrs := m.continuum.Add(m.formatEndpoint(node.Addr), 1)
	m.dropDiscardedPeers(discardedAddrs)
}

func (m *Server) NotifyUpdate(node *memberlist.Node) {
	m.log.Info("node has been updated", zap.String("node_name", node.Name))
}

func (m *Server) formatEndpoint(ip net.IP) string {
	return fmt.Sprintf("%s:%d", ip.String(), m.port)
}

func (m *Server) Resolve(ctx context.Context, request *sonm.ConnectRequest) (*sonm.RendezvousReply, error) {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errNoPeerInfo()
	}

	id := nppc.ResourceID{
		Protocol: request.Protocol,
		Addr:     common.BytesToAddress(request.ID),
	}
	m.log.Info("resolving remote peer", zap.Stringer("id", id))

	if _, ok := m.continuum.Get(common.BytesToAddress(request.ID)); !ok {
		return nil, errWrongNode()
	}

	m.log.Info("resolving remote peer", zap.String("id", request.ID))

	peerHandle := NewPeer(*peerInfo, request.PrivateAddrs)

	c, deleter := m.addServerWatch(id, peerHandle)
	defer deleter()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p := <-c:
		if p.Err != nil {
			return nil, status.Errorf(codes.Aborted, p.Err.Error())
		}
		m.log.Info("providing remote server endpoint(s)",
			zap.String("id", id),
			zap.Stringer("public_addr", p.Peer.Addr),
			zap.Any("private_addrs", p.Peer.privateAddrs),
		)
		return m.newReply(p.Peer)
	}
}

func (m *Server) ResolveAll(ctx context.Context, request *sonm.ID) (*sonm.ResolveMetaReply, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var ids []string
	for id, meeting := range m.rv {
		if id.Addr.Hex() == request.Id {
			for peerID := range meeting.servers {
				ids = append(ids, peerID.String())
			}
		}
	}

	if len(ids) == 0 {
		return nil, errPeerNotFound()
	}

	return &sonm.ResolveMetaReply{
		IDs: ids,
	}, nil
}

func (m *Server) Publish(ctx context.Context, request *sonm.PublishRequest) (*sonm.RendezvousReply, error) {
	peerInfo, ok := peer.FromContext(ctx)
	if !ok {
		return nil, errNoPeerInfo()
	}

	ethAddr, err := auth.ExtractWalletFromContext(ctx)
	if err != nil {
		return nil, err
	}

	id := nppc.ResourceID{
		Protocol: request.Protocol,
		Addr:     *ethAddr,
	}
	if nodeAddr, ok := m.continuum.Get(*ethAddr); !ok || nodeAddr != m.cfg.Addr.String() {
		return nil, errWrongNode()
	}

	m.log.Info("publishing remote peer", zap.String("id", ethAddr.String()))

	m.continuum.Track(*ethAddr)
	id := ethAddr.String()
	peerHandle := NewPeer(*peerInfo, request.PrivateAddrs)

	c, deleter := m.newClientWatch(id, peerHandle)
	defer deleter()

	m.continuum.Track(*ethAddr)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p := <-c:
		if p.Err != nil {
			return nil, status.Errorf(codes.Aborted, p.Err.Error())
		}
		m.log.Info("providing remote client endpoint(s)",
			zap.String("id", id),
			zap.Stringer("public_addr", p.Peer.Addr),
			zap.Any("private_addrs", p.Peer.privateAddrs),
		)
		return m.newReply(p.Peer)
	}
}

func (m *Server) addServerWatch(id nppc.ResourceID, peer Peer) (<-chan peerOrError, deleter) {
	c := make(chan peerOrError, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	meeting, ok := m.rv[id]
	if ok {
		// Notify both sides immediately if there is match between candidates.
		if server := meeting.randomServer(); server != nil {
			c <- peerOrError{Peer: server.Peer}
			server.C <- peerOrError{Peer: peer}
		} else {
			meeting.addClient(peer, c)
		}
	} else {
		meeting := newMeeting()
		meeting.addClient(peer, c)
		m.rv[id] = meeting
	}

	return c, func() { m.removeServerWatch(id, peer) }
}

func (m *Server) newClientWatch(id nppc.ResourceID, peer Peer) (<-chan peerOrError, deleter) {
	c := make(chan peerOrError, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	meeting, ok := m.rv[id]
	if ok {
		if client := meeting.randomClient(); client != nil {
			c <- peerOrError{Peer: client.Peer}
			client.C <- peerOrError{Peer: peer}
		} else {
			meeting.addServer(peer, c)
		}
	} else {
		meeting := newMeeting()
		meeting.addServer(peer, c)
		m.rv[id] = meeting
	}

	return c, func() { m.removeClientWatch(id, peer) }
}

func (m *Server) removeClientWatch(id nppc.ResourceID, peer Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	candidates, ok := m.rv[id]
	if ok {
		delete(candidates.servers, peer.ID)
	}

	m.maybeCleanMeeting(id, candidates)
}

func (m *Server) removeServerWatch(id nppc.ResourceID, peer Peer) {
	m.mu.Lock()
	defer m.mu.Unlock()

	candidates, ok := m.rv[id]
	if ok {
		delete(candidates.clients, peer.ID)
	}

	m.maybeCleanMeeting(id, candidates)
}

func (m *Server) maybeCleanMeeting(id nppc.ResourceID, candidates *meetingRoom) {
	if candidates == nil {
		return
	}

	if len(candidates.clients) == 0 && len(candidates.servers) == 0 {
		delete(m.rv, id)
	}
}

func (m *Server) newReply(peer Peer) (*sonm.RendezvousReply, error) {
	addr, err := sonm.NewAddr(peer.Addr)
	if err != nil {
		return nil, err
	}

	// If peer's public IP address suddenly arrived as a private this only
	// means that both Rendezvous and server/client are located within the
	// same VLAN. This should rarely happen, but in these cases we must
	// resolve public IP of ours.
	if addr.IsPrivate() {
		publicIP, err := m.resolver.PublicIP()
		if err != nil {
			return nil, err
		}

		addr.Addr.Addr = publicIP.String()
	}

	return &sonm.RendezvousReply{
		PublicAddr:   addr,
		PrivateAddrs: peer.privateAddrs,
	}, nil
}

func (m *Server) Info(ctx context.Context, request *sonm.Empty) (*sonm.RendezvousState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := make(map[string]*sonm.RendezvousMeeting)

	for id, meeting := range m.rv {
		clients := make(map[string]*sonm.RendezvousReply)
		servers := make(map[string]*sonm.RendezvousReply)

		for clientID, candidate := range meeting.clients {
			reply, err := m.newReply(candidate.Peer)
			if err != nil {
				return nil, err
			}

			clients[clientID.String()] = reply
		}

		for serverID, candidate := range meeting.servers {
			reply, err := m.newReply(candidate.Peer)
			if err != nil {
				return nil, err
			}
			servers[serverID.String()] = reply
		}

		state[id.String()] = &sonm.RendezvousMeeting{
			Servers: servers,
			Clients: clients,
		}
	}

	return &sonm.RendezvousState{
		State: state,
	}, nil
}

// Run starts accepting incoming connections, serving them by blocking the
// caller execution context until either explicitly terminated using Stop
// or some critical error occurred.
//
// Always returns non-nil error.
func (m *Server) Run() error {
	listener, err := net.Listen(m.cfg.Addr.Network(), m.cfg.Addr.String())
	if err != nil {
		return err
	}

	nodes, err := m.cluster.Join(m.cfg.Cluster.Members)
	if err != nil {
		return err
	}

	m.log.Info("joined cluster", zap.Int("num_nodes", nodes))
	m.log.Info("rendezvous is ready to serve", zap.Stringer("endpoint", listener.Addr()))
	return m.server.Serve(listener)
}

// Stop stops the server.
//
// It immediately closes all open connections and listeners. Also it cancels
// all active RPCs on the server side and the corresponding pending RPCs on
// the client side will get notified by connection errors.
func (m *Server) Stop() {
	m.log.Info("rendezvous is shutting down")
	m.server.Stop()
}

func errNoPeerInfo() error {
	return errors.New("no peer info provided")
}

func (m *Server) dropDiscardedPeers(discardedAddrs []common.Address) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, addr := range discardedAddrs {
		if room, ok := m.rv[addr.Hex()]; ok {
			for _, server := range room.servers {
				server.C <- peerOrError{Err: errors.New("server dropped due to cluster reconfiguration")}
			}
			for _, client := range room.clients {
				client.C <- peerOrError{Err: errors.New("client dropped due to cluster reconfiguration")}
			}
		}
	}
}

func errPeerNotFound() error {
	return status.Errorf(codes.NotFound, "peer not found")
}

func errWrongNode() error {
	return status.Errorf(codes.Aborted, "peer connected to wrong node")
}
