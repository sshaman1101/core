package connor

import (
	"context"
	"fmt"
	"github.com/sonm-io/core/connor/database"
	"github.com/sonm-io/core/proto"
	"log"
	"math/big"
	"strconv"
	"time"
	"github.com/sonm-io/core/connor/watchers"
)

/*
	This file for SONM only.
*/

//Tracking hashrate with using Connor's blacklist.
// Get data for 1 hour and another time => Detecting deviation.
func (p *PoolModule) AdvancedPoolHashrateTracking(ctx context.Context, reportedPool watchers.PoolWatcher, avgPool watchers.PoolWatcher) error {
	workers, err := p.c.db.GetWorkersFromDB()
	if err != nil {
		log.Printf("cannot get worker from pool DB")
		return err
	}
	for _, w := range workers {
		// FIXME: change value BadGuy in Db
		if w.BadGuy > 5 {
			continue
		}
		iteration := int32(w.Iterations + 1)
		log.Printf("Iteration :: %v", iteration)
		wId, err := strconv.Atoi(w.DealID)
		if err != nil {
			return fmt.Errorf("cannot atoi returns %v", err)
		}

		dealInfo, err := p.c.DealClient.Status(ctx, sonm.NewBigInt(big.NewInt(0).SetInt64(int64(wId))))
		if err != nil {
			log.Printf("Cannot get deal from market %v\r\n", w.DealID)
			return err
		}
		bidHashrate, err := p.ReturnBidHashrateForDeal(ctx, dealInfo)
		if err != nil {
			return err
		}

		if iteration < numberOfIterationsForH1 {
			if err = p.UpdateRHPoolData(ctx, reportedPool, p.c.cfg.PoolAddress.EthPoolAddr); err != nil {
				return err
			}
			workerReportedHashrate := uint64(w.WorkerReportedHashrate * hashes)
			changePercentRHWorker := float64(100 - float64(float64(workerReportedHashrate*100)/float64(bidHashrate)))
			if err = p.AdvancedDetectingDeviation(ctx, changePercentRHWorker, w, dealInfo); err != nil {
				return err
			}
		} else {
			workerAvgHashrate := uint64(w.WorkerAvgHashrate * hashes)
			p.UpdateAvgPoolData(ctx, avgPool, p.c.cfg.PoolAddress.EthPoolAddr+"/1")
			changeAvgWorker := float64(100 - float64(float64(workerAvgHashrate*100)/float64(bidHashrate)))
			if err = p.AdvancedDetectingDeviation(ctx, changeAvgWorker, w, dealInfo); err != nil {
				return err
			}
		}
		p.c.db.UpdateIterationPoolDB(iteration, w.DealID)
	}
	return nil

}

//Detects the percentage of deviation of the hashrate and save SupplierID (by MasterID) to Connor's blacklist .
func (p *PoolModule) AdvancedDetectingDeviation(ctx context.Context, changePercentDeviationWorker float64, worker *database.PoolDb, dealInfo *sonm.DealInfoReply) error {
	if changePercentDeviationWorker >= p.c.cfg.Sensitivity.WorkerLimitChangePercent {
		p.SendToConnorBlackList(ctx, dealInfo)

	} else if changePercentDeviationWorker >= 20 {
		if err := p.DestroyDeal(ctx, dealInfo); err != nil {
			return err
		}
		p.c.db.UpdateWorkerStatusInPoolDB(worker.DealID, int32(BanStatusWorkerInPool), time.Now())
	}
	return nil
}
