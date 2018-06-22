package connor

import (
	"context"
	"fmt"
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"
	"github.com/ethereum/go-ethereum/params"
	"github.com/sonm-io/core/connor/database"
)

// POOL MODULE
const (
	EthPool = "stratum+tcp://eth-eu1.nanopool.org:9999"
)

type Modules struct {
	c *Connor
}

func NewModules(c *Connor) *Modules {
	return &Modules{
		c: c,
	}
}

func (m *Modules) DeployNewContainer(ctx context.Context, cfg *Config, deal *sonm.Deal, image string) (*sonm.StartTaskReply, error) {
	env := map[string]string{
		"ETH_POOL": EthPool,
		"WALLET":   cfg.PoolAddress.EthPoolAddr,
		"WORKER":   deal.Id.String(),
	}
	container := &sonm.Container{
		Image: image,
		Env:   env,
	}
	spec := &sonm.TaskSpec{
		Container: container,
		Registry:  &sonm.Registry{},
		Resources: &sonm.AskPlanResources{},
	}
	startTaskRequest := &sonm.StartTaskRequest{
		DealID: deal.GetId(),
		Spec:   spec,
	}
	reply, err := m.c.TaskClient.Start(ctx, startTaskRequest)
	if err != nil {
		fmt.Printf("Cannot create start task request %s", err)
		return nil, err
	}
	return reply, nil
}

func PoolTrack(ctx context.Context, pool watchers.PoolWatcher, avgpool watchers.PoolWatcher, addr string, dealCli sonm.DealManagementClient, marketCli sonm.MarketClient) error {
	PoolTracking(ctx, dealCli, marketCli, pool, addr)
	return nil
}

func (m *Modules) SavePoolDataToDb(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}
	for _, rh := range dataRH.Data {
		fmt.Printf("w:: %v, data :: %v\r\n", rh.Worker, rh.Hashrate)
		// TODO: replace time + pool balance + pool hashrate
		m.c.db.SavePoolIntoDB(&database.PoolDb{
			PoolId:                 addr,
			PoolBalance:            0,
			PoolHashrate:           0,
			WorkerID:               rh.Worker,
			WorkerReportedHashrate: rh.Hashrate,
			WorkerAvgHashrate:      0,
			BadGuy:                 0,
			Iterations:             0,
			TimeStart:              time.Now(),
			TimeUpdate:             time.Now(),
		})
	}
	return nil
}

func (m *Modules) UpdateRHPoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.Data {
		// TODO: replace time
		m.c.db.UpdateReportedHashratePoolDB(rh.Worker, rh.Hashrate)
	}
	return nil
}

func (m *Modules) UpdateAvgPoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.Data {
		// TODO: replace time
		m.c.db.UpdateAvgPoolDB(rh.Worker, rh.Hashrate)
	}
	return nil
}

// TODO: avg pool не принимается =(((((((((((((((((((((((((
func (m *Modules) UpdatePoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	workers, err := m.c.db.GetWorkersFromDB()
	if err != nil {
		fmt.Printf("cannot get worker from pool DB")
		return err
	}
	// TODO: Bad guy detected :: if iterations < 4 && badguy <5
	for _, w := range workers {
		if w.Iterations < 4 {
			if err = m.UpdateRHPoolData(ctx, pool, addr); err != nil {
				log.Printf("cannot update RH pool data!")
				return err
			}
		} else {
			if err = m.UpdateAvgPoolData(ctx, pool, addr); err != nil {
				log.Printf("cannot update AVG pool data!")
				return err
			}
		}
	}
	return nil
}

