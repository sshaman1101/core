package database

import "time"

type PoolDb struct {
	PoolId                 string    `db:"POOL_ID"`
	PoolBalance            float64   `db:"POOL_BALANCE"`
	PoolHashrate           float64   `db:"POOL_HASH"`
	WorkerID               string    `db:"W_ID"`
	WorkerReportedHashrate float64   `db:"W_REP_HASH"`
	WorkerAvgHashrate      float64   `db:"W_AVG_HASH"`
	BadGuy                 int32     `db:"BAD_GUY"`
	Iterations             int64     `db:"ITERATIONS"`
	TimeStart              time.Time `db:"TIME_START"`
	TimeUpdate             time.Time `db:"TIME_UPDATE"`
}

type TokenDb struct {
	ID              string    `db:"TOKEN_ID"`
	Name            string    `db:"NAME"`
	UsdPrice        float64   `db:"USD_PRICE"`
	NetHashesPerSec float64   `db:"NET_HASHES_SEC"`
	BlockTime       int       `db:"BLOCK_TIME"`
	BlockReward     float64   `db:"BLOCK_REWARD"`
	ProfitPerMonth  float64   `db:"PROFIT_PER_MONTH_USD"`
	ProfitSNM       float64   `db:"PROFIT_SNM"`
	DateTime        time.Time `db:"DATE_TIME"`
}

type DealDb struct {
	DealID       int64     `db:"ID"`
	Status       int32     `db:"STATUS"`
	Price        int64     `db:"PRICE"`
	AskId        int64     `db:"ASK_ID"`
	BidID        int64     `db:"BID_ID"`
	DeployStatus int32     `db:"DEPLOY_STATUS"`
	StartTime    time.Time `db:"START_TIME"`
	LifeTime     time.Time `db:"LIFETIME"`
}

type OrderDb struct {
	OrderID         int64     `db:"ID"`
	Price           int64     `db:"PRICE"`
	Hashrate        uint64    `db:"HASHRATE"`
	StartTime       time.Time `db:"START_TIME"`
	ButterflyEffect int32     `db:"BUTTERFLY_EFFECT"`
	ActualStep      float64   `db:"ACTUAL_STEP"`
}
