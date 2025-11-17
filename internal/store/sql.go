package store

import (
	"context"
	"log"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/skdiver33/metrics-collector/internal/misc"
	"github.com/skdiver33/metrics-collector/models"
)

type SQLStorage struct {
	config *SQLStorageConfig
	db     *sql.DB
}

type SQLStorageConfig struct {
	DBAddress string
}

func NewSQLStorageConfig(address string) *SQLStorageConfig {
	storageConfig := SQLStorageConfig{DBAddress: address}
	return &storageConfig
}

func NewSQLStorage(address string) (*SQLStorage, error) {
	newStorage := SQLStorage{}
	newStorage.config = NewSQLStorageConfig(address)
	err := newStorage.InitializeConnection()
	if err != nil {
		return nil, err
	}

	err = newStorage.InitializeDB()
	if err != nil {
		return nil, err
	}
	return &newStorage, nil
}

func (storage *SQLStorage) InitializeConnection() error {
	db, err := sql.Open("pgx", storage.config.DBAddress)
	if err != nil {
		return err
	}
	storage.db = db
	return nil
}

func (storage *SQLStorage) InitializeDB() error {
	createTableString := "CREATE TABLE IF NOT EXISTS metrics (id VARCHAR(100) PRIMARY KEY, type VARCHAR(100) NOT NULL, delta bigint NULL, value double precision NULL);"

	requestFunc := func() error {
		_, err := storage.db.Exec(createTableString)
		return err
	}
	baseCtx := context.Background()
	err := misc.RetriableErrorHandler(baseCtx, requestFunc)

	if err != nil {
		storage.CloseConnection()
		return err
	}

	for _, metricsName := range models.GaugeMetricsNames {
		val := 0.0
		metrics := models.Metrics{ID: metricsName, MType: models.Gauge, Value: &val}
		if err := storage.AddMetrics(baseCtx, metrics); err != nil {
			log.Println("Error initialize storage. Cannot add metrics in DB.")
			storage.CloseConnection()
			return err
		}
	}
	for _, metricsName := range models.CounterMetricsNames {
		delta := int64(0)
		metrics := models.Metrics{ID: metricsName, MType: models.Counter, Delta: &delta}
		if err := storage.AddMetrics(baseCtx, metrics); err != nil {
			log.Println("Error initialize storage. Cannot add metrics in DB.")
			storage.CloseConnection()
			return err
		}
	}
	return nil

}

func (storage *SQLStorage) CloseConnection() {
	storage.db.Close()
}

func (storage *SQLStorage) AddMetrics(ctx context.Context, metrics models.Metrics) error {
	ctxWithTO, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	retryFunc := func() error {
		_, err := storage.db.ExecContext(ctxWithTO, "INSERT INTO metrics (id, type,delta,value) VALUES ($1, $2,$3,$4) ON CONFLICT DO NOTHING", metrics.ID, metrics.MType, metrics.Delta, metrics.Value)
		return err
	}
	err := misc.RetriableErrorHandler(ctxWithTO, retryFunc)

	if err != nil {
		storage.CloseConnection()
		return err
	}
	return nil
}
func (storage *SQLStorage) UpdateMetrics(ctx context.Context, metrics models.Metrics) error {
	ctxWithTO, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	retryFunc := func() error {
		_, err := storage.db.ExecContext(ctxWithTO, "UPDATE metrics SET delta = $1,value = $2 WHERE id = $3", metrics.Delta, metrics.Value, metrics.ID)
		return err
	}
	err := misc.RetriableErrorHandler(ctxWithTO, retryFunc)
	if err != nil {
		storage.CloseConnection()
		return err
	}
	return nil
}
func (storage *SQLStorage) GetMetrics(ctx context.Context, metricsName string) (models.Metrics, error) {
	ctxWithTO, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	metrics := models.Metrics{}
	row := storage.db.QueryRowContext(ctxWithTO, "SELECT * FROM metrics WHERE id=$1", metricsName)
	err := row.Scan(&metrics.ID, &metrics.MType, &metrics.Delta, &metrics.Value)
	if err != nil {
		return metrics, err
	}
	return metrics, nil
}

func (storage *SQLStorage) GetAllMetrics(ctx context.Context) *[]models.Metrics {
	ctxWithTO, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := storage.db.QueryContext(ctxWithTO, "SELECT * FROM metrics")
	if err != nil {
		log.Printf("error get all metrics %s", err.Error())
		return nil
	}
	defer rows.Close()

	result := make([]models.Metrics, 0)

	for rows.Next() {
		newMetrics := models.Metrics{}
		if err := rows.Scan(&newMetrics.ID, &newMetrics.MType, &newMetrics.Delta, &newMetrics.Value); err != nil {
			log.Printf("error parse result from DB %s", err.Error())
			return nil
		}
		result = append(result, newMetrics)
	}

	err = rows.Err()
	if err != nil {
		log.Printf("error load data from DB %s", err.Error())
		return nil
	}

	return &result
}

func (storage *SQLStorage) UpdateAllMetrics(ctx context.Context, allMetrics *[]models.Metrics) error {
	ctxWithTO, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := storage.db.Begin()
	if err != nil {
		log.Print("error create transaction")
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctxWithTO, "INSERT INTO metrics (id, type,delta,value) VALUES ($1, $2,$3,$4) ON CONFLICT (id) DO UPDATE SET delta=(SELECT delta FROM metrics WHERE id = $1)+$3,value=$4")
	if err != nil {
		log.Print("error prepere update statement")
		return err
	}

	for _, metrics := range *allMetrics {
		if _, err := stmt.ExecContext(ctxWithTO, metrics.ID, metrics.MType, metrics.Delta, metrics.Value); err != nil {
			log.Print("Error insert update request in statement")
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		log.Print("error commit update transaction")
		return err
	}

	return nil
}
func (storage *SQLStorage) CreateDBDump(ctx context.Context, filename string) {

}
func (storage *SQLStorage) RestoreDBDump(ctx context.Context, filename string) {

}

func (storage *SQLStorage) PingDB(ctx context.Context) error {
	ctxWithTO, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	if err := storage.db.PingContext(ctxWithTO); err != nil {
		return misc.NewRetrialableError(err)
	}
	return nil
}
