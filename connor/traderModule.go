package connor

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/params"
	"github.com/sonm-io/core/connor/database"
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
	"go.uber.org/zap"
)

const (
	hashes       = 1000000
	daysPerMonth = 30
	secsPerDay   = 86400

	hashingPower     = 1
	costPerkWh       = 0.0
	powerConsumption = 0.0
)

type TraderModule struct {
	c      *Connor
	pool   *PoolModule
	profit *ProfitableModule
}

func NewTraderModules(c *Connor, pool *PoolModule, profit *ProfitableModule) *TraderModule {
	return &TraderModule{
		c:      c,
		pool:   pool,
		profit: profit,
	}
}

type DeployStatus int32

const (
	DeployStatusDEPLOYED    DeployStatus = 3
	DeployStatusNOTDEPLOYED DeployStatus = 4
)

type OrderStatus int32

const (
	OrderStatusCANCELLED OrderStatus = 3
	OrderStatusREINVOICE OrderStatus = 4
)

func (t *TraderModule) getTokenConfiguration(symbol string, cfg *Config) (float64, float64, float64) {
	switch symbol {
	case "ETH":
		return cfg.ChargeIntervalETH.Start, cfg.ChargeIntervalETH.Destination, cfg.Distances.StepForETH
	case "ZEC":
		return cfg.ChargeIntervalZEC.Start, cfg.ChargeIntervalZEC.Destination, cfg.Distances.StepForZEC
	case "XMR":
		return cfg.ChargeIntervalXMR.Start, cfg.ChargeIntervalXMR.Destination, cfg.Distances.StepForXMR
	}
	return 0, 0, 0
}

func (t *TraderModule) ChargeOrdersOnce(ctx context.Context, symbol string, token watchers.TokenWatcher, snm watchers.PriceWatcher, balanceReply *sonm.BalanceReply) error {
	t.c.db.CreateOrderDB()
	start, destination, step := t.getTokenConfiguration(symbol, t.c.cfg)

	count, err := t.c.db.GetCountFromDB()
	if err != nil {
		log.Printf("Cannot get count from DB: %v", err)
		return err
	}
	if count == 0 {
		log.Printf("Save TEST order cause DB is empty! \r\n")
		if err := t.c.db.SaveOrderIntoDB(&database.OrderDb{
			OrderID:         0,
			Price:           0,
			Hashrate:        0,
			StartTime:       time.Time{},
			ButterflyEffect: 2,
			ActualStep:      start,
		}); err != nil {
			fmt.Printf("Cannot save order into DB %v\r\n", err)
		}
	}

	pricePerMonthUSD, pricePerSecMh, err := t.GetPriceForTokenPerSec(token, symbol)
	if err != nil {
		t.c.logger.Error("cannot get profit for tokens", zap.Error(err))
		return err
	}
	limitChargeInSNM := t.profit.LimitChargeSNM(balanceReply.GetSideBalance().Unwrap(), t.c.cfg.Sensitivity.PartCharge)
	limitChargeInSNMClone := big.NewInt(0).Set(limitChargeInSNM)
	limitChargeInUSD := t.profit.ConvertingToUSDBalance(limitChargeInSNMClone, snm.GetPrice())

	mhashForToken, err := t.c.db.GetLastActualStepFromDb()
	if err != nil {
		t.c.logger.Error("cannot get last actual step from DB", zap.Error(err))
		return err
	}

	pricePackMhInUSDPerMonth := mhashForToken * (pricePerMonthUSD * t.c.cfg.Sensitivity.MarginAccounting)
	sumOrdersPerMonth := limitChargeInUSD / pricePackMhInUSDPerMonth
	if limitChargeInSNM.Int64() <= 0 {
		t.c.logger.Error("balance SNM is not enough for create orders!", zap.Error(err))
		return err
	}

	t.c.logger.Info("i", zap.String("CHARGE ORDERS ONCE: ", symbol),
		zap.String("Limit for Charge SNM :", sonm.NewBigInt(limitChargeInSNM).ToPriceString()),
		zap.Float64("Limit for Charge USD :", limitChargeInUSD),
		zap.Float64("Pack Price per month USD", pricePackMhInUSDPerMonth),
		zap.Int64("Sum orders per month", int64(sumOrdersPerMonth)))

	for i := 0; i < int(sumOrdersPerMonth); i++ {
		if mhashForToken >= destination {
			t.c.logger.Info("charge is finished cause reached the limit", zap.Float64("charge destination", t.c.cfg.ChargeIntervalETH.Destination))
			break
		}
		pricePerSecPack := t.FloatToBigInt(mhashForToken * pricePerSecMh)
		zap.String("Price", sonm.NewBigInt(pricePerSecPack).ToPriceString())
		mhashForToken, err = t.ChargeOrders(ctx, symbol, pricePerSecPack, step, mhashForToken)
		if err != nil {
			return fmt.Errorf("cannot charging market! %v\r\n", err)
		}
	}
	return nil
}

