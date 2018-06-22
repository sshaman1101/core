package modules

import (
	"fmt"
	rec "github.com/sonm-io/core/connor/records"
	"github.com/sonm-io/core/connor/watchers"
	"log"
	"time"
)

const (
	hashingPower     = 1
	costPerkWh       = 0.0
	powerConsumption = 0.0
)

type powerAndDivider struct {
	power float64
	div   float64
}

func getHashPowerAndDividerForToken(s string, hp float64) (float64, float64, bool) {
	var tokenHashPower = map[string]powerAndDivider{
		"ETH": {div: 1, power: hashingPower * 1000000.0},
		"XMR": {div: 1, power: hp / 1000000.0},
		"ZEC": {div: 1, power: hp / 1000.0},
	}
	p, ok := tokenHashPower[s]
	if !ok {
		return .0, .0, false
	}
	return p.power, p.div, true
}

type TokenMainData struct {
	Symbol            string
	ProfitPerDaySnm   float64
	ProfitPerMonthSnm float64
	ProfitPerMonthUsd float64
}

func getTokensForProfitCalculation() []*TokenMainData {
	return []*TokenMainData{
		{Symbol: "ETH"},
		{Symbol: "XMR"},
		{Symbol: "ZEC"},
	}
}
func CollectTokensMiningProfit(t watchers.TokenWatcher) ([]*TokenMainData, error) {
	var tokensForCalc = getTokensForProfitCalculation()
	for _, token := range tokensForCalc {
		tokenData, err := t.GetTokenData(token.Symbol)
		if err != nil {
			log.Printf("cannot get token data %v\r\n", err)
		}
		hashesPerSecond, divider, ok := getHashPowerAndDividerForToken(tokenData.Symbol, tokenData.NetHashPerSec)
		if !ok {
			log.Printf("DEBUG :: cannot process tokenData %s, not in list\r\n", tokenData.Symbol)
			continue
		}
		token.ProfitPerMonthUsd = calculateMiningProfit(tokenData.PriceUSD, hashesPerSecond, tokenData.NetHashPerSec, tokenData.BlockReward, divider, tokenData.BlockTime)

		if token.Symbol == "ETH" {
			rec.SaveProfitToken(&rec.TokenDb{
				ID:              tokenData.CmcID,
				Name:            token.Symbol,
				UsdPrice:        tokenData.PriceUSD,
				NetHashesPerSec: tokenData.NetHashPerSec,
				BlockTime:       tokenData.BlockTime,
				BlockReward:     tokenData.BlockReward,
				ProfitPerMonth:  token.ProfitPerMonthUsd,
				DateTime:        time.Now(),
			})
		}
	}
	return tokensForCalc, nil
}
func calculateMiningProfit(usd, hashesPerSecond, netHashesPerSecond, blockReward, div float64, blockTime int) float64 {
	currentHashingPower := hashesPerSecond / div
	miningShare := currentHashingPower / (netHashesPerSecond + currentHashingPower)

	minedPerDay := miningShare * 86400 / float64(blockTime) * (blockReward / div)

	powerCostPerDayUSD := (powerConsumption * 24) / 1000 * costPerkWh
	returnPerDayUSD := (usd*minedPerDay - (usd * minedPerDay * 0.01)) - powerCostPerDayUSD

	//profitRatio := (returnPerDayUSD / (powerCostPerDayUSD)) / 10
	perMonthUSD := float64(returnPerDayUSD * 30)
	//marginPrice := perMonthUSD + (perMonthUSD * 0.05)
	return perMonthUSD
}

func GetProfitForTokenBySymbol(tokens []*TokenMainData, symbol string) (float64, error) {
	for _, t := range tokens {
		if t.Symbol == symbol {
			return t.ProfitPerMonthUsd, nil
		}
	}
	return 0, fmt.Errorf("Cannot get price from token! ")
}
