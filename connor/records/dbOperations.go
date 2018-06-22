package records

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	deals = `
--	DROP TABLE IF EXISTS deals;
	CREATE TABLE IF NOT EXISTS DEALS (
		ID INT NOT NULL UNIQUE,
		STATUS TEXT NOT NULL ,
		PRICE INT NOT NULL,
		ASK_ID INT NOT NULL,
		BID_ID INT NOT NULL,
		DEPLOY_STATUS INT NOT NULL,
		START_TIME DATETIME NOT NULL,
		LIFETIME DATETIME NOT NULL
	);
`
	orders = `
	CREATE TABLE IF NOT EXISTS ORDERS(
		ID INT NOT NULL UNIQUE,
		PRICE INT NOT NULL,
		HASHRATE INT NOT NULL,
		START_TIME DATETIME NOT NULL,
		BUTTERFLY_EFFECT INT NOT NULL, --fact of transformation into a deal
		ACTUAL_STEP FLOAT NOT NULL
	);
`

	tokens = `
	CREATE TABLE IF NOT EXISTS TOKENS (
		TOKEN_ID TEXT NOT NULL,
		NAME TEXT NOT NULL,
		USD_PRICE FLOAT NOT NULL,
		NET_HASHES_SEC FLOAT NOT NULL,
		BLOCK_TIME INT NOT NULL,
		BLOCK_REWARD FLOAT NOT NULL,
		PROFIT_PER_MONTH_USD FLOAT NOT NULL,
		DATE_TIME DATETIME NOT NULL
	);
`
	pools = `
	CREATE TABLE IF NOT EXISTS POOLS (
		POOL_ID TEXT NOT NULL,
		POOL_BALANCE FLOAT NOT NULL,
		POOL_HASH FLOAT NOT NULL,
		W_ID TEXT NOT NULL UNIQUE,
		W_REP_HASH FLOAT NOT NULL,
		W_AVG_HASH FLOAT NOT NULL,
		BAD_GUY INT NOT NULL,
		ITERATIONS INT NOT NULL, 
		TIME_START DATETIME NOT NULL,
		TIME_UPDATE DATETIME NOT NULL
	);
`
	driver     = "sqlite3"
	dataSource = "insonmnia/arbBot/test.sq3"
)

var db *sqlx.DB

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

func CreateOrderDB() {
	db, err := sqlx.Connect(driver, dataSource)
	if err != nil {
		log.Fatalln(err)
	}
	_, err = db.Exec(orders)
	if err != nil {
		log.Fatalln(err)
	}
}

func SaveDealIntoDB(deal *DealDb) error {
	db, err := sqlx.Connect(driver, dataSource)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	_, err = db.Exec(deals)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	tx := db.MustBegin()
	tx.NamedExec("INSERT INTO DEALS(ID, STATUS, PRICE, ASK_ID, BID_ID, DEPLOY_STATUS, START_TIME, LIFETIME ) VALUES (:ID, :STATUS, :PRICE, :ASK_ID, :BID_ID, :DEPLOY_STATUS, :START_TIME, :LIFETIME )",
		deal)
	tx.Commit()
	return nil
}
func SaveOrderIntoDB(order *OrderDb) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	_, err = conn.Exec(orders)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	tx := conn.MustBegin()
	tx.NamedExec("INSERT INTO ORDERS(ID, PRICE, HASHRATE, START_TIME, BUTTERFLY_EFFECT, ACTUAL_STEP) VALUES (:ID, :PRICE, :HASHRATE, :START_TIME, :BUTTERFLY_EFFECT, :ACTUAL_STEP)",
		order)
	tx.Commit()
	return nil
}
func SaveProfitToken(token *TokenDb) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	_, err = conn.Exec(tokens)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	tx := conn.MustBegin()
	tx.NamedExec("INSERT INTO TOKENS (TOKEN_ID, NAME, USD_PRICE, NET_HASHES_SEC, BLOCK_TIME, BLOCK_REWARD,  PROFIT_PER_MONTH_USD, DATE_TIME) VALUES (:TOKEN_ID, :NAME, :USD_PRICE, :NET_HASHES_SEC, :BLOCK_TIME, :BLOCK_REWARD, :PROFIT_PER_MONTH_USD, :DATE_TIME)",
		token)
	tx.Commit()
	return nil
}
func SavePoolIntoDB(pool *PoolDb) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	_, err = conn.Exec(pools)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	tx := conn.MustBegin()
	tx.NamedExec("INSERT INTO POOLS (POOL_ID, POOL_BALANCE, POOL_HASH, W_ID, W_REP_HASH,W_AVG_HASH, BAD_GUY, ITERATIONS, TIME_START, TIME_UPDATE ) VALUES (:POOL_ID, :POOL_BALANCE, :POOL_HASH,  :W_ID, :W_REP_HASH, :W_AVG_HASH, :BAD_GUY, :ITERATIONS, :TIME_START, :TIME_UPDATE)",
		pool)
	tx.Commit()
	return nil
}

