package connor

import (
	"context"
	"fmt"
	"github.com/sonm-io/core/connor/database"
	"github.com/sonm-io/core/connor/watchers"
	"github.com/sonm-io/core/proto"
	"log"
	"math/big"
	"strconv"
	"time"
)

type PoolModule struct {
	c *Connor
}

func NewPoolModules(c *Connor) *PoolModule {
	return &PoolModule{
		c: c,
	}
}

func (p *PoolModule) DeployNewContainer(ctx context.Context, cfg *Config, deal *sonm.Deal, image string) (*sonm.StartTaskReply, error) {
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
	reply, err := p.c.TaskClient.Start(ctx, startTaskRequest)
	if err != nil {
		log.Printf("Cannot create start task request %s", err)
		return nil, err
	}
	return reply, nil
}

func (p *PoolModule) SavePoolDataToDb(ctx context.Context, pool watchers.PoolWatcher, addr string) error {
	pool.Update(ctx)
	dataRH, err := pool.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}
	if len(dataRH.PoolWorkersData.Data) > 0 {
		for _, rh := range dataRH.PoolWorkersData.Data {
			poolBalance, err := strconv.ParseFloat(dataRH.PoolData.Data.Balance, 64)
			if err != nil {
				log.Printf("Cannot parse pooldata :: balance %v", err)
				return err
			}
			poolHashrate, err := strconv.ParseFloat(dataRH.PoolData.Data.Hashrate, 64)
			if err != nil {
				log.Printf("Cannot parse pooldata :: hashrate %v", err)
				return err
			}
			p.c.db.SavePoolIntoDB(&database.PoolDb{
				PoolId:                 addr,
				PoolBalance:            poolBalance,
				PoolHashrate:           poolHashrate,
				WorkerID:               rh.Worker,
				WorkerReportedHashrate: rh.Hashrate,
				WorkerAvgHashrate:      0,
				BadGuy:                 0,
				Iterations:             0,
				TimeStart:              time.Now(),
				TimeUpdate:             time.Now(),
			})
		}
	} else {
		log.Printf("Cannot detect the workers in pool :: %v", addr)
	}
	return nil
}

func (p *PoolModule) UpdateRHPoolData(ctx context.Context, poolRHData watchers.PoolWatcher, addr string) error {
	poolRHData.Update(ctx)
	dataRH, err := poolRHData.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data RH  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.PoolWorkersData.Data {
		p.c.db.UpdateReportedHashratePoolDB(rh.Worker, rh.Hashrate, time.Now())
	}
	return nil
}

func (p *PoolModule) UpdateAvgPoolData(ctx context.Context, poolAvgData watchers.PoolWatcher, addr string) error {
	poolAvgData.Update(ctx)
	dataRH, err := poolAvgData.GetData(addr)
	if err != nil {
		log.Printf("Cannot get data AvgPool  --> %v\r\n", err)
		return err
	}

	for _, rh := range dataRH.PoolWorkersData.Data {
		p.c.db.UpdateAvgPoolDB(rh.Worker, rh.Hashrate, time.Now())
	}
	return nil
}

func (p *PoolModule) UpdateMainData(ctx context.Context, poolRHData watchers.PoolWatcher, poolAvgData watchers.PoolWatcher, addr string) error {
	workers, err := p.c.db.GetWorkersFromDB()
	if err != nil {
		log.Printf("cannot get worker from pool DB")
		return err
	}
	for _, w := range workers {
		if w.Iterations < numberOfIterationsForH1 && w.BadGuy < numberOfLives {
			if err = p.UpdateRHPoolData(ctx, poolRHData, addr); err != nil {
				log.Printf("cannot update RH pool data!")
				return err
			}
		} else {
			if err = p.UpdateAvgPoolData(ctx, poolAvgData, addr); err != nil {
				log.Printf("cannot update AVG pool data!")
				return err
			}
		}
	}
	return nil
}

