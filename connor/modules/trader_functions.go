package modules

import (
	"math/big"
	"github.com/ethereum/go-ethereum/params"
	"github.com/pkg/errors"
	"fmt"
	"context"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/connor/records"
	"time"
)

func ClonePrice(def *big.Int) (*big.Int, error) {
	clone := big.NewInt(def.Int64())
	return big.NewInt(0).Set(clone), nil
}

func GetChangePercent(actualPriceForPack *big.Int, dealPrice *big.Int) (float64, error) {
	newClone, err := ClonePrice(actualPriceForPack)
	if err != nil {
		return 0, errors.Errorf("cannot get clone price ", err)
	}
	dealClone, err := ClonePrice(dealPrice)
	if err != nil {
		return 0, errors.Errorf("cannot get clone price ", err)
	}
	div := big.NewFloat(params.Ether)

	vNew := big.NewFloat(0).SetInt(newClone)
	rNew := big.NewFloat(0).Quo(vNew, div)

	vOld := big.NewFloat(0).SetInt(dealClone)
	rOld := big.NewFloat(0).Quo(vOld, div)

	fOld, _ := rOld.Float64()
	fnew, _ := rNew.Float64()
	delta := (fnew * 100) / fOld
	return delta, nil
}
func SaveActiveDealsIntoDB(ctx context.Context, dealCli sonm.DealManagementClient) error {
	getDeals, err := dealCli.List(ctx, &sonm.Count{Count: 100})
	if err != nil {
		fmt.Printf("Cannot get Deals list %v\r\n", err)
		return err
	}
	deals := getDeals.Deal
	if len(deals) > 0 {
		for _, deal := range deals {
			records.SaveDealIntoDB(&records.DealDb{
				DealID:       deal.GetId().Unwrap().Int64(),
				Status:       int32(deal.GetStatus()),
				Price:        deal.GetPrice().Unwrap().Int64(),
				AskId:        deal.GetAskID().Unwrap().Int64(),
				DeployStatus: 4,
				StartTime:    deal.GetStartTime().Unix(),
				LifeTime:     deal.GetEndTime().Unix(),
			})
		}
	} else {
		fmt.Printf("No active deals\r\n")
		time.Sleep(15 * time.Second)
	}
	return nil
}
func GetDeployedDeals() ([]int64, error) {
	dealsDB, err := records.GetDealsFromDB()
	if err != nil {
		return nil, fmt.Errorf("cannot create benchmarkes for symbol \"%s\"", err)
	}
	deployedDeals := make([]int64, 0)
	for _, d := range dealsDB {
		if d.DeployStatus == 3 { // Status :: deployed
			deal := d.DealID
			deployedDeals = append(deployedDeals, deal)
		}
	}
	return deployedDeals, nil
}

func GetBidBenchmarks(bidOrder *sonm.Order) (map[string]uint64, error) {
	getBench := bidOrder.GetBenchmarks()
	bMap := map[string]uint64{
		"ram-size":            getBench.RAMSize(),
		"cpu-cores":           getBench.CPUCores(),
		"cpu-sysbench-single": getBench.CPUSysbenchOne(),
		"cpu-sysbench-multi":  getBench.CPUSysbenchMulti(),
		"net-download":        getBench.NetTrafficIn(),
		"net-upload":          getBench.NetTrafficOut(),
		"gpu-count":           getBench.GPUCount(),
		"gpu-mem":             getBench.GPUMem(),
		"gpu-eth-hashrate":    getBench.GPUEthHashrate(), // H/s
	}
	return bMap, nil
}
func FloatToBigInt(val float64) (*big.Int) {
	price := val * params.Ether
	return big.NewInt(int64(price))
}

// Init benchmarks
func newBaseBenchmarks() map[string]uint64 {
	return map[string]uint64{
		"ram-size":            1000000,
		"cpu-cores":           1,
		"cpu-sysbench-single": 800,
		"cpu-sysbench-multi":  1000,
		"net-download":        1000,
		"net-upload":          1000,
		"gpu-count":           1,
		"gpu-mem":             4096000000,
	}
}
func newBenchmarksWithGPU(ethHashRate uint64) map[string]uint64 {
	m := newBaseBenchmarks()
	m["gpu-eth-hashrate"] = ethHashRate
	return m
}
func newBenchmarksWithoutGPU() map[string]uint64 {
	return newBaseBenchmarks()
}
func getBenchmarksForSymbol(symbol string, ethHashRate uint64) (map[string]uint64, error) {
	switch symbol {
	case "ETH":
		return newBenchmarksWithGPU(ethHashRate), nil
	case "ZEC":
		return newBenchmarksWithoutGPU(), nil
	case "XMR":
		return newBenchmarksWithGPU(ethHashRate), nil
	default:
		return nil, fmt.Errorf("cannot create benchmakes for symbol \"%s\"", symbol)
	}
}
func PriceToString(m *big.Int) string {
	v := big.NewFloat(0).SetInt(m)
	div := big.NewFloat(params.Ether)
	r := big.NewFloat(0).Quo(v, div)
	return r.Text('f', -18)
}