// Prepare price and Map depends on token symbol. Create orders to the market, until the budget is over.
func (t *TraderModule) ChargeOrders(ctx context.Context, symbol string, priceForHashPerSec *big.Int, step float64, buyMghash float64) (float64, error) {
	requiredHashRate := uint64(buyMghash * hashes)
	benchmarks, err := t.getBenchmarksForSymbol(symbol, uint64(requiredHashRate))
	if err != nil {
		return 0, err
	}
	buyMghash, err = t.CreateOrderOnMarketStep(ctx, step, benchmarks, buyMghash, priceForHashPerSec)
	if err != nil {
		return 0, err
	}
	return buyMghash, nil
}

// Create order on market depends on token.
func (t *TraderModule) CreateOrderOnMarketStep(ctx context.Context, step float64, benchmarks map[string]uint64, buyMgHash float64, price *big.Int) (float64, error) {
	actOrder, err := t.c.Market.CreateOrder(ctx, &sonm.BidOrder{
		Tag:      "Connor bot",
		Duration: &sonm.Duration{},
		Price: &sonm.Price{
			PerSecond: sonm.NewBigInt(price),
		},
		Identity: t.c.cfg.OtherParameters.IdentityForBid,
		Resources: &sonm.BidResources{
			Benchmarks: benchmarks,
			Network: &sonm.BidNetwork{
				Overlay:  false,
				Outbound: true,
				Incoming: false,
			},
		},
	})
	if err != nil {
		t.c.logger.Error("cannot create bidOrder:", zap.Error(err))
		return 0, err
	}
	if actOrder.GetId() != nil && actOrder.GetPrice() != nil {
		reBuyHash := buyMgHash + buyMgHash*step
		if err := t.c.db.SaveOrderIntoDB(&database.OrderDb{
			OrderID:         actOrder.GetId().Unwrap().Int64(),
			Price:           actOrder.GetPrice().Unwrap().Int64(),
			Hashrate:        actOrder.GetBenchmarks().GPUEthHashrate(),
			StartTime:       time.Now(),
			ButterflyEffect: int64(actOrder.GetOrderStatus()),
			ActualStep:      reBuyHash,
		}); err != nil {
			return 0, fmt.Errorf("cannot save order to database: %v", err)
		}
		t.c.logger.Info("Order created",
			zap.Int64("id", actOrder.GetId().Unwrap().Int64()),
			zap.String("price", sonm.NewBigInt(actOrder.GetPrice().Unwrap()).ToPriceString()),
			zap.Uint64("hashrate", actOrder.GetBenchmarks().GPUEthHashrate()))
		return reBuyHash, nil
	}
	return buyMgHash, nil
}