func (m *Modules) PoolTracking(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	m.UpdatePoolData(ctx, pool, addr)
	workers, err := m.c.db.GetWorkersFromDB()
	if err != nil {
		fmt.Printf("cannot get worker from pool DB")
		return err
	}

	for _, w := range workers {
		wId, err := strconv.Atoi(w.WorkerID)

		dealID, err := m.c.DealClient.Status(ctx, &sonm.BigInt{Abs: big.NewInt(int64(wId)).Bytes()})
		if err != nil {
			fmt.Printf("Cannot get deal from market %v\r\n", w.WorkerID)
			return err
		}
		m.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
			Id:            dealID.Deal.Id,
			BlacklistType: 1,
		})
		os.Exit(1)
		bidOrder, err := m.c.Market.GetOrderByID(ctx, &sonm.ID{Id: dealID.Deal.BidID.Unwrap().String()})
		if err != nil {
			fmt.Printf("cannot get order from market by ID")
			return err
		}

		bidHashrate := bidOrder.GetBenchmarks().GPUEthHashrate()
		iteration := int32(w.Iterations + 1)

		if iteration < 4 {
			log.Printf("ITERATION :: %v for worker :: %v !\r\n", iteration, w.WorkerID)
			workerReportedHashrate := uint64(w.WorkerReportedHashrate * 1000000)
			if workerReportedHashrate < bidHashrate {
				log.Printf("ID :: %v ==> wRH %v < deal (bid) hashrate %v ==> PID! \r\n", w.WorkerID, workerReportedHashrate, bidHashrate)
				if w.BadGuy < 5 {
					newStatus := w.BadGuy + 1
					m.c.db.UpdateStatusPoolDB(w.WorkerID, newStatus)
				} else {
					m.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
						Id:            dealID.Deal.Id,
						BlacklistType: 1,
					})

					fmt.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
				}
			}
		} else {
			log.Printf("Iteration for worker :: %v more than 4 == > get Avg Data", w.WorkerID)
			workerAvgHashrate := uint64(w.WorkerAvgHashrate * 1000000)
			if workerAvgHashrate < bidHashrate {
				log.Printf("ID :: %v ==> wRH %v < deal (bid) hashrate %v ==> PID! \r\n", w.WorkerID, workerAvgHashrate, bidHashrate)
				if w.BadGuy < 5 {
					newStatus := w.BadGuy + 1
					m.c.db.UpdateStatusPoolDB(w.WorkerID, newStatus)
				} else {
					m.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
						Id:            dealID.Deal.Id,
						BlacklistType: 1,
					})
					fmt.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
				}
			}
		}
		m.c.db.UpdateIterationPoolDB(w.WorkerID, iteration)
	}
	return nil
}

// trader module
const (
	hashes       = 1000000
	daysPerMonth = 30
	secsPerDay   = 86400
	partCharge   = 0.5
)

type DeployStatus int32

const (
	DeployStatusDEPLOYED    DeployStatus = 3
	DeployStatusNOTDEPLOYED DeployStatus = 4
	DeployStatusDESTROYED   DeployStatus = 5 //send asker to blackList
)

type OrderStatus int32

const (
	OrderStatusCancelled OrderStatus = 3
	OrderStatusReinvoice OrderStatus = 4
)

func (m *Modules) getTokenConfiguration(symbol string, cfg *Config) (float64, float64, float64, error) {
	switch symbol {
	case "ETH":
		return cfg.ChargeIntervalETH.Start, cfg.ChargeIntervalETH.Destination, cfg.Distances.StepForETH, nil
	case "ZEC":
		return cfg.ChargeIntervalZEC.Start, cfg.ChargeIntervalZEC.Destination, cfg.Distances.StepForZEC, nil
	case "XMR":
		return cfg.ChargeIntervalXMR.Start, cfg.ChargeIntervalXMR.Destination, cfg.Distances.StepForXMR, nil
	}
	return 0, 0, 0, nil
}

