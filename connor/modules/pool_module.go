package modules

import (
	"context"
	"fmt"
	"github.com/sonm-io/core/connor/config"
	"github.com/sonm-io/core/connor/records"
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"
)

const (
	EthPool = "stratum+tcp://eth-eu1.nanopool.org:9999"
)

func DeployNewContainer(ctx context.Context, cfg *config.Config, deal *sonm.Deal, n sonm.TaskManagementClient, image string) (*sonm.StartTaskReply, error) {
	fmt.Printf("deal = %#v, client = %#v, image = %s\n", deal, n, image)
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
	reply, err := n.Start(ctx, startTaskRequest)
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

func SavePoolDataToDb(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}
	for _, rh := range dataRH.Data {
		fmt.Printf("w:: %v, data :: %v\r\n", rh.Worker, rh.Hashrate)
		// TODO: replace time + pool balance + pool hashrate
		records.SavePoolIntoDB(&records.PoolDb{
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

func UpdateRHPoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.Data {
		// TODO: replace time
		records.UpdateRHPoolDB(rh.Worker, rh.Hashrate)
	}
	return nil
}

func UpdateAvgPoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.Data {
		// TODO: replace time
		records.UpdateAvgPoolDB(rh.Worker, rh.Hashrate)
	}
	return nil
}

func UpdatePoolData(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	workers, err := records.GetWorkersFromDB()
	if err != nil {
		fmt.Printf("cannot get worker from pool DB")
		return err
	}
	// TODO: Bad guy detected :: if iterations < 4 && badguy <5
	for _, w := range workers {
		if w.Iterations < 4 {
			if err = UpdateRHPoolData(ctx, pool, addr); err != nil {
				log.Printf("cannot update RH pool data!")
				return err
			}
		} else {
			if err = UpdateAvgPoolData(ctx, pool, addr); err != nil {
				log.Printf("cannot update AVG pool data!")
				return err
			}
		}
	}
	return nil
}

func PoolTracking(ctx context.Context, dealCli sonm.DealManagementClient, marketCli sonm.MarketClient, pool watchers.PoolWatcher, addr string) error {
	UpdatePoolData(ctx, pool, addr)
	workers, err := records.GetWorkersFromDB()
	if err != nil {
		fmt.Printf("cannot get worker from pool DB")
		return err
	}

	for _, w := range workers {
		wId, err := strconv.Atoi(w.WorkerID)

		dealID, err := dealCli.Status(ctx, &sonm.BigInt{Abs: big.NewInt(int64(wId)).Bytes()})
		if err != nil {
			fmt.Printf("Cannot get deal from market %v\r\n", w.WorkerID)
			return err
		}
		dealCli.Finish(ctx, &sonm.DealFinishRequest{
			Id:            dealID.Deal.Id,
			BlacklistType: 1,
		})
		os.Exit(1)
		bidOrder, err := marketCli.GetOrderByID(ctx, &sonm.ID{Id: dealID.Deal.BidID.Unwrap().String()})
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
					records.UpdateStatusPoolDB(w.WorkerID, newStatus)
				} else {
					dealCli.Finish(ctx, &sonm.DealFinishRequest{
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
					records.UpdateStatusPoolDB(w.WorkerID, newStatus)
				} else {
					dealCli.Finish(ctx, &sonm.DealFinishRequest{
						Id:            dealID.Deal.Id,
						BlacklistType: 1,
					})
					fmt.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
				}
			}
		}
		records.UpdateIterationPoolDB(w.WorkerID, iteration)
	}
	return nil
}