func UpdateOrderInDB(id int64, bfly int32) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE ORDERS SET BUTTERFLY_EFFECT = ?	 WHERE ID = ?", bfly, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}
func UpdateDealInDB(id int64, deployStatus int32) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE DEALS SET DEPLOY_STATUS = ?	 WHERE ID = ?", deployStatus, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}
func UpdateStatusPoolDB(id string, badGuy int32) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE POOLS SET BAD_GUY = ? WHERE W_ID = ?", badGuy, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}
func UpdateRHPoolDB(id string, reportedHashrate float64) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE POOLS SET W_REP_HASH = ? WHERE W_ID = ?", reportedHashrate, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}
func UpdateAvgPoolDB(id string, avgHashrate float64) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE POOLS SET W_AVG_HASH = ? WHERE W_ID = ?", avgHashrate, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}
func UpdateIterationPoolDB(id string, iteration int32) error {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return err
	}
	result, err := conn.Exec("UPDATE POOLS SET ITERATIONS = ? WHERE W_ID = ?", iteration, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	fmt.Printf("rows: %v\r\n", rowsAffected)
	return nil
}

func GetCountFromDB() (counts int, err error) {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return 0, err
	}
	rows, err := conn.Query("SELECT COUNT(ID) AS count FROM ORDERS")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}
	for rows.Next() {
		var count int
		err = rows.Scan(&count)
		if err != nil {
			log.Fatal(err)
		}
		return count, nil
	}
	return 0, fmt.Errorf("")
}

func GetLastActualStepFromDb() (float64, error) {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return 0, err
	}
	rows, err := conn.Query("SELECT MAX(ACTUAL_STEP) AS max FROM ORDERS WHERE BUTTERFLY_EFFECT = 2")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}

	for rows.Next() {
		var max float64
		err = rows.Scan(&max)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		return max, nil
	}
	return 0, nil
}
func GetOrdersFromDB() ([]*OrderDb, error) {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query("SELECT * FROM ORDERS")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	orders := make([]*OrderDb, 0)
	for rows.Next() {
		order := new(OrderDb)
		err := rows.Scan(&order.OrderID, &order.Price, &order.Hashrate, &order.StartTime, &order.ButterflyEffect, &order.ActualStep)
		if err != nil {
			log.Fatal(err)
		}
		orders = append(orders, order)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return orders, err
}
func GetDealsFromDB() ([]*DealDb, error) {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query("SELECT * FROM DEALS")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	deals := make([]*DealDb, 0)
	for rows.Next() {
		deal := new(DealDb)
		err := rows.Scan(&deal.DealID, &deal.Status, &deal.Price, &deal.AskId, &deal.BidID, &deal.DeployStatus, &deal.StartTime, &deal.LifeTime)
		if err != nil {
			log.Fatal(err)
		}
		deals = append(deals, deal)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return deals, err
}
func GetWorkersFromDB() ([]*PoolDb, error) {
	conn, err := NewDatabaseConnect(driver, dataSource)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query("SELECT * FROM POOLS")
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	workers := make([]*PoolDb, 0)
	for rows.Next() {
		worker := new(PoolDb)
		err := rows.Scan(&worker.PoolId, &worker.PoolBalance, &worker.PoolHashrate, &worker.WorkerID, &worker.WorkerReportedHashrate, &worker.WorkerAvgHashrate, &worker.BadGuy, &worker.Iterations, &worker.TimeStart, &worker.TimeUpdate)
		if err != nil {
			log.Fatal(err)
		}
		workers = append(workers, worker)
	}
	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	return workers, err
}
func NewDatabaseConnect(driver, dataSource string) (*sqlx.DB, error) {
	var err error
	if db == nil {
		db, err = sqlx.Connect(driver, dataSource)
		if err != nil {
			return nil, err
		}
	}
	return db, nil
}