func (m *Modules) ChargeOrdersOnce(ctx context.Context, symbol string, marketClient sonm.MarketClient, token watchers.TokenWatcher, snm watchers.PriceWatcher, balanceReply *sonm.BalanceReply, cfg *Config, ethAddr *sonm.EthAddress) error {
	m.c.db.CreateOrderDB()
	start, destination, step, err := m.getTokenConfiguration(symbol, cfg)
	if err != nil {
		log.Printf("cannot get token configuraton %v\r\n", err)
		return err
	}
	count, err := m.c.db.GetCountFromDB()
	if err != nil {
		log.Printf("Cannot get count from DB: %v\r\n", err)
		return err
	}
	if count == 0 {
		log.Printf("Save TEST order cause DB is empty! \r\n")
		if err := m.c.db.SaveOrderIntoDB(&database.OrderDb{
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

	pricePerMonthUSD, pricePerSecMh, err := m.GetPriceForTokenPerSec(token, symbol)
	if err != nil {
		log.Printf("Cannot get profit for tokens: %v\r\n", err)
		return err
	}
	limitChargeInSNM := m.LimitChargeSNM(balanceReply.GetSideBalance().Unwrap(), partCharge)
	limitChargeInSNMClone := big.NewInt(0).Set(limitChargeInSNM)
	limitChargeInUSD := m.ConvertingToUSDBalance(limitChargeInSNMClone, snm.GetPrice())

	mhashForToken, err := m.c.db.GetLastActualStepFromDb()
	if err != nil {
		log.Printf("Cannot get last actual step from DB %v\r\n", err)
		return err
	}

	pricePackMhInUSDPerMonth := mhashForToken * (pricePerMonthUSD * cfg.Sensitivity.MarginAccounting)
	sumOrdersPerMonth := limitChargeInUSD / pricePackMhInUSDPerMonth

	log.Printf("CHARGE %v ORDERS ONCE: \r\n"+
		"Price per month %.2f USD, price per sec %v USD for Mh/s\r\n"+
		"Limit for Charge		   :: %.2f $ (%.8v SNM)\r\n"+
		"Price Order Per Month	   :: %f $\r\n"+
		"Default step			   :: %.2f MH/s\r\n"+
		"You can create 			   :: %v orders ranging from: %.2f Mh/s - %.2f Mh/s with step: %.2f\r\n"+
		"START CHARGE ..................................",
		symbol, pricePerMonthUSD, pricePerSecMh, limitChargeInUSD, m.PriceToString(limitChargeInSNM), pricePackMhInUSDPerMonth, mhashForToken, int(sumOrdersPerMonth),
		start, destination, step)
	os.Exit(1)
	for i := 0; i < int(sumOrdersPerMonth); i++ {
		if mhashForToken >= destination {
			fmt.Printf("Charge is finished cause reached the limit %.2f Mh/s\r\n", cfg.ChargeIntervalETH.Destination)
			break
		}
		pricePerSecPack := m.FloatToBigInt(mhashForToken * pricePerSecMh)
		log.Printf("Price :: %v\r\n", m.PriceToString(pricePerSecPack))
		mhashForToken, err = m.ChargeOrders(ctx, cfg, marketClient, symbol, pricePerSecPack, step, mhashForToken, ethAddr)
		if err != nil {
			return fmt.Errorf("Cannot charging market! %v\r\n", err)
		}
	}
	log.Printf("Balance is not enough to make deals!\r\n")
	return nil
}

// Prepare price and Map depends on token symbol.
// Create orders to the market, until the budget is over.
func (m *Modules) ChargeOrders(ctx context.Context, cfg *Config, client sonm.MarketClient, symbol string, priceForHashPerSec *big.Int, step float64, buyMghash float64, ethAddr *sonm.EthAddress) (float64, error) {
	requiredHashRate := uint64(buyMghash * hashes)
	benchmarks, err := m.getBenchmarksForSymbol(symbol, uint64(requiredHashRate))
	if err != nil {
		return 0, err
	}
	buyMghash, err = m.CreateOrderOnMarketStep(ctx, cfg, client, step, benchmarks, buyMghash, priceForHashPerSec, ethAddr)
	if err != nil {
		return 0, err
	}
	return buyMghash, nil
}

// Create order on market depends on token.
func (m *Modules) CreateOrderOnMarketStep(ctx context.Context, cfg *Config, market sonm.MarketClient, step float64, benchmarks map[string]uint64, buyMgHash float64, price *big.Int, ethAddr *sonm.EthAddress) (float64, error) {
	actOrder, err := market.CreateOrder(ctx, &sonm.BidOrder{
		Tag:      "Connor bot",
		Duration: &sonm.Duration{},
		Price: &sonm.Price{
			PerSecond: sonm.NewBigInt(price),
		},
		Blacklist: ethAddr,
		Identity:  cfg.OtherParameters.IdentityForBid,
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
		log.Printf("Сannot create bidOrder: %v", err)
		return 0, err
	}
	if actOrder.GetId() != nil && actOrder.GetPrice() != nil {
		reBuyHash := buyMgHash + buyMgHash*step
		if err := m.c.db.SaveOrderIntoDB(&database.OrderDb{
			OrderID:         actOrder.GetId().Unwrap().Int64(),
			Price:           actOrder.GetPrice().Unwrap().Int64(),
			Hashrate:        actOrder.GetBenchmarks().GPUEthHashrate(),
			StartTime:       time.Now(),
			ButterflyEffect: int32(actOrder.GetOrderStatus()),
			ActualStep:      reBuyHash,
		}); err != nil {
			return 0, fmt.Errorf("cannot save order to database: %v", err)
		}
		log.Printf("Order created ==> ID: %v, Price: %v $, Hashrate: %v H/s \r\n",
			actOrder.GetId().Unwrap().Int64(),
			m.PriceToString(actOrder.GetPrice().Unwrap()),
			actOrder.GetBenchmarks().GPUEthHashrate())
		return reBuyHash, nil
	}
	return buyMgHash, nil
}

func (m *Modules) GetPriceForTokenPerSec(token watchers.TokenWatcher, symbol string) (float64, float64, error) {
	tokens, err := m.CollectTokensMiningProfit(token)
	if err != nil {
		return 0, 0, fmt.Errorf("Cannot calculate token prices: %v\r\n", err)
	}
	pricePerMonthUSD, err := m.GetProfitForTokenBySymbol(tokens, symbol)
	if err != nil {
		return 0, 0, fmt.Errorf("Cannot get profit for tokens: %v\r\n", err)
	}
	pricePerSec := pricePerMonthUSD / (secsPerDay * daysPerMonth)
	return pricePerMonthUSD, pricePerSec, nil
}

// After charge orders
func (m *Modules) TradeObserve(ctx context.Context, c *Connor, pool watchers.PoolWatcher, token watchers.TokenWatcher, cfg *Config) error {
	log.Printf("MODULE TRADE OBSERVE :: ")
	err := m.SaveActiveDealsIntoDB(ctx, c.DealClient)
	if err != nil {
		fmt.Printf("cannot save active deals intoDB %v\r\n", err)
	}

	_, pricePerSec, err := m.GetPriceForTokenPerSec(token, "ETH")
	if err != nil {
		fmt.Printf("cannot get pricePerSec for token per sec %v\r\n", err)
	}

	actualPrice := m.FloatToBigInt(pricePerSec)
	log.Printf("Actual price per sec :: %v\r\n", m.PriceToString(actualPrice))

	deals, err := m.c.db.GetDealsFromDB()
	if err != nil {
		return fmt.Errorf("cannot get deals from DB %v\r\n", err)
	}
	orders, err := m.c.db.GetOrdersFromDB()
	if err != nil {
		return fmt.Errorf("cannot get orders from DB %v\r\n", err)
	}
	m.OrdersProfitTracking(ctx, cfg, actualPrice, orders)
	m.ResponseActiveDeals(ctx, cfg, deals, c.DealClient, c.TaskClient, cfg.Images.Image)
	m.DealsProfitTracking(ctx, actualPrice, c.DealClient, c.Market, deals)
	return nil
}

func (m *Modules) ReinvoiceOrder(ctx context.Context, cfg *Config, price *sonm.Price, bench map[string]uint64, tag string) error {
	order, err := m.c.Market.CreateOrder(ctx, &sonm.BidOrder{
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
		fmt.Printf("Cannot created Lucky Order: %v\r\n", err)
		return err
	}
	if err := m.c.db.SaveOrderIntoDB(&database.OrderDb{
		OrderID:         order.GetId().Unwrap().Int64(),
		Price:           order.GetPrice().Unwrap().Int64(),
		Hashrate:        order.GetBenchmarks().GPUEthHashrate(),
		StartTime:       time.Now(),
		ButterflyEffect: int32(OrderStatusReinvoice),
		ActualStep:      0, // step copy
	}); err != nil {
		return fmt.Errorf("cannot save reinvoice order %s to DB: %v \r\n", order.GetId().Unwrap().String(), err)
	}
	log.Printf("REINVOICE Order ===> %v created (descendant: %v), price: %v, hashrate: %v",
		order.GetId(), tag, order.GetPrice(), order.GetBenchmarks().GPUEthHashrate())
	return nil
}

// Get active deals --> for each active deal DEPLOY NEW CONTAINER --> Reinvoice order
func (m *Modules) ResponseActiveDeals(ctx context.Context, cfg *Config, dealsDb []*database.DealDb, dealCli sonm.DealManagementClient, tMng sonm.TaskManagementClient, imageMonero string) error {
	for _, dealDb := range dealsDb {
		if dealDb.Status == 1 && dealDb.DeployStatus == 4 {
			getDealFromMarket, err := dealCli.Status(ctx, &sonm.BigInt{Abs: big.NewInt(dealDb.DealID).Bytes()})
			if err != nil {
				return fmt.Errorf("cannot get deal from Market %v\r\n", err)
			}
			deal := getDealFromMarket.Deal

			fmt.Printf("Deploying NEW CONTAINER ==> for dealDB %v (deal on market: %v\r\n)", dealDb.DealID, deal.GetId().Unwrap().String())
			task, err := m.DeployNewContainer(ctx, cfg, deal, imageMonero)
			if err != nil {
				m.c.db.UpdateDealInDB(deal.Id.Unwrap().Int64(), int32(DeployStatusNOTDEPLOYED))
				return fmt.Errorf("Cannot deploy new container from task %s\r\n", err)
			} else {
				m.c.db.UpdateDealInDB(deal.Id.Unwrap().Int64(), int32(DeployStatusDEPLOYED))
				fmt.Printf("New deployed task %v, for deal ID: %v\r\n", task.GetId(), deal.GetId().String())
			}

			bidOrder, err := m.c.Market.GetOrderByID(ctx, &sonm.ID{Id: deal.GetBidID().String()})
			if err != nil {
				fmt.Printf("Cannot get order by Id: %v\r\n", err)
				return err
			}
			bench, err := m.GetBidBenchmarks(bidOrder)
			if err != nil {
				fmt.Printf("Cannot get benchmarks from bid Order : %v\r\n", bidOrder.Id.Unwrap().Int64())
				return err
			}
			m.ReinvoiceOrder(ctx, cfg, &sonm.Price{PerSecond: deal.GetPrice()}, bench, "Reinvoice")
		} else {
			fmt.Printf("For all received deals status :: DEPLOYED")
		}
	}
	return nil
}

func (m *Modules) CmpChangeOfPrice(change float64, def float64) (int32, error) {
	if change >= 100+def {
		return 1, nil //increase
	} else if change < 100-def {
		return -1, nil
	}
	return 0, nil
}

func (m *Modules) OrdersProfitTracking(ctx context.Context, cfg *Config, actualPrice *big.Int, ordersDb []*database.OrderDb) error {
	log.Printf("MODULE :: Orders Profit Tracking")
	actualPriceCur := big.NewInt(actualPrice.Int64())

	for _, orderDb := range ordersDb {
		actualPriceClone := big.NewInt(0).Set(actualPriceCur)
		order, err := m.c.Market.GetOrderByID(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
		if err != nil {
			log.Printf("cannot get order from market %v\r\n", err)
			return err
		}
		if orderDb.ButterflyEffect != 3 { // 3 :: Cancelled (is not)
			if order.GetOrderStatus() == 2 { // 2 :: Active
				orderPrice := order.Price.Unwrap()
				pack := int64(order.GetBenchmarks().GPUEthHashrate()) / hashes
				pricePerSecForPack := actualPriceClone.Mul(actualPriceClone, big.NewInt(pack))

				//actualPackPrice := big.NewInt(pricePerSecForPack.Int64())
				change, err := m.GetChangePercent(pricePerSecForPack, orderPrice)
				if err != nil {
					return fmt.Errorf("cannot get changes percent: %v", err)
				}
				log.Printf("Active Order Id: %v (price: %v), actual price for PACK: %v (for Mg/h :: %v)change percent: %.2f %%\r\n", orderDb.OrderID, m.PriceToString(orderPrice), m.PriceToString(pricePerSecForPack), m.PriceToString(actualPrice), change)
				commandPrice, err := m.CmpChangeOfPrice(change, 5)
				if commandPrice == 1 || commandPrice == -1 {
					bench, err := m.GetBidBenchmarks(order)
					if err != nil {
						fmt.Printf("Cannot get benchmarks from Order : %v\r\n", order.Id.Unwrap().Int64())
						return err
					}
					tag := strconv.Itoa(int(orderDb.OrderID))
					m.ReinvoiceOrder(ctx, cfg, &sonm.Price{PerSecond: sonm.NewBigInt(pricePerSecForPack)}, bench, "Reinvoice OldOrder: "+tag)
					m.c.Market.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
				}
			} else {
				fmt.Printf("Order is not ACTIVE %v\r\n", order.Id)
				m.c.db.UpdateOrderInDB(orderDb.OrderID, 3)
			}
		}
	}
	return nil
}

// Pursue a profitable lvl of deal :: profitable price > deal price  => resale order with new price else do nothing
// Debug :: actualPrice == actualPriceForPack
func (m *Modules) DealsProfitTracking(ctx context.Context, actualPrice *big.Int, dealClient sonm.DealManagementClient, marketClient sonm.MarketClient, dealsDb []*database.DealDb) error {
	log.Printf("DEALS PROFIT TRACKING ::")
	for _, d := range dealsDb {
		actualPriceClone, err := m.ClonePrice(actualPrice)
		if err != nil {
			return fmt.Errorf("Cannot get clone price: %v", err)
		}

		dealOnMarket, err := dealClient.Status(ctx, &sonm.BigInt{Abs: big.NewInt(d.DealID).Bytes()})
		if err != nil {
			return fmt.Errorf("cannot get deal info %v\r\n", err)
		}
		bidOrder, err := marketClient.GetOrderByID(ctx, &sonm.ID{Id: dealOnMarket.Deal.BidID.Unwrap().String()})
		if err != nil {
			return err
		}
		pack := float64(bidOrder.Benchmarks.GPUEthHashrate()) / float64(1000000)
		actualPriceForPack := actualPriceClone.Mul(actualPriceClone, big.NewInt(int64(pack)))
		dealPrice := dealOnMarket.Deal.Price.Unwrap()
		log.Printf("Deal id::%v (Bid:: %v) price :: %v, actual price for pack :: %v (hashes %v)\r\n", dealOnMarket.Deal.Id.String(), bidOrder.Id.String(), m.PriceToString(dealPrice), m.PriceToString(actualPriceForPack), pack)
		os.Exit(1)

		if actualPriceForPack.Cmp(dealPrice) >= 1 {
			changePercent, err := m.GetChangePercent(actualPriceForPack, dealPrice)
			if err != nil {
				return fmt.Errorf("cannot get change percent from deal: %v", err)
			}
			log.Printf("Create CR ===> Active Deal Id: %v (price: %v), actual price for PACK: %v (for Mg/h :: %v) change percent: %.2f %%\r\n",
				dealOnMarket.Deal.Id.String(), m.PriceToString(dealPrice), m.PriceToString(actualPriceForPack), m.PriceToString(actualPrice), changePercent)
			dealChangeRequest, err := dealClient.CreateChangeRequest(ctx, &sonm.DealChangeRequest{
				Id:          nil,
				DealID:      dealOnMarket.Deal.Id,
				RequestType: 1,
				Duration:    0,
				Price:       &sonm.BigInt{Abs: actualPriceForPack.Bytes()},
				Status:      1,
				CreatedTS:   nil,
			})
			if err != nil {
				return fmt.Errorf("cannot create change request %v\r\n", err)
			}
			fmt.Printf("CR :: %v for DealId :: %v\r\n", dealChangeRequest.String(), dealOnMarket.Deal.Id)
		}
	}
	return nil
}

// Get orders FROM DATABASE ==> if order's created time more cfg.Days -> order is cancelled ==> save to BD as "cancelled" (3).
func (m *Modules) CheckAndCancelOldOrders(ctx context.Context, cfg *Config) {
	ordersDb, err := m.c.db.GetOrdersFromDB()
	if err != nil {
		fmt.Printf("Cannot get orders from DB %v\r\n", err)
		os.Exit(1)
	}
	for _, o := range ordersDb {
		subtract := time.Now().AddDate(0, 0, -o.StartTime.Day()).Day()
		if subtract >= cfg.Sensitivity.SensitivityForOrders && subtract > 30 {
			fmt.Printf("Orders suspected of cancellation: : %v, passed time: %v\r\n", o.OrderID, subtract)
			//TODO: change status to "Cancelled"
			m.c.Market.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(o.OrderID))})
			m.c.db.UpdateOrderInDB(o.OrderID, int32(OrderStatusCancelled))
		}
	}
}

func (m *Modules) GetChangeRequest(ctx context.Context, dealCli sonm.DealManagementClient) {
	//TODO : Create check ChangeRequest status (by approve)
}

func (m *Modules) ClonePrice(def *big.Int) (*big.Int, error) {
	clone := big.NewInt(def.Int64())
	return big.NewInt(0).Set(clone), nil
}

func (m *Modules) GetChangePercent(actualPriceForPack *big.Int, dealPrice *big.Int) (float64, error) {
	newClone, err := m.ClonePrice(actualPriceForPack)
	if err != nil {
		return 0, fmt.Errorf("cannot get clone price %v", err)
	}
	dealClone, err := m.ClonePrice(dealPrice)
	if err != nil {
		return 0, fmt.Errorf("cannot get clone price %v", err)
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
func (m *Modules) SaveActiveDealsIntoDB(ctx context.Context, dealCli sonm.DealManagementClient) error {
	getDeals, err := dealCli.List(ctx, &sonm.Count{Count: 100})
	if err != nil {
		fmt.Printf("Cannot get Deals list %v\r\n", err)
		return err
	}
	deals := getDeals.Deal
	if len(deals) > 0 {
		for _, deal := range deals {
			m.c.db.SaveDealIntoDB(&database.DealDb{
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

func (m *Modules) GetDeployedDeals() ([]int64, error) {
	dealsDB, err := m.c.db.GetDealsFromDB()
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

func (m *Modules) GetBidBenchmarks(bidOrder *sonm.Order) (map[string]uint64, error) {
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
func (m *Modules) FloatToBigInt(val float64) *big.Int {
	price := val * params.Ether
	return big.NewInt(int64(price))
}

// Init benchmarks
func (m *Modules) newBaseBenchmarks() map[string]uint64 {
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
func (m *Modules) newBenchmarksWithGPU(ethHashRate uint64) map[string]uint64 {
	b := m.newBaseBenchmarks()
	b["gpu-eth-hashrate"] = ethHashRate
	return b
}
func (m *Modules) newBenchmarksWithoutGPU() map[string]uint64 {
	return m.newBaseBenchmarks()
}
func (m *Modules) getBenchmarksForSymbol(symbol string, ethHashRate uint64) (map[string]uint64, error) {
	switch symbol {
	case "ETH":
		return m.newBenchmarksWithGPU(ethHashRate), nil
	case "ZEC":
		return m.newBenchmarksWithoutGPU(), nil
	case "XMR":
		return m.newBenchmarksWithGPU(ethHashRate), nil
	default:
		return nil, fmt.Errorf("cannot create benchmakes for symbol \"%s\"", symbol)
	}
}
func (m *Modules) PriceToString(c *big.Int) string {
	v := big.NewFloat(0).SetInt(c)
	div := big.NewFloat(params.Ether)
	r := big.NewFloat(0).Quo(v, div)
	return r.Text('f', -18)
}

// CALCULATE TOKENS
const (
	hashingPower     = 1
	costPerkWh       = 0.0
	powerConsumption = 0.0
)

type powerAndDivider struct {
	power float64
	div   float64
}

func (m *Modules) getHashPowerAndDividerForToken(s string, hp float64) (float64, float64, bool) {
	var tokenHashPower = map[string]powerAndDivider{
		"ETH": {div: 1, power: hashingPower * 1000000.0},
		"XMR": {div: 1, power: 1},
		"ZEC": {div: 1, power: 1},
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

func (m *Modules) getTokensForProfitCalculation() []*TokenMainData {
	return []*TokenMainData{
		{Symbol: "ETH"},
		{Symbol: "XMR"},
		{Symbol: "ZEC"},
	}
}
func (m *Modules) CollectTokensMiningProfit(t watchers.TokenWatcher) ([]*TokenMainData, error) {
	var tokensForCalc = m.getTokensForProfitCalculation()
	for _, token := range tokensForCalc {
		tokenData, err := t.GetTokenData(token.Symbol)
		if err != nil {
			log.Printf("cannot get token data %v\r\n", err)
		}
		hashesPerSecond, divider, ok := m.getHashPowerAndDividerForToken(tokenData.Symbol, tokenData.NetHashPerSec)
		if !ok {
			log.Printf("DEBUG :: cannot process tokenData %s, not in list\r\n", tokenData.Symbol)
			continue
		}
		netHashesPersec := int64(tokenData.NetHashPerSec)
		token.ProfitPerMonthUsd = m.CalculateMiningProfit(tokenData.PriceUSD, hashesPerSecond, float64(netHashesPersec), tokenData.BlockReward, divider, tokenData.BlockTime)
		log.Printf("TOKEN :: %v, priceUSD: %v, hashes per Sec: %v, net hashes per sec : %v, block reward : %v, divider %v, blockTime : %v, PROFIT PER MONTH : %v\r\n",
			token.Symbol, tokenData.PriceUSD, hashesPerSecond, netHashesPersec, tokenData.BlockReward, divider, tokenData.BlockTime, token.ProfitPerMonthUsd)
		if token.Symbol == "ETH" {
			m.c.db.SaveProfitToken(&database.TokenDb{
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
func (m *Modules) CalculateMiningProfit(usd, hashesPerSecond, netHashesPerSecond, blockReward, div float64, blockTime int) float64 {
	currentHashingPower := hashesPerSecond / div
	miningShare := currentHashingPower / (netHashesPerSecond + currentHashingPower)
	minedPerDay := miningShare * 86400 / float64(blockTime) * blockReward / div
	powerCostPerDayUSD := (powerConsumption * 24) / 1000 * costPerkWh
	returnPerDayUSD := (usd*float64(minedPerDay) - (usd * float64(minedPerDay) * 0.01)) - powerCostPerDayUSD
	perMonthUSD := float64(returnPerDayUSD * 30)
	return perMonthUSD
}

func (m *Modules) GetProfitForTokenBySymbol(tokens []*TokenMainData, symbol string) (float64, error) {
	for _, t := range tokens {
		if t.Symbol == symbol {
			return t.ProfitPerMonthUsd, nil
		}
	}
	return 0, fmt.Errorf("Cannot get price from token! ")
}

//Limit balance for Charge orders. Default value = 0.5
func (m *Modules) LimitChargeSNM(balance *big.Int, partCharge float64) *big.Int {
	limitChargeSNM := balance.Div(balance, big.NewInt(100))
	limitChargeSNM = limitChargeSNM.Mul(balance, big.NewInt(int64(partCharge*100)))
	return limitChargeSNM
}

//converting snmBalance = > USD Balance
func (m *Modules) ConvertingToUSDBalance(balanceSide *big.Int, snmPrice float64) float64 {
	bal := balanceSide.Mul(balanceSide, big.NewInt(int64(snmPrice*1e18)))
	bal = bal.Div(bal, big.NewInt(1e18))
	d, e := bal.DivMod(bal, big.NewInt(1e18), big.NewInt(0))
	f, _ := big.NewFloat(0).SetInt(e).Float64()
	res := float64(d.Int64()) + (f / 1e18)
	return res
}
