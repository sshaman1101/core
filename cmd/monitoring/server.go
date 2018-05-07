package main

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/blockchain"
	"github.com/sonm-io/core/insonmnia/logging"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	Namespace    = "sonm"
	ETHSubsystem = "eth"
	DWHSubsystem = "dwh"
	OrderId      = 1
)

type Config struct {
	Account     *accounts.EthConfig
	Blockchain  *blockchain.Config
	DWHEndpoint string
}

type ethMetrics struct {
	getOrderInfoOk        prometheus.Gauge
	getOrderInfoHistogram prometheus.Histogram
}

func newETHMetrics() *ethMetrics {
	err := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: ETHSubsystem,
		Name:      "get_order_info_ok",
		Help:      "SONM ETH GetOrderInfo OK",
	})

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: ETHSubsystem,
		Name:      "get_order_info_response_time",
		Help:      "SONM ETH GetOrderInfo response times",
	})

	prometheus.MustRegister(err, histogram)

	m := &ethMetrics{
		getOrderInfoOk:        err,
		getOrderInfoHistogram: histogram,
	}

	return m
}

type dwhMetrics struct {
	getOrdersOk        prometheus.Gauge
	getOrdersHistogram prometheus.Histogram
}

func newDWHMetrics() *dwhMetrics {
	err := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: Namespace,
		Subsystem: DWHSubsystem,
		Name:      "orders_ok",
		Help:      "SONM DWH Orders OK",
	})

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: Namespace,
		Subsystem: DWHSubsystem,
		Name:      "orders_response_times",
		Help:      "SONM DWH Orders response times",
	})

	prometheus.MustRegister(err, histogram)

	m := &dwhMetrics{
		getOrdersOk:        err,
		getOrdersHistogram: histogram,
	}

	return m
}

type Collector struct {
	market blockchain.MarketAPI
	dwh    sonm.DWHClient

	ethMetrics *ethMetrics
	dwhMetrics *dwhMetrics

	privateKey *ecdsa.PrivateKey
	log        *zap.SugaredLogger
}

func NewCollector(cfg Config, log *zap.Logger) (*Collector, error) {
	privateKey, err := cfg.Account.LoadKey(accounts.Silent())
	if err != nil {
		return nil, err
	}

	api, err := blockchain.NewAPI(blockchain.WithConfig(cfg.Blockchain))
	if err != nil {
		return nil, err
	}

	_, TLSConfig, err := util.NewHitlessCertRotator(context.Background(), privateKey)
	if err != nil {
		panic(err)
	}

	credentials := util.NewTLS(TLSConfig)

	conn, err := xgrpc.NewClient(context.Background(), cfg.DWHEndpoint, credentials)
	if err != nil {
		return nil, err
	}
	dwh := sonm.NewDWHClient(conn)

	m := &Collector{
		market: api.Market(),
		dwh:    dwh,

		ethMetrics: newETHMetrics(),
		dwhMetrics: newDWHMetrics(),

		privateKey: privateKey,
		log:        log.With(zap.String("type", "collector")).Sugar(),
	}

	return m, nil
}

func (m *Collector) Watch(ctx context.Context) error {
	wg := errgroup.Group{}
	wg.Go(func() error { return m.watch(ctx) })
	return wg.Wait()
}

func (m *Collector) watch(ctx context.Context) error {
	timer := time.NewTicker(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			m.log.Infof("collecting metrics")
			go m.watchETHOrderInfo(ctx)
			go m.watchDWHOrders(ctx)
		}
	}
}

func (m *Collector) watchETHOrderInfo(ctx context.Context) {
	t := prometheus.NewTimer(m.ethMetrics.getOrderInfoHistogram)
	defer t.ObserveDuration()

	_, err := m.market.GetOrderInfo(ctx, big.NewInt(OrderId))
	if err != nil {
		m.ethMetrics.getOrderInfoOk.Set(0)
		m.log.Warnw("failed ETH", zap.Error(err))
	} else {
		m.ethMetrics.getOrderInfoOk.Set(1)
	}
}

func (m *Collector) watchDWHOrders(ctx context.Context) {
	t := prometheus.NewTimer(m.dwhMetrics.getOrdersHistogram)
	defer t.ObserveDuration()

	_, err := m.dwh.GetOrders(ctx, &sonm.OrdersRequest{})
	if err != nil {
		m.dwhMetrics.getOrdersOk.Set(0)
		m.log.Warnw("failed DWH", zap.Error(err))
	} else {
		m.dwhMetrics.getOrdersOk.Set(1)
	}
}

func main() {
	liveURL, err := url.Parse("https://rinkeby.infura.io/00iTrs5PIy0uGODwcsrb")
	if err != nil {
		panic(err)
	}

	sidechainURL, err := url.Parse("https://sidechain-dev.sonm.com/ping")
	if err != nil {
		panic(err)
	}

	cfg := Config{
		Account: &accounts.EthConfig{
			Keystore:   "./keys",
			Passphrase: "any",
		},
		Blockchain: &blockchain.Config{
			Endpoint:          *liveURL,
			SidechainEndpoint: *sidechainURL,
		},
		DWHEndpoint: "8125721C2413d99a33E351e1F6Bb4e56b6b633FD@5.178.85.52:15021",
	}

	log := logging.BuildLogger(*logging.NewLevel(zap.DebugLevel))
	collector, err := NewCollector(cfg, log)
	if err != nil {
		panic(err)
	}

	go collector.Watch(context.Background())

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe("[::1]:8090", nil)
}
