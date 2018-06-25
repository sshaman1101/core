package connor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"google.golang.org/grpc/credentials"
	"time"

	"github.com/noxiouz/zapctx/ctxlog"
	"github.com/sonm-io/core/connor/database"
	"github.com/sonm-io/core/connor/watchers"
	"go.uber.org/zap"
	"os"
)

const (
	coinMarketCapTicker     = "https://api.coinmarketcap.com/v1/ticker/"
	coinMarketCapSonmTicker = coinMarketCapTicker + "sonm/"
	cryptoCompareCoinData   = "https://www.cryptocompare.com/api/data/coinsnapshotfullbyid/?id="
	poolReportedHashRate    = "http://178.62.225.107:3000/v1/eth/reportedhashrates/"
	poolAverageHashRate     = "http://178.62.225.107:3000/v1/eth/avghashrateworkers/"
)

const (
	driver     = "sqlite3"
	dataSource = "./connor/tests/test.sq3"
)

type Connor struct {
	key         *ecdsa.PrivateKey
	Market      sonm.MarketClient
	TaskClient  sonm.TaskManagementClient
	DealClient  sonm.DealManagementClient
	TokenClient sonm.TokenManagementClient

	cfg *Config
	db  *database.Database
}

func NewConnor(ctx context.Context, key *ecdsa.PrivateKey, cfg *Config) (*Connor, error) {
	connor := &Connor{
		key: key,
		cfg: cfg,
	}

	creds, err := newCredentials(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("can't create TLS credentials: %v", err)
	}

	nodeCC, err := xgrpc.NewClient(ctx, cfg.Market.Endpoint, creds)
	if err != nil {
		return nil, fmt.Errorf("can't create node connection: %v\r\n", err)
	}

	connor.Market = sonm.NewMarketClient(nodeCC)
	connor.TaskClient = sonm.NewTaskManagementClient(nodeCC)
	connor.DealClient = sonm.NewDealManagementClient(nodeCC)
	connor.TokenClient = sonm.NewTokenManagementClient(nodeCC)

	connor.db, err = database.NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return nil, err
	}

	balanceReply, err := connor.TokenClient.Balance(ctx, &sonm.Empty{})
	if err != nil {
		return nil, fmt.Errorf("Cannot load balanceReply %v\r\n", err)
	}

	logger := ctxlog.GetLogger(ctx)

	logger.Info("Config",
		zap.String("Eth Node address", cfg.Market.Endpoint),
		zap.String("key", crypto.PubkeyToAddress(key.PublicKey).String()))
	logger.Info("Balance",
		zap.String("live", balanceReply.GetLiveBalance().Unwrap().String()),
		zap.String("Side", balanceReply.GetSideBalance().ToPriceString()))

	return connor, nil
}

func (c *Connor) Serve(ctx context.Context) error {
	var err error

	dataUpdate := time.NewTicker(3 * time.Second)
	defer dataUpdate.Stop()
	tradeUpdate := time.NewTicker(3 * time.Second)
	defer tradeUpdate.Stop()
	poolTrack := time.NewTicker(900 * time.Second)
	defer poolTrack.Stop()
	poolInit := time.NewTimer(900 * time.Second)
	defer poolInit.Stop()

	snm := watchers.NewSNMPriceWatcher(coinMarketCapSonmTicker)
	token := watchers.NewTokenPriceWatcher(coinMarketCapTicker, cryptoCompareCoinData)
	reportedPool := watchers.NewPoolWatcher(poolReportedHashRate, []string{c.cfg.PoolAddress.EthPoolAddr})
	avgPool := watchers.NewPoolWatcher(poolAverageHashRate, []string{c.cfg.PoolAddress.EthPoolAddr + "/1"})

	if err = snm.Update(ctx); err != nil {
		return fmt.Errorf("cannot update snm data: %v", err)
	}
	if err = token.Update(ctx); err != nil {
		return fmt.Errorf("cannot update token data: %v", err)
	}
	if err = reportedPool.Update(ctx); err != nil {
		return fmt.Errorf("cannot update reportedPool data: %v", err)
	}
	if err = avgPool.Update(ctx); err != nil {
		return fmt.Errorf("cannot update avgPool data: %v", err)
	}

	profitModule := NewProfitableModules(c)
	poolModule := NewPoolModules(c)
	traderModule := NewTraderModules(c, poolModule, profitModule)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context done")
		case <-dataUpdate.C:
			if err = snm.Update(ctx); err != nil {
				return fmt.Errorf(" cannot update SNM data: %v\n", err)
			}
			if err = token.Update(ctx); err != nil {
				return fmt.Errorf("cannot update TOKEN data: %v\n", err)
			}
			go profitModule.CollectTokensMiningProfit(token)
		case <-tradeUpdate.C:
			traderModule.TradeObserve(ctx, reportedPool, token, c.cfg)
		case <-poolInit.C:
			poolModule.SavePoolDataToDb(ctx, reportedPool, c.cfg.PoolAddress.EthPoolAddr)
		case <-poolTrack.C:
			poolModule.PoolHashrateTracking(ctx, reportedPool, avgPool, c.cfg.PoolAddress.EthPoolAddr)
		}
	}
}

func newCredentials(ctx context.Context, key *ecdsa.PrivateKey) (credentials.TransportCredentials, error) {
	_, TLSConfig, err := util.NewHitlessCertRotator(ctx, key)
	if err != nil {
		return nil, err
	}
	return util.NewTLS(TLSConfig), nil
}
