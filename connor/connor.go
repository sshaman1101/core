package connor

import (
	"github.com/sonm-io/core/util/xgrpc"
	"context"
	"crypto/ecdsa"
	"github.com/sonm-io/core/connor/config"
	"google.golang.org/grpc/credentials"
	"github.com/sonm-io/core/util"
	"log"
	"github.com/sonm-io/core/proto"
	"time"
	"github.com/ethereum/go-ethereum/crypto"
	"fmt"

	w "github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/connor/modules"
)

const (
	coinMarketCapTiker    = "https://api.coinmarketcap.com/v1/ticker/"
	coinMarketCapSnmTiker = coinMarketCapTiker + "sonm/"
	cryptoCompareCoinData = "https://www.cryptocompare.com/api/data/coinsnapshotfullbyid/?id="
	poolReportedHashRate  = "http://178.62.225.107:3000/v1/eth/reportedhashrates/"
	poolAverageHashRate   = "http://178.62.225.107:3000/v1/eth/avghashrateworkers/"
)

type Connor struct {
	key         *ecdsa.PrivateKey
	market      sonm.MarketClient
	taskClient  sonm.TaskManagementClient
	dealClient  sonm.DealManagementClient
	tokenClient sonm.TokenManagementClient

	cfg *config.Config
}

func NewConnor(ctx context.Context, key *ecdsa.PrivateKey, cfg *config.Config) (*Connor, error) {
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

	connor.market = sonm.NewMarketClient(nodeCC)
	connor.taskClient = sonm.NewTaskManagementClient(nodeCC)
	connor.dealClient = sonm.NewDealManagementClient(nodeCC)
	connor.tokenClient = sonm.NewTokenManagementClient(nodeCC)

	balanceReply, err := connor.tokenClient.Balance(ctx, &sonm.Empty{})
	if err != nil {
		return nil, fmt.Errorf("Cannot load balanceReply %v\r\n", err)
	}

	log.Printf(" >>>>> Balance 			   :: Live: %v, Side %v SNM \r\n", balanceReply.GetLiveBalance().Unwrap().String(), balanceReply.GetSideBalance().ToPriceString())
	log.Printf(" >>>>> Eth Node Address    :: %s\r\n", cfg.Market.Endpoint)
	log.Printf(" >>>>> Public Key address  :: %v\r\n", crypto.PubkeyToAddress(key.PublicKey).Hex())

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

	snm := w.NewSNMPriceWatcher(coinMarketCapSnmTiker)
	token := w.NewTokenPriceWatcher(coinMarketCapTiker, cryptoCompareCoinData)
	reportedPool := w.NewPoolWatcher(poolReportedHashRate, []string{c.cfg.PoolAddress.EthPoolAddr})
	avgPool := w.NewPoolWatcher(poolAverageHashRate, []string{c.cfg.PoolAddress.EthPoolAddr + "/1"})

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

	ethAddr := &sonm.EthAddress{Address: crypto.PubkeyToAddress(c.key.PublicKey).Bytes()}

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
			go modules.CollectTokensMiningProfit(token)

		case <-tradeUpdate.C:
			modules.TradeObserve(ctx, ethAddr, c.dealClient, reportedPool, token, c.market, c.taskClient, c.cfg)
		case <-poolInit.C:
			modules.SavePoolDataToDb(ctx, reportedPool, c.cfg.PoolAddress.EthPoolAddr)
		case <-poolTrack.C:
			modules.PoolTrack(ctx, reportedPool, avgPool, c.cfg.PoolAddress.EthPoolAddr, c.dealClient, c.market)
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
