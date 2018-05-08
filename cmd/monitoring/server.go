package main

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/mapstructure"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/cmd"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	configFlag  string
	versionFlag bool
	appVersion  string
)

type Method struct {
	FullName     string
	MessageType  reflect.Type
	ResponseType reflect.Type
	MethodValue  reflect.Value
}

type Service struct {
	FullName string
	Methods  map[string]*Method
	Service  interface{}
}

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

type GRPCRegistry struct {
	connections *GRPCConnectionPool
	services    map[string]*Service
}

func NewGRPCRegistry(ctx context.Context, cfg accounts.EthConfig) (*GRPCRegistry, error) {
	connections, err := NewGRPCConnectionPool(ctx, cfg)
	if err != nil {
		return nil, err
	}

	m := &GRPCRegistry{
		connections: connections,
		services:    map[string]*Service{},
	}

	return m, nil
}

func (m *GRPCRegistry) RegisterService(interfacePtr interface{}) error {
	//}
	//	conn, err := m.connections.GetOrCreateConnection(target)
	//	if err != nil {
	//		return err
	//	}
	//
	//	return m.RegisterServiceClient(interfacePtr, sonm.NewDWHClient(conn))
	//}

	//func (m *GRPCRegistry) RegisterServiceClient(interfacePtr, concretePtr interface{}) error {
	interfaceType := reflect.TypeOf(interfacePtr).Elem()
	serviceName := strings.Replace(interfaceType.Name(), "Client", "Server", 1)

	service, ok := m.services[serviceName]
	if ok {
		return fmt.Errorf("service %s has already been registered", serviceName)
	}

	fullServiceName := "/" + strings.Replace(interfaceType.String(), "Client", "", 1)

	service = &Service{
		Methods:  map[string]*Method{},
		Service:  concretePtr,
		FullName: fullServiceName,
	}

	concrete := reflect.ValueOf(concretePtr)
	if !concrete.Type().Implements(interfaceType) {
		return fmt.Errorf("concrete object of type %s must implement the provided interface %s", concrete.Type().Name(), interfaceType.Name())
	}

	for i := 0; i < interfaceType.NumMethod(); i++ {
		method := interfaceType.Method(i)
		fmt.Printf("r M, %d %v\n", i, method)

		if method.Type.NumIn() != 2 {
			continue
		}

		messageType := method.Type.In(1)
		streamType := reflect.TypeOf((*grpc.ServerStream)(nil)).Elem()
		if messageType.Implements(streamType) {
			continue
		}

		if method.Type.NumOut() != 2 {
			return fmt.Errorf("failed not register method %s for service %s - invalid number of returned arguments", method.Name, serviceName)
		}

		service.Methods[method.Name] = &Method{
			FullName:     fullServiceName + "/" + method.Name,
			MessageType:  method.Type.In(1),
			ResponseType: method.Type.Out(0),
			MethodValue:  concrete.MethodByName(method.Name),
		}
	}

	m.services[serviceName] = service

	return nil
}

func (m *GRPCRegistry) Services() map[string]*Service {
	return m.services
}

type Manager struct {
}

func main() {
	cmd.NewCmd("monitoring", appVersion, &configFlag, &versionFlag, run).Execute()
}

