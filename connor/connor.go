package connor

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"google.golang.org/grpc/credentials"

	"github.com/noxiouz/zapctx/ctxlog"
	"github.com/sonm-io/core/connor/database"
	"github.com/sonm-io/core/connor/watchers"
	"go.uber.org/zap"
)

const (
	coinMarketCapTicker     = "https://api.coinmarketcap.com/v1/ticker/"
	coinMarketCapSonmTicker = coinMarketCapTicker + "sonm/"
	cryptoCompareCoinData   = "https://www.cryptocompare.com/api/data/coinsnapshotfullbyid/?id="
	poolReportedHashRate    = "https://api.nanopool.org/v1/eth/reportedhashrates/"
	poolAverageHashRate     = "https://api.nanopool.org/v1/eth/avghashrateworkers/"
)

type Connor struct {
	key          *ecdsa.PrivateKey
	Market       sonm.MarketClient
	TaskClient   sonm.TaskManagementClient
	DealClient   sonm.DealManagementClient
	TokenClient  sonm.TokenManagementClient
	MasterClient sonm.MasterManagementClient

	cfg    *Config
	db     *database.Database
	logger *zap.Logger
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
	connor.MasterClient = sonm.NewMasterManagementClient(nodeCC)

	connor.db, err = database.NewDatabaseConnect(connor.cfg.Database.Driver, connor.cfg.Database.DataSource)
	if err != nil {
		return nil, err
	}

	balanceReply, err := connor.TokenClient.Balance(ctx, &sonm.Empty{})
	if err != nil {
		connor.logger.Error("cannot load balanceReply", zap.Error(err))
		return nil, err
	}

	connor.logger = ctxlog.GetLogger(ctx)
	connor.logger.Info("Config",
		zap.String("Eth Node address", cfg.Market.Endpoint),
		zap.String("key", crypto.PubkeyToAddress(key.PublicKey).String()))
	connor.logger.Info("Balance",
		zap.String("live", balanceReply.GetLiveBalance().Unwrap().String()),
		zap.String("Side", balanceReply.GetSideBalance().ToPriceString()))
	connor.logger.Info("configuring Connor", zap.Any("config", cfg))
	return connor, nil
}

func (c *Connor) Serve(ctx context.Context) error {
	c.logger.Info("Connor started work ...")
	defer c.logger.Info("Connor has been stopped")

	dataUpdate := time.NewTicker(10 * time.Second)
	defer dataUpdate.Stop()
	tradeUpdate := time.NewTicker(15 * time.Second)
	defer tradeUpdate.Stop()
	poolInit := time.NewTicker(900 * time.Second)
	defer poolInit.Stop()

	snm := watchers.NewSNMPriceWatcher(coinMarketCapSonmTicker)
	token := watchers.NewTokenPriceWatcher(coinMarketCapTicker, cryptoCompareCoinData)
	reportedPool := watchers.NewPoolWatcher(poolReportedHashRate, []string{c.cfg.PoolAddress.EthPoolAddr})
	avgPool := watchers.NewPoolWatcher(poolAverageHashRate, []string{c.cfg.PoolAddress.EthPoolAddr + "/1"})

	if err := snm.Update(ctx); err != nil {
		return fmt.Errorf("cannot update snm data: %v", err)
	}
	if err := token.Update(ctx); err != nil {
		return fmt.Errorf("cannot update token data: %v", err)
	}
	if err := reportedPool.Update(ctx); err != nil {
		return fmt.Errorf("cannot update reportedPool data: %v", err)
	}
	if err := avgPool.Update(ctx); err != nil {
		return fmt.Errorf("cannot update avgPool data: %v", err)
	}

	profitModule := NewProfitableModules(c)
	poolModule := NewPoolModules(c)
	traderModule := NewTraderModules(c, poolModule, profitModule)
	balanceReply, err := c.TokenClient.Balance(ctx, &sonm.Empty{})
	if err != nil {
		return err
	}
	go traderModule.ChargeOrdersOnce(ctx, c.cfg.UsingToken.Token, token, snm, balanceReply)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context done %v", ctx.Err())
		case <-dataUpdate.C:
			if err := snm.Update(ctx); err != nil {
				return fmt.Errorf(" cannot update SNM data: %v\n", err)
			}
			if err := token.Update(ctx); err != nil {
				return fmt.Errorf("cannot update TOKEN data: %v\n", err)
			}
			if err := reportedPool.Update(ctx); err != nil {
				return fmt.Errorf("cannot update reported pool data: %v\n", err)
			}
			if err := avgPool.Update(ctx); err != nil {
				return fmt.Errorf("cannot update avg pool data: %v\n", err)
			}
			go profitModule.CollectTokensMiningProfit(token)
		case <-tradeUpdate.C:
			err := traderModule.SaveActiveDealsIntoDB(ctx, c.DealClient)
			if err != nil {
				return fmt.Errorf("cannot save active deals : %v\n", err)
			}

			_, pricePerSec, err := traderModule.GetPriceForTokenPerSec(token, c.cfg.UsingToken.Token)
			if err != nil {
				return fmt.Errorf("cannot get pricePerSec for token per sec %v\r\n", err)
			}

			actualPrice := traderModule.FloatToBigInt(pricePerSec)
			deals, err := traderModule.c.db.GetDealsFromDB()
			if err != nil {
				return fmt.Errorf("cannot get deals from DB %v\r\n", err)
			}
			if len(deals) > 0 {
				err := traderModule.DealsProfitTracking(ctx, actualPrice, deals, c.cfg.Images.Image)
				if err != nil {
					return fmt.Errorf("cannot start deals profit tracking: %v", err)
				}
			}
			orders, err := traderModule.c.db.GetOrdersFromDB()
			if err != nil {
				return fmt.Errorf("cannot get orders from DB %v\r\n", err)
			}
			if len(orders) > 0 {
				err := traderModule.OrdersProfitTracking(ctx, c.cfg, actualPrice, orders)
				if err != nil {
					return fmt.Errorf("cannot start orders profit tracking: %v", err)
				}
			}
		case <-poolInit.C:
			dealsDb, err := traderModule.c.db.GetDealsFromDB()
			if err != nil {
				return fmt.Errorf("cannot get deals from DB %v\r\n", err)
			}
			for _, dealDb := range dealsDb {
				if dealDb.DeployStatus == int64(DeployStatusDEPLOYED) {
					dealOnMarket, err := c.DealClient.Status(ctx, sonm.NewBigIntFromInt(dealDb.DealID))
					if err != nil {
						return fmt.Errorf("cannot get deal from market %v", dealDb.DealID)
					}
					if err := poolModule.AddWorkerToPoolDB(ctx, dealOnMarket, c.cfg.PoolAddress.EthPoolAddr); err != nil {
						return fmt.Errorf("cannot add worker to Db : %v", err)
					}
				}
			}
			if err := poolModule.DefaultPoolHashrateTracking(ctx, reportedPool, avgPool); err != nil {
				return err
			}
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