func (t *TraderModule) GetProfitForTokenBySymbol(tokens []*TokenMainData, symbol string) (float64, error) {
	for _, t := range tokens {
		if t.Symbol == symbol {
			return t.ProfitPerMonthUsd, nil
		}
	}
	return 0, fmt.Errorf("cannot get price from token! ")
}

func (t *TraderModule) GetPriceForTokenPerSec(token watchers.TokenWatcher, symbol string) (float64, float64, error) {
	tokens, err := t.profit.CollectTokensMiningProfit(token)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot calculate token prices: %v\r\n", err)
	}
	pricePerMonthUSD, err := t.GetProfitForTokenBySymbol(tokens, symbol)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot get profit for tokens: %v\r\n", err)
	}
	pricePerSec := pricePerMonthUSD / (secsPerDay * daysPerMonth)
	return pricePerMonthUSD, pricePerSec, nil
}

func (t *TraderModule) ReinvoiceOrder(ctx context.Context, cfg *Config, price *sonm.Price, bench map[string]uint64, tag string) error {
	order, err := t.c.Market.CreateOrder(ctx, &sonm.BidOrder{
		Duration: &sonm.Duration{Nanoseconds: 0},
		Price:    price,
		Tag:      tag,
		Identity: cfg.OtherParameters.IdentityForBid,
		Resources: &sonm.BidResources{
			Network: &sonm.BidNetwork{
				Overlay:  false,
				Outbound: true,
				Incoming: false,
			},
			Benchmarks: bench,
		},
	})
	if err != nil {
		t.c.logger.Error("cannot created Lucky Order", zap.Error(err))
		return err
	}
	if err := t.c.db.SaveOrderIntoDB(&database.OrderDb{
		OrderID:         order.GetId().Unwrap().Int64(),
		Price:           order.GetPrice().Unwrap().Int64(),
		Hashrate:        order.GetBenchmarks().GPUEthHashrate(),
		StartTime:       time.Now(),
		ButterflyEffect: int64(OrderStatusREINVOICE),
		ActualStep:      0,
	}); err != nil {
		return fmt.Errorf("cannot save reinvoice order %s to DB: %v \r\n", order.GetId().Unwrap().String(), err)
	}
	t.c.logger.Info("REINVOICE Order", zap.String("Order", order.Id.Unwrap().String()), zap.String("tag", tag),
		zap.String("price", order.Price.Unwrap().String()), zap.Uint64("hashrate", order.GetBenchmarks().GPUEthHashrate()))
	return nil
}

func (t *TraderModule) CmpChangeOfPrice(change float64, def float64) (int32, error) {
	if change >= 100+def {
		return 1, nil
	} else if change < 100-def {
		return -1, nil
	}
	return 0, nil
}