func run() error {
	cfg, err := NewConfig(configFlag)
	if err != nil {
		return fmt.Errorf("failed to load config file: %s", err)
	}

	//log := logging.BuildLogger(cfg.Logging.LogLevel()).Sugar()

	wg, ctx := errgroup.WithContext(context.Background())

	registry, err := NewGRPCRegistry(ctx, cfg.Account)
	if err != nil {
		return fmt.Errorf("failed to construct service registry: %s", err)
	}

	registry.RegisterService((*sonm.DWHClient)(nil))

	// Sanitize.
	for id, c := range cfg.Checks {
		switch c.Type {
		case "eth":
		case "grpc":
			check := c

			parts := strings.Split(check.RequestURI, "/")
			if len(parts) != 2 {
				return fmt.Errorf("service URI %s must be in format {service}/{method}", check.RequestURI)
			}

			serviceName, methodName := parts[0], parts[1]

			service, err := registry.GetOrCreateService(serviceName, check.Target)
			if err != nil {
				return fmt.Errorf("failed to obtain service: %s", err)
			}

			fmt.Printf("%v\n", service.Methods)
			method, ok := service.Methods[methodName]
			if !ok {
				return fmt.Errorf("method %s for service %s not found", methodName, serviceName)
			}

			requestValue := reflect.New(method.MessageType)
			if err := mapstructure.Decode(check.Args, requestValue.Interface()); err != nil {
				return fmt.Errorf("failed to deserialize `args` into a request value: %s", err)
			}

			callParam := []reflect.Value{
				reflect.ValueOf(ctx),
				reflect.ValueOf(requestValue),
			}

			wg.Go(func() error {
				resp := method.MethodValue.Call(callParam)
				if len(resp) != 2 {
				}

				fmt.Printf("!!! %v+\n", resp)
				// Get service from registry by name.
				// Deserialize args into a type.
				// Create closure.
				//cfg.Checks[id]
				return nil
			})
		default:
			return fmt.Errorf("unknown check %d type: %s", id, c.Type)
		}
	}

	return wg.Wait()
}

