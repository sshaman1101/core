package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sonm-io/core/blockchain"
)

const (
	hexKey = ""
)


func main() {

	p, err := loadSNMPriceUSD()
	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}

	price := divideSNM(p)

	log.Println(price)

	prv, err := crypto.HexToECDSA(hexKey)
	if err != nil {
		log.Fatalln(err)
		return
	}

	api, err := blockchain.NewAPI()
	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}

	tx, err := api.OracleUSD().SetCurrentPrice(context.TODO(), prv, price)
	if err != nil {
		log.Fatalln(err)
		os.Exit(1)
	}
	log.Println("txHash: ", tx.Hash().Hex())
}

func divideSNM(price float64) *big.Int {
	return big.NewInt(int64(1 / price * 1e18))
}

func loadSNMPriceUSD() (float64, error) {
	const url string = "https://api.coinmarketcap.com/v1/ticker/sonm/"
	body, err := getJson(url)
	if err != nil {
		return 0, err
	}
	var tickerSnm []*tokenData
	_ = json.Unmarshal(body, &tickerSnm)
	return strconv.ParseFloat(tickerSnm[0].PriceUsd, 64)
}

func getJson(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

type tokenData struct {
	PriceUsd string `json:"price_usd"`
}