func (t *TraderModule) OrdersProfitTracking(ctx context.Context, cfg *Config, actualPrice *big.Int, ordersDb []*database.OrderDb) error {
	actualPriceCur := big.NewInt(actualPrice.Int64())

	for _, orderDb := range ordersDb {
		actualPriceClone := big.NewInt(0).Set(actualPriceCur)
		order, err := t.c.Market.GetOrderByID(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
		if err != nil {
			t.c.logger.Error("cannot get order from market", zap.Error(err))
			return err
		}
		if orderDb.ButterflyEffect != int64(OrderStatusCANCELLED) {
			if order.GetOrderStatus() == sonm.OrderStatus_ORDER_ACTIVE {
				orderPrice := order.Price.Unwrap()
				pack := int64(order.GetBenchmarks().GPUEthHashrate()) / hashes
				pricePerSecForPack := actualPriceClone.Mul(actualPriceClone, big.NewInt(pack))

				change, err := t.GetChangePercent(pricePerSecForPack, orderPrice)
				if err != nil {
					return fmt.Errorf("cannot get changes percent: %v", err)
				}

				commandPrice, err := t.CmpChangeOfPrice(change, cfg.Sensitivity.OrdersChangePercent)
				if commandPrice == 1 || commandPrice == -1 {
					log.Printf("Active Order Id: %v (price: %v), actual price for PACK: %v (for Mg/h :: %v)change percent: %.2f %%\r\n",
						orderDb.OrderID, sonm.NewBigInt(orderPrice).ToPriceString(), sonm.NewBigInt(pricePerSecForPack).ToPriceString(), sonm.NewBigInt(actualPrice).ToPriceString(), change)
					bench, err := t.GetBidBenchmarks(order)
					if err != nil {
						fmt.Printf("Cannot get benchmarks from Order : %v\r\n", order.Id.Unwrap().Int64())
						return err
					}
					tag := strconv.Itoa(int(orderDb.OrderID))
					t.ReinvoiceOrder(ctx, cfg, &sonm.Price{PerSecond: sonm.NewBigInt(pricePerSecForPack)}, bench, "Reinvoice(update price): "+tag)
					t.c.Market.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
				}
			} else {
				log.Printf("Order is not ACTIVE %v\r\n", order.Id)
				t.c.db.UpdateOrderInDB(orderDb.OrderID, int64(OrderStatusCANCELLED))
			}
		}
	}
	return nil
}

// Pursue a profitable lvl of deal :: profitable price > deal price  => resale order with new price else do nothing. Using deployed and not deployed deals
func (t *TraderModule) DealsProfitTracking(ctx context.Context, actualPrice *big.Int, dealsDb []*database.DealDb, image string) error {
	for _, dealDb := range dealsDb {
		actualPriceClone := t.ClonePrice(actualPrice)
		dealOnMarket, err := t.c.DealClient.Status(ctx, sonm.NewBigIntFromInt(dealDb.DealID))
		if err != nil {
			return fmt.Errorf("cannot get deal info %v\r\n", err)
		}
		if dealOnMarket.Deal.Status != sonm.DealStatus_DEAL_CLOSED && dealDb.DeployStatus == int64(DeployStatusDEPLOYED) {
			bidOrder, err := t.c.Market.GetOrderByID(ctx, &sonm.ID{Id: dealOnMarket.Deal.BidID.Unwrap().String()})
			if err != nil {
				return err
			}
			pack := float64(bidOrder.Benchmarks.GPUEthHashrate()) / float64(hashes)
			actualPriceForPack := actualPriceClone.Mul(actualPriceClone, big.NewInt(int64(pack)))
			dealPrice := dealOnMarket.Deal.Price.Unwrap()

			if actualPriceForPack.Cmp(dealPrice) >= 1 {
				changePercent, err := t.GetChangePercent(actualPriceForPack, dealPrice)
				if err != nil {
					return fmt.Errorf("cannot get change percent from deal: %v", err)
				}
				dealChangeRequest, err := t.c.DealClient.CreateChangeRequest(ctx, &sonm.DealChangeRequest{
					Id:          nil,
					DealID:      dealOnMarket.Deal.Id,
					RequestType: sonm.OrderType_BID,
					Duration:    0,
					Price:       sonm.NewBigIntFromInt(actualPriceForPack.Int64()),
					Status:      sonm.ChangeRequestStatus_REQUEST_CREATED,
					CreatedTS:   nil,
				})
				if err != nil {
					return fmt.Errorf("cannot create change request %v\r\n", err)
				}
				t.c.logger.Info("CREATE DEAL CHANGE REQUEST :: ", zap.String("CR ID ", dealChangeRequest.Unwrap().String()),
					zap.String("active dealID: ", dealOnMarket.Deal.Id.String()),
					zap.String("deal price :", sonm.NewBigInt(dealPrice).ToPriceString()),
					zap.String("actual price for pack :", sonm.NewBigInt(actualPriceForPack).ToPriceString()),
					zap.String("actual price", sonm.NewBigInt(actualPrice).ToPriceString()),
					zap.Float64("change percent", changePercent))
				go t.GetChangeRequest(ctx, dealOnMarket)
			}

		} else if dealOnMarket.Deal.Status != sonm.DealStatus_DEAL_CLOSED && dealDb.DeployStatus == int64(DeployStatusNOTDEPLOYED) {
			getDealFromMarket, err := t.c.DealClient.Status(ctx, sonm.NewBigIntFromInt(dealDb.DealID))
			if err != nil {
				return fmt.Errorf("cannot get deal from Market %v\r\n", err)
			}
			deal := getDealFromMarket.Deal
			t.c.logger.Info("Deploying NEW CONTAINER", zap.Int64("deal", dealDb.DealID))
			task, err := t.pool.DeployNewContainer(ctx, t.c.cfg, deal, image)
			if err != nil {
				t.c.db.UpdateDealInDB(deal.Id.Unwrap().Int64(), int64(DeployStatusNOTDEPLOYED))
				return fmt.Errorf("cannot deploy new container from task %s\r\n", err)
			} else {
				t.c.db.UpdateDealInDB(deal.Id.Unwrap().Int64(), int64(DeployStatusDEPLOYED))
				t.c.logger.Info("New deployed TASK", zap.String("task", task.GetId()), zap.String("deal", deal.GetId().String()))
			}

			bidOrder, err := t.c.Market.GetOrderByID(ctx, &sonm.ID{Id: deal.GetBidID().String()})
			if err != nil {
				return fmt.Errorf("cannot get order by Id: %v\r\n", err)
			}
			bench, err := t.GetBidBenchmarks(bidOrder)
			if err != nil {
				return fmt.Errorf("cannot get benchmarks from bid Order : %v\r\n", bidOrder.Id.Unwrap().Int64())
			}
			if err := t.ReinvoiceOrder(ctx, t.c.cfg, &sonm.Price{PerSecond: deal.GetPrice()}, bench, "Reinvoice(active deal)"); err != nil {
				return fmt.Errorf("cannot reinvoice order %v", err)
			}
		}
	}
	return nil
}

func (t *TraderModule) GetChangeRequest(ctx context.Context, dealChangeRequest *sonm.DealInfoReply) error {
	time.Sleep(time.Duration(t.c.cfg.Sensitivity.WaitingTimeCRSec))
	requestsList, err := t.c.DealClient.ChangeRequestsList(ctx, dealChangeRequest.Deal.Id)
	if err != nil {
		return err
	}
	for _, cr := range requestsList.Requests {
		if cr.DealID == dealChangeRequest.Deal.Id && cr.Status == sonm.ChangeRequestStatus_REQUEST_REJECTED {
			t.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
				Id: dealChangeRequest.Deal.Id,
			})
		}
	}
	return nil

}