//import (
//	"context"
//	"crypto/ecdsa"
//	"math/big"
//	"net/http"
//	"net/url"
//	"time"
//
//	"github.com/prometheus/client_golang/prometheus"
//	"github.com/prometheus/client_golang/prometheus/promhttp"
//	"github.com/sonm-io/core/accounts"
//	"github.com/sonm-io/core/blockchain"
//	"github.com/sonm-io/core/insonmnia/logging"
//	"github.com/sonm-io/core/proto"
//	"github.com/sonm-io/core/util"
//	"github.com/sonm-io/core/util/xgrpc"
//	"go.uber.org/zap"
//	"golang.org/x/sync/errgroup"
//)
//
//const (
//	Namespace    = "sonm"
//	ETHSubsystem = "eth"
//	DWHSubsystem = "dwh"
//	OrderId      = 1
//)
//
//type Config struct {
//	Account     *accounts.EthConfig
//	Blockchain  *blockchain.Config
//	DWHEndpoint string
//}
//
//type ethMetrics struct {
//	getOrderInfoOk        prometheus.Gauge
//	getOrderInfoHistogram prometheus.Histogram
//}
//
//func newETHMetrics() *ethMetrics {
//	err := prometheus.NewGauge(prometheus.GaugeOpts{
//		Namespace: Namespace,
//		Subsystem: ETHSubsystem,
//		Name:      "get_order_info_ok",
//		Help:      "SONM ETH GetOrderInfo OK",
//	})
//
//	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
//		Namespace: Namespace,
//		Subsystem: ETHSubsystem,
//		Name:      "get_order_info_response_time",
//		Help:      "SONM ETH GetOrderInfo response times",
//	})
//
//	prometheus.MustRegister(err, histogram)
//
//	m := &ethMetrics{
//		getOrderInfoOk:        err,
//		getOrderInfoHistogram: histogram,
//	}
//
//	return m
//}
//
//type dwhMetrics struct {
//	getOrdersOk        prometheus.Gauge
//	getOrdersHistogram prometheus.Histogram
//}
//
//func newDWHMetrics() *dwhMetrics {
//	err := prometheus.NewGauge(prometheus.GaugeOpts{
//		Namespace: Namespace,
//		Subsystem: DWHSubsystem,
//		Name:      "orders_ok",
//		Help:      "SONM DWH Orders OK",
//	})
//
//	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
//		Namespace: Namespace,
//		Subsystem: DWHSubsystem,
//		Name:      "orders_response_times",
//		Help:      "SONM DWH Orders response times",
//	})
//
//	prometheus.MustRegister(err, histogram)
//
//	m := &dwhMetrics{
//		getOrdersOk:        err,
//		getOrdersHistogram: histogram,
//	}
//
//	return m
//}
//
//type Collector struct {
//	market blockchain.MarketAPI
//	dwh    sonm.DWHClient
//
//	ethMetrics *ethMetrics
//	dwhMetrics *dwhMetrics
//
//	privateKey *ecdsa.PrivateKey
//	log        *zap.SugaredLogger
//}
//
//func NewCollector(cfg Config, log *zap.Logger) (*Collector, error) {
//	privateKey, err := cfg.Account.LoadKey(accounts.Silent())
//	if err != nil {
//		return nil, err
//	}
//
//	api, err := blockchain.NewAPI(blockchain.WithConfig(cfg.Blockchain))
//	if err != nil {
//		return nil, err
//	}
//
//	_, TLSConfig, err := util.NewHitlessCertRotator(context.Background(), privateKey)
//	if err != nil {
//		panic(err)
//	}
//
//	credentials := util.NewTLS(TLSConfig)
//
//	conn, err := xgrpc.NewClient(context.Background(), cfg.DWHEndpoint, credentials)
//	if err != nil {
//		return nil, err
//	}
//	dwh := sonm.NewDWHClient(conn)
//
//	m := &Collector{
//		market: api.Market(),
//		dwh:    dwh,
//
//		ethMetrics: newETHMetrics(),
//		dwhMetrics: newDWHMetrics(),
//
//		privateKey: privateKey,
//		log:        log.With(zap.String("type", "collector")).Sugar(),
//	}
//
//	return m, nil
//}
//
//func (m *Collector) Watch(ctx context.Context) error {
//	wg := errgroup.Group{}
//	wg.Go(func() error { return m.watch(ctx) })
//	return wg.Wait()
//}
//
//func (m *Collector) watch(ctx context.Context) error {
//	timer := time.NewTicker(1 * time.Second)
//	defer timer.Stop()
//
//	for {
//		select {
//		case <-ctx.Done():
//			return ctx.Err()
//		case <-timer.C:
//			m.log.Infof("collecting metrics")
//			go m.watchETHOrderInfo(ctx)
//			go m.watchDWHOrders(ctx)
//		}
//	}
//}
//
//func (m *Collector) watchETHOrderInfo(ctx context.Context) {
//	t := prometheus.NewTimer(m.ethMetrics.getOrderInfoHistogram)
//	defer t.ObserveDuration()
//
//	_, err := m.market.GetOrderInfo(ctx, big.NewInt(OrderId))
//	if err != nil {
//		m.ethMetrics.getOrderInfoOk.Set(0)
//		m.log.Warnw("failed ETH", zap.Error(err))
//	} else {
//		m.ethMetrics.getOrderInfoOk.Set(1)
//	}
//}
//
//func (m *Collector) watchDWHOrders(ctx context.Context) {
//	t := prometheus.NewTimer(m.dwhMetrics.getOrdersHistogram)
//	defer t.ObserveDuration()
//
//	_, err := m.dwh.GetOrders(ctx, &sonm.OrdersRequest{})
//	if err != nil {
//		m.dwhMetrics.getOrdersOk.Set(0)
//		m.log.Warnw("failed DWH", zap.Error(err))
//	} else {
//		m.dwhMetrics.getOrdersOk.Set(1)
//	}
//}
//
//func main() {
//	liveURL, err := url.Parse("https://rinkeby.infura.io/00iTrs5PIy0uGODwcsrb")
//	if err != nil {
//		panic(err)
//	}
//
//	sidechainURL, err := url.Parse("https://sidechain-dev.sonm.com/ping")
//	if err != nil {
//		panic(err)
//	}
//
//	cfg := Config{
//		Account: &accounts.EthConfig{
//			Keystore:   "./keys",
//			Passphrase: "any",
//		},
//		Blockchain: &blockchain.Config{
//			Endpoint:          *liveURL,
//			SidechainEndpoint: *sidechainURL,
//		},
//		DWHEndpoint: "8125721C2413d99a33E351e1F6Bb4e56b6b633FD@5.178.85.52:15021",
//	}
//
//	log := logging.BuildLogger(*logging.NewLevel(zap.DebugLevel))
//	collector, err := NewCollector(cfg, log)
//	if err != nil {
//		panic(err)
//	}
//
//	go collector.Watch(context.Background())
//
//	http.Handle("/metrics", promhttp.Handler())
//	http.ListenAndServe("[::1]:8090", nil)
//}
