package database

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"time"
)

type Database struct {
	connect *sqlx.DB
}

func (d *Database) DeleteOrder(id int64) error {
	_, err := d.connect.Exec("DELETE FROM ORDERS WHERE ID=?", id)
	if err != nil {
		return err
	}
	return nil

}

func NewDatabaseConnect(driver, dataSource string) (*Database, error) {
	var err error
	d := &Database{}
	d.connect, err = sqlx.Connect(driver, dataSource)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func (d *Database) CreateOrderDB() error {
	_, err := d.connect.Exec(orders)
	if err != nil {
		return err
	}
	return nil
}
func (d *Database) CreatePoolDB() error {
	_, err := d.connect.Exec(pools)
	if err != nil {
		return err
	}
	// TODO: check error in result
	return nil
}
func (d *Database) CreateDealsDB() error {
	_, err := d.connect.Exec(deals)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) CreateBlacklistDB() error {
	_, err := d.connect.Exec(blacklist)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) SaveDealIntoDB(deal *DealDb) error {
	_, err := d.connect.Exec(deals)
	if err != nil {
		return err
	}
	tx := d.connect.MustBegin()
	tx.NamedExec(insertDeals, deal)
	tx.Commit()
	return nil
}

func (d *Database) SaveBlacklistIntoDB(blacklistData *BlackListDb) error {
	_, err := d.connect.Exec(blacklist)
	if err != nil {
		return err
	}
	tx := d.connect.MustBegin()
	tx.NamedExec(insertBlackList, blacklistData)
	tx.Commit()
	return nil
}

func (d *Database) SaveOrderIntoDB(order *OrderDb) error {
	_, err := d.connect.Exec(orders)
	if err != nil {
		return err
	}
	tx := d.connect.MustBegin()
	tx.NamedExec(insertOrders, order)
	tx.Commit()
	return nil
}
func (d *Database) SaveProfitToken(token *TokenDb) error {
	_, err := d.connect.Exec(tokens)
	if err != nil {
		return err
	}
	tx := d.connect.MustBegin()
	tx.NamedExec(insertToken, token)
	tx.Commit()
	return nil
}
func (d *Database) SavePoolIntoDB(pool *PoolDb) error {
	_, err := d.connect.Exec(pools)
	if err != nil {
		return err
	}
	tx := d.connect.MustBegin()
	tx.NamedExec(insertPools, pool)
	tx.Commit()
	return nil
}

func (d *Database) UpdateOrderInDB(id int64, bfly int32) error {
	result, err := d.connect.Exec(updateOrders, bfly, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
func (d *Database) UpdateDealInDB(id int64, deployStatus int32) error {
	result, err := d.connect.Exec(updateDeals, deployStatus, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
func (d *Database) UpdateWorkerStatusInPoolDB(id string, badGuy int32, timeUpdate time.Time) error {
	result, err := d.connect.Exec(updateStatusPoolDB, badGuy, timeUpdate, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
func (d *Database) UpdateReportedHashratePoolDB(id string, reportedHashrate float64, timeUpdate time.Time) error {
	result, err := d.connect.Exec(updateReportedHashrate, reportedHashrate, timeUpdate, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}
func (d *Database) UpdateAvgPoolDB(id string, avgHashrate float64, timeUpdate time.Time) error {
	result, err := d.connect.Exec(updateAvgPool, avgHashrate, timeUpdate, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) UpdateBanStatusBlackListDB(masterID string, banStatus int32) error {
	result, err := d.connect.Exec(updateBlackList, banStatus, masterID)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) UpdateIterationPoolDB(id string, iteration int32) error {
	result, err := d.connect.Exec(updateIterationPool, iteration, id)
	if err != nil {
		return err
	}
	_, err = result.RowsAffected()
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetCountFromDB() (counts int, err error) {
	rows, err := d.connect.Query(getCountFromDb)
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
func (d *Database) GetLastActualStepFromDb() (float64, error) {
	rows, err := d.connect.Query(getLastActualStep)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var max float64
		err = rows.Scan(&max)
		if err != nil {
			log.Fatal(err)
		}
		return max, nil
	}
	return 0, nil
}
func (d *Database) GetOrdersFromDB() ([]*OrderDb, error) {
	rows, err := d.connect.Query(getOrders)
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
func (d *Database) GetBlacklistFromDb(failSupplierID string) (string, error) {
	rows, err := d.connect.Query(getSupplierIDFromBlackList, failSupplierID)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var failSupplier string
		err = rows.Scan(&failSupplier)
		if err != nil {
			log.Fatal(err)
		}
		return failSupplier, nil
	}
	return "already in Blacklist", nil
}

func (d *Database) GetWorkerFromPoolDb(dealID string) (string, error) {
	rows, err := d.connect.Query(getWorkerIDFromPool, dealID)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var dealID string
		err = rows.Scan(&dealID)
		if err != nil {
			log.Fatal(err)
		}
		return dealID, nil
	}
	return "already in Pool!", nil
}

func (d *Database) GetCountFailSupplierFromDb(masterID string) (int64, error) {
	rows, err := d.connect.Query(getCountSupplierIDFromBlackList, masterID)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var failSupplier int64
		err = rows.Scan(&failSupplier)
		if err != nil {
			log.Fatal(err)
		}
		return failSupplier, nil
	}
	return 0, nil
}

func (d *Database) GetDealsFromDB() ([]*DealDb, error) {
	rows, err := d.connect.Query(getDeals)
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
func (d *Database) GetWorkersFromDB() ([]*PoolDb, error) {
	rows, err := d.connect.Query(getWorkersFromPool)
	defer rows.Close()
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	workers := make([]*PoolDb, 0)
	for rows.Next() {
		worker := new(PoolDb)
		err := rows.Scan(&worker.DealID, &worker.PoolId, &worker.WorkerReportedHashrate,
			&worker.WorkerAvgHashrate, &worker.BadGuy, &worker.Iterations, &worker.TimeStart, &worker.TimeUpdate)
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