func (t *TraderModule) ClonePrice(def *big.Int) *big.Int {
	clone := big.NewInt(def.Int64())
	return big.NewInt(0).Set(clone)
}

func (t *TraderModule) GetChangePercent(actualPriceForPack *big.Int, dealPrice *big.Int) (float64, error) {
	newClone := t.ClonePrice(actualPriceForPack)
	dealClone := t.ClonePrice(dealPrice)
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
func (t *TraderModule) SaveActiveDealsIntoDB(ctx context.Context, dealCli sonm.DealManagementClient) error {
	getDeals, err := dealCli.List(ctx, &sonm.Count{Count: 100})
	if err != nil {
		return fmt.Errorf("Cannot get Deals list %v\r\n", err)
	}
	deals := getDeals.Deal
	if len(deals) > 0 {
		for _, deal := range deals {
			t.c.db.SaveDealIntoDB(&database.DealDb{
				DealID:       deal.GetId().Unwrap().Int64(),
				Status:       int64(deal.GetStatus()),
				Price:        deal.GetPrice().Unwrap().Int64(),
				AskID:        deal.GetAskID().Unwrap().Int64(),
				BidID:        deal.GetBidID().Unwrap().Int64(),
				DeployStatus: int64(DeployStatusNOTDEPLOYED),
				StartTime:    deal.GetStartTime().Unix(),
				LifeTime:     deal.GetEndTime().Unix(),
			})
		}
	} else {
		t.c.logger.Info("No active deals")
	}
	return nil
}
func (t *TraderModule) GetDeployedDeals() ([]int64, error) {
	dealsDB, err := t.c.db.GetDealsFromDB()
	if err != nil {
		return nil, fmt.Errorf("cannot create benchmarkes for symbol \"%s\"", err)
	}
	deployedDeals := make([]int64, 0)
	for _, d := range dealsDB {
		if d.DeployStatus == int64(DeployStatusDEPLOYED) {
			deal := d.DealID
			deployedDeals = append(deployedDeals, deal)
		}
	}
	return deployedDeals, nil
}
func (t *TraderModule) GetBidBenchmarks(bidOrder *sonm.Order) (map[string]uint64, error) {
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
		"gpu-eth-hashrate":    getBench.GPUEthHashrate(),
	}
	return bMap, nil
}
func (t *TraderModule) FloatToBigInt(val float64) *big.Int {
	price := val * params.Ether
	return big.NewInt(int64(price))
}

