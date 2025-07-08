package store

import (
	"context"
	"flag"
	"os"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/skdiver33/metrics-collector/models"
)

type SqlStorage struct {
	config *SqlStorageConfig
	db     *sql.DB
}

type SqlStorageConfig struct {
	DBAddress string
	//dbUser     string
	//dbPassword string
	//dbName     string
}

func NewSqlStorageConfig() *SqlStorageConfig {
	storageConfig := SqlStorageConfig{}

	storageFlags := flag.NewFlagSet("Sql storage config flags", flag.ContinueOnError)
	storageFlags.StringVar(&storageConfig.DBAddress, "d", "host=192.168.1.45 user=bob password=secret dbname=metrics sslmode=disable", "adress for connect DB. default 192.168.1.45:5432")
	storageFlags.Parse(os.Args[1:])

	envDBAddr, ok := os.LookupEnv("DATABASE_DSN")
	if ok {
		storageConfig.DBAddress = envDBAddr
	}
	return &storageConfig
}

func NewSQLStorage() (*SqlStorage, error) {
	newStorage := SqlStorage{}
	newStorage.config = NewSqlStorageConfig()
	err := newStorage.InitializeConnection()
	if err != nil {
		return nil, err
	}
	return &newStorage, nil
}

func (storage *SqlStorage) InitializeConnection() error {
	//ps := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
	//	storage.config.DBAddress, storage.config.dbUser, storage.config.dbPassword, storage.config.dbName)

	db, err := sql.Open("pgx", storage.config.DBAddress)
	if err != nil {
		return err
	}
	storage.db = db
	err = storage.PingDB()
	if err != nil {
		return err
	}
	return nil
}

type DBInterface interface {
	PingDB() error
}

func (storage *SqlStorage) CloseConnection() {
	storage.db.Close()
}

func (storage *SqlStorage) AddMetrics(metricsName string, metricsValue models.Metrics) error {
	return nil
}
func (storage *SqlStorage) UpdateMetrics(metricsName string, metricsValue models.Metrics) error {
	return nil
}
func (storage *SqlStorage) GetMetrics(metricsName string) (models.Metrics, error) {
	return models.Metrics{}, nil
}
func (storage *SqlStorage) GetAllMetricsNames() ([]string, error) {
	return make([]string, 0), nil
}
func (storage *SqlStorage) GetAllMetrics() *[]models.Metrics {
	return nil
}
func (storage *SqlStorage) SaveMetricsInFile(filename string) {

}
func (storage *SqlStorage) RestoreMetricsFromFile(filename string) {

}

func (storage *SqlStorage) PingDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := storage.db.PingContext(ctx); err != nil {
		return err
	}
	return nil
}
