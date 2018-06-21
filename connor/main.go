package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"os"
	"time"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonm-io/core/connor/config"
	"github.com/sonm-io/core/connor/modules"
	w "github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const configPath = "insonmnia/arbBot/bot.yaml"

func main() {
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		fmt.Printf("cannot load config file: %s\r\n", err)
		os.Exit(1)
	}

	key, err := cfg.Eth.LoadKey()
	if err != nil {
		fmt.Printf("cannot open keys: %v\r\n", err)
		os.Exit(1)
	}
	ctx := context.Background()

	creds, err := newCredentials(ctx, key)
	if err != nil {
		fmt.Printf("can't create TLS credentials: %v\r\n", err)
		os.Exit(1)
	}

	nodeCC, err := newNodeClientConn(ctx, creds, cfg.Market.Endpoint)
	if err != nil {
		fmt.Printf("can't create node connection: %v\r\n", err)
		os.Exit(1)
	}

	marketClient := sonm.NewMarketClient(nodeCC)
	taskCli := sonm.NewTaskManagementClient(nodeCC)
	dealCli := sonm.NewDealManagementClient(nodeCC)
	tokenCli := sonm.NewTokenManagementClient(nodeCC)

	balanceReply, err := tokenCli.Balance(ctx, &sonm.Empty{})
	if err != nil {
		fmt.Printf("Cannot load balanceReply %v\r\n", err)
		os.Exit(1)
	}

	fmt.Printf(" >>>>> Balance 			   :: Live: %v, Side %v SNM \r\n", balanceReply.GetLiveBalance().Unwrap().String(), balanceReply.GetSideBalance().ToPriceString())
	fmt.Printf(" >>>>> Eth Node Address    :: %s\r\n", cfg.Market.Endpoint)
	fmt.Printf(" >>>>> Public Key address  :: %v\r\n", crypto.PubkeyToAddress(key.PublicKey).Hex())

	ethAddr := &sonm.EthAddress{Address: crypto.PubkeyToAddress(key.PublicKey).Bytes()}
	identityLVl := GetIdentityLvl(cfg)

	dataUpdate := time.NewTicker(3 * time.Second)
	defer dataUpdate.Stop()
	tradeUpdate := time.NewTicker(3 * time.Second)
	defer tradeUpdate.Stop()
	poolTrack := time.NewTicker(900 * time.Second) //900
	defer poolTrack.Stop()
	poolInit := time.NewTimer(900 * time.Second)
	defer poolInit.Stop()

	snm := w.NewSNMPriceWatcher("https://api.coinmarketcap.com/v1/ticker/sonm/")
	token := w.NewTokenPriceWatcher("https://api.coinmarketcap.com/v1/ticker/", "https://www.cryptocompare.com/api/data/coinsnapshotfullbyid/?id=")
	reportedPool := w.NewPoolWatcher("http://178.62.225.107:3000/v1/eth/reportedhashrates/", []string{cfg.PoolAddress.EthPoolAddr})
	avgPool := w.NewPoolWatcher("http://178.62.225.107:3000/v1/eth/avghashrateworkers/", []string{cfg.PoolAddress.EthPoolAddr + "/1"})
	if err = snm.Update(ctx); err != nil {
		log.Printf("cannot update snm data: %v\n", err)
	}
	if err = token.Update(ctx); err != nil {
		log.Printf("cannot update token data: %v\n", err)
	}
	if err = reportedPool.Update(ctx); err != nil {
		log.Printf("cannot update reportedPool data: %v\n", err)
	}
	if err = avgPool.Update(ctx); err != nil {
		log.Printf("cannot update avgPool data: %v\n", err)
	}

	modules.ChargeOrdersOnce(ctx, marketClient, token, snm, balanceReply, cfg, ethAddr, identityLVl)

	for {
		select {
		case <-ctx.Done():
			log.Println("context done")
			return

		case <-dataUpdate.C:
			if err = snm.Update(ctx); err != nil {
				log.Printf(" cannot update SNM data: %v\n", err)
			}
			if err = token.Update(ctx); err != nil {
				log.Printf("cannot update TOKEN data: %v\n", err)
			}
			go modules.CollectTokensMiningProfit(token)

		case <-tradeUpdate.C:
			modules.TradeObserve(ctx, ethAddr, dealCli, reportedPool, token, marketClient, taskCli, cfg, identityLVl)
		case <-poolInit.C:
			modules.SavePoolDataToDb(ctx, reportedPool, cfg.PoolAddress.EthPoolAddr)
		case <-poolTrack.C:
			modules.PoolTrack(ctx, reportedPool, avgPool, cfg.PoolAddress.EthPoolAddr, dealCli, marketClient)
		}
	}
}

func GetIdentityLvl(cfg *config.Config) (sonm.IdentityLevel) {
	switch cfg.OtherParameters.IdentityForBid {
	case 1:
		return sonm.IdentityLevel_ANONYMOUS
	case 2:
		return sonm.IdentityLevel_REGISTERED
	case 3:
		return sonm.IdentityLevel_IDENTIFIED
	case 4:
		return sonm.IdentityLevel_PROFESSIONAL
	}
	return sonm.IdentityLevel_UNKNOWN
}
func newCredentials(ctx context.Context, key *ecdsa.PrivateKey) (credentials.TransportCredentials, error) {
	_, TLSConfig, err := util.NewHitlessCertRotator(ctx, key)
	if err != nil {
		return nil, err
	}
	return util.NewTLS(TLSConfig), nil
}
func newNodeClientConn(ctx context.Context, creds credentials.TransportCredentials, endpoint string) (*grpc.ClientConn, error) {
	cc, err := xgrpc.NewClient(ctx, endpoint, creds)
	if err != nil {
		return nil, err
	}
	return cc, nil
}