// Init benchmarks
func (t *TraderModule) newBaseBenchmarks() map[string]uint64 {
	return map[string]uint64{
		"ram-size":            t.c.cfg.Benchmark.RamSize,
		"cpu-cores":           t.c.cfg.Benchmark.CpuCores,
		"cpu-sysbench-single": t.c.cfg.Benchmark.CpuSysbenchSingle,
		"cpu-sysbench-multi":  t.c.cfg.Benchmark.CpuSysbenchMulti,
		"net-download":        t.c.cfg.Benchmark.NetDownload,
		"net-upload":          t.c.cfg.Benchmark.NetUpload,
		"gpu-count":           t.c.cfg.Benchmark.GpuCount,
		"gpu-mem":             t.c.cfg.Benchmark.GpuMem,
	}
}
func (t *TraderModule) newBenchmarksWithGPU(ethHashRate uint64) map[string]uint64 {
	b := t.newBaseBenchmarks()
	b["gpu-eth-hashrate"] = ethHashRate
	return b
}
func (t *TraderModule) newBenchmarksWithoutGPU() map[string]uint64 {
	return t.newBaseBenchmarks()
}
func (t *TraderModule) getBenchmarksForSymbol(symbol string, ethHashRate uint64) (map[string]uint64, error) {
	switch symbol {
	case "ETH":
		return t.newBenchmarksWithGPU(ethHashRate), nil
	case "ZEC":
		return t.newBenchmarksWithoutGPU(), nil
	case "XMR":
		return t.newBenchmarksWithGPU(ethHashRate), nil
	default:
		return nil, fmt.Errorf("cannot create benchmakes for symbol \"%s\"", symbol)
	}
}

//FIXME: Redo the waiting time for old orders
// Get orders FROM DATABASE ==> if order's created time more cfg.Days -> order is cancelled ==> save to BD as "cancelled" (3).
func (t *TraderModule) CheckAndCancelOldOrders(ctx context.Context, cfg *Config) {
	ordersDb, err := t.c.db.GetOrdersFromDB()
	if err != nil {
		fmt.Printf("Cannot get orders from DB %v\r\n", err)
	}
	for _, o := range ordersDb {
		subtract := time.Now().AddDate(0, 0, -o.StartTime.Day()).Day()
		if int64(subtract) >= cfg.Sensitivity.SensitivityForOrders && subtract > daysPerMonth {
			fmt.Printf("Orders suspected of cancellation: : %v, passed time: %v\r\n", o.OrderID, subtract)
			//TODO: change status to "Cancelled"
			t.c.Market.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(o.OrderID))})
			t.c.db.UpdateOrderInDB(o.OrderID, int64(OrderStatusCANCELLED))
		}
	}
}
