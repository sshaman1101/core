package main

import (
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/connor/config"
	"context"
	"fmt"
	"os"
	"log"
)

const configPath = "insonmnia/arbitrageBot/bot.yaml"

func main() {
	var err error
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		fmt.Printf("cannot load config file: %s\r\n", err)
		os.Exit(1)
	}

	PoolWatcher := watchers.NewPoolWatcher("https://api.nanopool.org/v1/eth/user/", []string{cfg.PoolAddress.EthPoolAddr})
	err = PoolWatcher.Update(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Printf("EthAddr = %v\r\n", cfg.PoolAddress.EthPoolAddr)
	data, err := PoolWatcher.GetData(cfg.PoolAddress.EthPoolAddr)
	if err != nil {
		fmt.Printf("cannot get data from Pool %v\r\n", err)
	}
	zecPoolWatcher := watchers.NewPoolWatcher("https://api.nanopool.org/v1/zec/user/", []string{cfg.PoolZecConfig()})
	err = zecPoolWatcher.Update(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Printf("ZecAddr = %v\r\n", cfg.PoolZecConfig())
	for _, w := range zecPoolWatcher.GetData(cfg.PoolZecConfig()).Data.Workers {
		fmt.Printf("Workers => ID: %s, HashRate: %s\r\n", w.ID, w.Hashrate)
	}

	xmrPoolWatcher := watchers.NewPoolWatcher("https://api.nanopool.org/v1/xmr/user/", []string{cfg.PoolXmrConfig()})
	err = xmrPoolWatcher.Update(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	average := xmrPoolWatcher.GetData(cfg.PoolXmrConfig()).Data.AvgHashrate.H1
	log.Printf("Pool => XmrAddr = %v, average = %v\r\n", cfg.PoolXmrConfig(), average)


	snmPriceWatcher := watchers.NewSNMPriceWatcher("https://api.coinmarketcap.com/v1/ticker/sonm/")
	err = snmPriceWatcher.Update(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Printf("SNM price => %f\r\n", snmPriceWatcher.GetPrice())

	t := watchers.NewTokenPriceWatcher(
		"https://api.coinmarketcap.com/v1/ticker/",
		"https://www.cryptocompare.com/api/data/coinsnapshotfullbyid/?id=")
	err = t.Update(context.Background())
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var (
		eth = t.GetTokenData("ETH"); xmr = t.GetTokenData("XMR");zec = t.GetTokenData("ZEC")
	)
	log.Printf("Id: %s, Symbol: %s, BlockTime: %v, BlockReward: %v, Hash: %v, Price usd: %f\r\n",
		eth.CmcID, eth.Symbol, eth.BlockTime, eth.BlockReward, eth.NetHashPerSec, eth.PriceUSD)
	log.Printf("Id: %s, Symbol: %s, BlockTime: %v, BlockReward: %v, Hash: %v, Price usd: %f\r\n",
		xmr.CmcID, xmr.Symbol, xmr.BlockTime, xmr.BlockReward, xmr.NetHashPerSec, xmr.PriceUSD)
	log.Printf("Id: %s, Symbol: %s, BlockTime: %v, BlockReward: %v, Hash: %v, Price usd: %f\r\n",
		zec.CmcID, zec.Symbol, zec.BlockTime, zec.BlockReward, zec.NetHashPerSec, zec.PriceUSD)
}
