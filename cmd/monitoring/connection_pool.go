package main

import (
	"context"

	"github.com/hashicorp/go-multierror"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type GRPCConnectionPool struct {
	certificate util.HitlessCertRotator
	credentials credentials.TransportCredentials
	connections map[string]*grpc.ClientConn
}

func NewGRPCConnectionPool(ctx context.Context, cfg accounts.EthConfig) (*GRPCConnectionPool, error) {
	privateKey, err := cfg.LoadKey(accounts.Silent())
	if err != nil {
		return nil, err
	}

	certificate, TLSConfig, err := util.NewHitlessCertRotator(ctx, privateKey)
	if err != nil {
		return nil, err
	}

	m := &GRPCConnectionPool{
		certificate: certificate,
		credentials: util.NewTLS(TLSConfig),
		connections: map[string]*grpc.ClientConn{},
	}

	return m, nil
}

func (m *GRPCConnectionPool) GetOrCreateConnection(target string) (*grpc.ClientConn, error) {
	conn, ok := m.connections[target]
	if ok {
		return conn, nil
	}

	conn, err := xgrpc.NewClient(context.Background(), target, m.credentials)
	if err != nil {
		return nil, err
	}

	m.connections[target] = conn
	return conn, nil
}

func (m *GRPCConnectionPool) Close() error {
	m.certificate.Close()

	err := &multierror.Error{}

	for _, conn := range m.connections {
		err = multierror.Append(err, conn.Close())
	}

	return err.ErrorOrNil()
}
