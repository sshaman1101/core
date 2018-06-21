package modules

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/sonm-io/core/connor/config"
	"github.com/sonm-io/core/connor/records"
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
)

const (
	hashes       = 1000000
	symbETH      = "ETH"
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

func ChargeOrdersOnce(ctx context.Context, marketClient sonm.MarketClient, token watchers.TokenWatcher, snm watchers.PriceWatcher, balanceReply *sonm.BalanceReply, cfg *config.Config, ethAddr *sonm.EthAddress, identityLvl sonm.IdentityLevel) error {
	records.CreateOrderDB()
	count, err := records.GetCountFromDB()
	if err != nil {
		log.Printf("Cannot get count from DB: %v\r\n", err)
		return err
	}
	if count == 0 {
		log.Printf("Save TEST order cause DB is empty! \r\n")
		if err := records.SaveOrderIntoDB(&records.OrderDb{
			OrderID:         0,
			Price:           0,
			Hashrate:        0,
			StartTime:       time.Time{},
			ButterflyEffect: 2,
			ActualStep:      cfg.ChargeInterval.Start,
		});
			err != nil {
			fmt.Printf("Cannot save order into DB %v\r\n", err)
		}
	}

	pricePerMonthUSD, pricePerSecMh, err := GetPriceForTokenPerSec(token)
	if err != nil {
		log.Printf("Cannot get profit for tokens: %v\r\n", err)
		return err
	}
	limitChargeInSNM := LimitChargeSNM(balanceReply.GetSideBalance().Unwrap(), partCharge)
	limitChargeInSNMClone := big.NewInt(0).Set(limitChargeInSNM)
	limitChargeInUSD := ConvertingToUSDBalance(limitChargeInSNMClone, snm.GetPrice())

	mhashForETH, err := records.GetLastActualStepFromDb()
	if err != nil {
		log.Printf("Cannot get last actual step from DB %v\r\n", err)
		return err
	}

	pricePackMhInUSDPerMonth := mhashForETH * (pricePerMonthUSD * cfg.Sensitivity.MarginAccounting)
	sumOrdersPerMonth := limitChargeInUSD / pricePackMhInUSDPerMonth

	log.Printf("CHARGE ORDERS ONCE: \r\n"+
		"Price per month %.2f USD, price per sec %v USD for Mh/s\r\n"+
		"Limit for Charge		   :: %.2f $ (%.8v SNM)\r\n"+
		"Price Order Per Month	   :: %f $\r\n"+
		"Default step			   :: %.2f MH/s\r\n"+
		"You can create 			   :: %v orders ranging from: %.2f Mh/s - %.2f Mh/s with step: %.2f\r\n"+
		"START CHARGE ..................................",
		pricePerMonthUSD, pricePerSecMh, limitChargeInUSD, PriceToString(limitChargeInSNM), pricePackMhInUSDPerMonth, mhashForETH, int(sumOrdersPerMonth),
		cfg.ChargeInterval.Start, cfg.ChargeInterval.Destination, cfg.Distances.StepForETH)
	os.Exit(1)
	for i := 0; i < int(sumOrdersPerMonth); i++ {
		if mhashForETH >= cfg.ChargeInterval.Destination {
			fmt.Printf("Charge is finished cause reached the limit %.2f Mh/s\r\n", cfg.ChargeInterval.Destination)
			break
		}
		pricePerSecPack := FloatToBigInt(mhashForETH * pricePerSecMh)
		log.Printf("Price :: %v\r\n", PriceToString(pricePerSecPack))
		mhashForETH, err = ChargeOrders(ctx, marketClient, symbETH, pricePerSecPack, cfg.Distances.StepForETH, mhashForETH, ethAddr, identityLvl)
		if err != nil {
			return fmt.Errorf("Cannot charging market! %v\r\n", err)
		}
	}
	log.Printf("Balance is not enough to make deals!\r\n")
	return nil
}

// Prepare price and Map depends on token symbol.
// Create orders to the market, until the budget is over.
func ChargeOrders(ctx context.Context, client sonm.MarketClient, symbol string, priceForHashPerSec *big.Int, step float64, buyMghash float64, ethAddr *sonm.EthAddress, identityLvl sonm.IdentityLevel) (float64, error) {
	requiredHashRate := uint64(buyMghash * hashes)
	benchmarks, err := getBenchmarksForSymbol(symbol, uint64(requiredHashRate))
	if err != nil {
		return 0, err
	}
	buyMghash, err = CreateOrderOnMarketStep(ctx, client, step, benchmarks, buyMghash, priceForHashPerSec, ethAddr, identityLvl)
	if err != nil {
		return 0, err
	}
	return buyMghash, nil
}

// Create order on market depends on token.
func CreateOrderOnMarketStep(ctx context.Context, market sonm.MarketClient, step float64, benchmarks map[string]uint64, buyMgHash float64, price *big.Int, ethAddr *sonm.EthAddress, identityLvl sonm.IdentityLevel) (float64, error) {
	actOrder, err := market.CreateOrder(ctx, &sonm.BidOrder{
		Tag:      "Connor bot",
		Duration: &sonm.Duration{},
		Price: &sonm.Price{
			PerSecond: sonm.NewBigInt(price),
		},
		Blacklist: ethAddr,
		Identity:  identityLvl,
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
		if err := records.SaveOrderIntoDB(&records.OrderDb{
			OrderID:         actOrder.GetId().Unwrap().Int64(),
			Price:           actOrder.GetPrice().Unwrap().Int64(),
			Hashrate:        actOrder.GetBenchmarks().GPUEthHashrate(),
			StartTime:       time.Now(),
			ButterflyEffect: int32(actOrder.GetOrderStatus()),
			ActualStep:      reBuyHash,
		}); err != nil {
			return 0, errors.WithMessage(err, "cannot save order to database")
		}
		log.Printf("Order created ==> ID: %v, Price: %v $, Hashrate: %v H/s \r\n",
			actOrder.GetId().Unwrap().Int64(),
			PriceToString(actOrder.GetPrice().Unwrap()),
			actOrder.GetBenchmarks().GPUEthHashrate())
		return reBuyHash, nil
	}
	return buyMgHash, nil
}

func GetPriceForTokenPerSec(token watchers.TokenWatcher) (float64, float64, error) {
	tokens, err := CollectTokensMiningProfit(token)
	if err != nil {
		return 0, 0, fmt.Errorf("Cannot calculate token prices: %v\r\n", err)
	}
	pricePerMonthUSD, err := GetProfitForTokenBySymbol(tokens, symbETH)
	if err != nil {
		return 0, 0, fmt.Errorf("Cannot get profit for tokens: %v\r\n", err)
	}
	pricePerSec := pricePerMonthUSD / (secsPerDay * daysPerMonth)
	return pricePerMonthUSD, pricePerSec, nil
}

// After charge orders
func TradeObserve(ctx context.Context, ethAddr *sonm.EthAddress, dealCli sonm.DealManagementClient, pool watchers.PoolWatcher, token watchers.TokenWatcher, marketClient sonm.MarketClient, taskCli sonm.TaskManagementClient, cfg *config.Config, identityLvl sonm.IdentityLevel) (error) {
	log.Printf("MODULE TRADE OBSERVE :: ")
	err := SaveActiveDealsIntoDB(ctx, dealCli)
	if err != nil {
		fmt.Printf("cannot save active deals intoDB %v\r\n", err)
	}

	_, pricePerSec, err := GetPriceForTokenPerSec(token)
	if err != nil {
		fmt.Printf("cannot get pricePerSec for token per sec %v\r\n", err)
	}

	actualPrice := FloatToBigInt(pricePerSec)
	log.Printf("Actual price per sec :: %v\r\n", PriceToString(actualPrice))

	deals, err := records.GetDealsFromDB()
	if err != nil {
		return fmt.Errorf("cannot get deals from DB %v\r\n", err)
	}
	orders, err := records.GetOrdersFromDB()
	if err != nil {
		return fmt.Errorf("cannot get orders from DB %v\r\n", err)
	}
	OrdersProfitTracking(ctx, actualPrice, orders, marketClient, identityLvl)
	ResponseActiveDeals(ctx, cfg, deals, marketClient, dealCli, taskCli, cfg.Images.Image, identityLvl)
	DealsProfitTracking(ctx, actualPrice, dealCli, marketClient, deals)
	return nil
}

func ReinvoiceOrder(ctx context.Context, m sonm.MarketClient, price *sonm.Price, bench map[string]uint64, tag string, identityLvl sonm.IdentityLevel) error {
	order, err := m.CreateOrder(ctx, &sonm.BidOrder{
		Duration: &sonm.Duration{Nanoseconds: 0},
		Price:    price,
		Tag:      tag,
		Identity: identityLvl,
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
	if err := records.SaveOrderIntoDB(&records.OrderDb{
		OrderID:         order.GetId().Unwrap().Int64(),
		Price:           order.GetPrice().Unwrap().Int64(),
		Hashrate:        order.GetBenchmarks().GPUEthHashrate(),
		StartTime:       time.Now(),
		ButterflyEffect: int32(OrderStatusReinvoice),
		ActualStep:      0, // step copy
	}); err != nil {
		return errors.Errorf("cannot save reinvoice order to DB %v\r\n", order.GetId().Unwrap().String(), err)
	}
	log.Printf("REINVOICE Order ===> %v created (descendant: %v), price: %v, hashrate: %v",
		order.GetId(), tag, order.GetPrice(), order.GetBenchmarks().GPUEthHashrate())
	return nil
}

// Get active deals --> for each active deal DEPLOY NEW CONTAINER --> Reinvoice order
func ResponseActiveDeals(ctx context.Context, cfg *config.Config, dealsDb []*records.DealDb, m sonm.MarketClient, dealCli sonm.DealManagementClient, tMng sonm.TaskManagementClient, imageMonero string, identityLvl sonm.IdentityLevel) error {
	for _, dealDb := range dealsDb {
		if dealDb.Status == 1 && dealDb.DeployStatus == 4 {
			getDealFromMarket, err := dealCli.Status(ctx, &sonm.BigInt{Abs: big.NewInt(dealDb.DealID).Bytes()})
			if err != nil {
				return fmt.Errorf("cannot get deal from Market %v\r\n", err)
			}
			deal := getDealFromMarket.Deal

			fmt.Printf("Deploying NEW CONTAINER ==> for dealDB %v (deal on market: %v\r\n)", dealDb.DealID, deal.GetId().Unwrap().String())
			task, err := DeployNewContainer(ctx, cfg, deal, tMng, imageMonero)
			if err != nil {
				records.UpdateDealInDB(deal.Id.Unwrap().Int64(), int32(DeployStatusNOTDEPLOYED))
				return fmt.Errorf("Cannot deploy new container from task %s\r\n", err)
			} else {
				records.UpdateDealInDB(deal.Id.Unwrap().Int64(), int32(DeployStatusDEPLOYED))
				fmt.Printf("New deployed task %v, for deal ID: %v\r\n", task.GetId(), deal.GetId().String())
			}

			bidOrder, err := m.GetOrderByID(ctx, &sonm.ID{Id: deal.GetBidID().String()})
			if err != nil {
				fmt.Printf("Cannot get order by Id: %v\r\n", err)
				return err
			}
			bench, err := GetBidBenchmarks(bidOrder)
			if err != nil {
				fmt.Printf("Cannot get benchmarks from bid Order : %v\r\n", bidOrder.Id.Unwrap().Int64())
				return err
			}
			ReinvoiceOrder(ctx, m, &sonm.Price{deal.GetPrice()}, bench, "Reinvoice", identityLvl)
		} else {
			fmt.Printf("For all received deals status :: DEPLOYED")
		}
	}
	return nil
}

func CmpChangeOfPrice(change float64, def float64) (int32, error) {
	if change >= 100+def {
		return 1, nil //increase
	} else if change < 100-def {
		return -1, nil
	}
	return 0, nil
}

func OrdersProfitTracking(ctx context.Context, actualPrice *big.Int, ordersDb []*records.OrderDb, m sonm.MarketClient, identityLvl sonm.IdentityLevel) error {
	log.Printf("MODULE :: Orders Profit Tracking")
	actualPriceCur := big.NewInt(actualPrice.Int64())

	for _, orderDb := range ordersDb {
		actualPriceClone := big.NewInt(0).Set(actualPriceCur)
		order, err := m.GetOrderByID(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
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
				change, err := GetChangePercent(pricePerSecForPack, orderPrice)
				if err != nil {
					return errors.Errorf("cannot get changes percent!", err)
				}
				log.Printf("Active Order Id: %v (price: %v), actual price for PACK: %v (for Mg/h :: %v)change percent: %.2f %%\r\n", orderDb.OrderID, PriceToString(orderPrice), PriceToString(pricePerSecForPack), PriceToString(actualPrice), change)
				commandPrice, err := CmpChangeOfPrice(change, 5)
				if commandPrice == 1 || commandPrice == -1 {
					bench, err := GetBidBenchmarks(order)
					if err != nil {
						fmt.Printf("Cannot get benchmarks from Order : %v\r\n", order.Id.Unwrap().Int64())
						return err
					}
					tag := strconv.Itoa(int(orderDb.OrderID))
					ReinvoiceOrder(ctx, m, &sonm.Price{sonm.NewBigInt(pricePerSecForPack)}, bench, "Reinvoice OldOrder: "+tag, identityLvl)
					m.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(orderDb.OrderID))})
				}
			} else {
				fmt.Printf("Order is not ACTIVE %v\r\n", order.Id)
				records.UpdateOrderInDB(orderDb.OrderID, 3)
			}
		}
	}
	return nil
}