func (p *PoolModule) PoolHashrateTracking(ctx context.Context, poolRHData watchers.PoolWatcher, poolAvgData watchers.PoolWatcher, addr string) error {
	p.UpdateMainData(ctx, poolRHData, poolAvgData, addr)
	workers, err := p.c.db.GetWorkersFromDB()
	if err != nil {
		log.Printf("cannot get worker from pool DB")
		return err
	}

	for _, w := range workers {
		wId, err := strconv.Atoi(w.WorkerID)
		if err != nil {
			return fmt.Errorf("cannot atoi returns %v", err)
		}

		dealID, err := p.c.DealClient.Status(ctx, &sonm.BigInt{Abs: big.NewInt(int64(wId)).Bytes()})
		if err != nil {
			log.Printf("Cannot get deal from market %v\r\n", w.WorkerID)
			return err
		}
		bidOrder, err := p.c.Market.GetOrderByID(ctx, &sonm.ID{Id: dealID.Deal.BidID.Unwrap().String()})
		if err != nil {
			log.Printf("cannot get order from market by ID")
			return err
		}

		iteration := int32(w.Iterations + 1)
		bidHashrate := bidOrder.GetBenchmarks().GPUEthHashrate()

		// FIXME: change value BadGuy in Db
		if w.BadGuy > 5 {
			continue
		}

		if iteration < numberOfIterationsForH1 {
			workerReportedHashrate := uint64(w.WorkerReportedHashrate * hashes)
			changeReportedWorker := 100 - ((workerReportedHashrate * 100) / bidHashrate)
			if changeReportedWorker >= uint64(p.c.cfg.Sensitivity.WorkerLimitChangePercent) {
				log.Printf("ID :: %v ==> wRH %v < deal (bid) hashrate %v ==> PID! \r\n", w.WorkerID, workerReportedHashrate, bidHashrate)
				if w.BadGuy < numberOfLives {
					newStatus := w.BadGuy + 1
					p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, newStatus, time.Now())
					//TODO: make a deviation of N percent for Bad guy status
				} else {
					p.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
						Id:            dealID.Deal.Id,
						BlacklistType: 1,
					})
					killed := 6
					p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, int32(killed), time.Now())
					log.Printf("This deal is destroyed (bad status in Pool) : %v!\r\n", dealID.Deal.Id)
				}
			} else if changeReportedWorker >= 20 {
				p.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
					Id:            dealID.Deal.Id,
					BlacklistType: 1,
				})
				killed := 6
				p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, int32(killed), time.Now())
				log.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
			}
		} else {
			log.Printf("Iteration for worker :: %v more than 4 == > get Avg Data", w.WorkerID)
			workerAvgHashrate := uint64(w.WorkerAvgHashrate * hashes)
			changeAvgWorker := 100 - ((workerAvgHashrate * 100) / bidHashrate)
			if changeAvgWorker >= uint64(p.c.cfg.Sensitivity.WorkerLimitChangePercent) {
				log.Printf("ID :: %v ==> wRH %v < deal (bid) hashrate %v ==> PID! \r\n", w.WorkerID, workerAvgHashrate, bidHashrate)
				if w.BadGuy < 5 {
					newStatus := w.BadGuy + 1
					p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, newStatus, time.Now())
				} else {
					p.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
						Id:            dealID.Deal.Id,
						BlacklistType: 1,
					})
					killed := 6
					p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, int32(killed), time.Now())
					log.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
				}
			} else if changeAvgWorker >= 20 {
				p.c.DealClient.Finish(ctx, &sonm.DealFinishRequest{
					Id:            dealID.Deal.Id,
					BlacklistType: 1,
				})
				killed := 6
				p.c.db.UpdateWorkerStatusInPoolDB(w.WorkerID, int32(killed), time.Now())
				log.Printf("This deal is destroyed (Pidor more than 5) : %v!\r\n", dealID.Deal.Id)
			}
		}
		p.c.db.UpdateIterationPoolDB(w.WorkerID, iteration)
	}
	return nil
}