// Pursue a profitable lvl of deal :: profitable price > deal price  => resale order with new price else do nothing
// Debug :: actualPrice == actualPriceForPack
func DealsProfitTracking(ctx context.Context, actualPrice *big.Int, dealClient sonm.DealManagementClient, marketClient sonm.MarketClient, dealsDb []*records.DealDb) error {
	log.Printf("DEALS PROFIT TRACKING ::")
	for _, d := range dealsDb {
		actualPriceClone, err := ClonePrice(actualPrice)
		if err != nil {
			return errors.Errorf("Cannot get clone price!", err)
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
		log.Printf("Deal id::%v (Bid:: %v) price :: %v, actual price for pack :: %v (hashes %v)\r\n", dealOnMarket.Deal.Id.String(), bidOrder.Id.String(), PriceToString(dealPrice), PriceToString(actualPriceForPack), pack)
		os.Exit(1)

		if actualPriceForPack.Cmp(dealPrice) >= 1 {
			changePercent, err := GetChangePercent(actualPriceForPack, dealPrice)
			if err != nil {
				return errors.Errorf("cannot get change percent from deal!", err)
			}
			log.Printf("Create CR ===> Active Deal Id: %v (price: %v), actual price for PACK: %v (for Mg/h :: %v) change percent: %.2f %%\r\n",
				dealOnMarket.Deal.Id.String(), PriceToString(dealPrice), PriceToString(actualPriceForPack), PriceToString(actualPrice), changePercent)
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
func CheckAndCancelOldOrders(ctx context.Context, m sonm.MarketClient, cfg *config.Config) () {
	ordersDb, err := records.GetOrdersFromDB()
	if err != nil {
		fmt.Printf("Cannot get orders from DB %v\r\n", err)
		os.Exit(1)
	}
	for _, o := range ordersDb {
		subtract := time.Now().AddDate(0, 0, - o.StartTime.Day()).Day()
		if subtract >= cfg.Sensitivity.SensitivityForOrders && subtract > 30 {
			fmt.Printf("Orders suspected of cancellation: : %v, passed time: %v\r\n", o.OrderID, subtract)
			//TODO: change status to "Cancelled"
			m.CancelOrder(ctx, &sonm.ID{Id: strconv.Itoa(int(o.OrderID)),})
			records.UpdateOrderInDB(o.OrderID, int32(OrderStatusCancelled))
		}
	}
}

func GetChangeRequest(ctx context.Context, dealCli sonm.DealManagementClient) () {
	//TODO : Create check ChangeRequest status (by approve)
}
