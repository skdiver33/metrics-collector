package store

import "github.com/skdiver33/metrics-collector/models"

type StorageInterface interface {
	AddMetrics(metricsValue models.Metrics) error
	UpdateMetrics(metricsValue models.Metrics) error
	GetMetrics(metricsName string) (models.Metrics, error)
	GetAllMetricsNames() ([]string, error)
	GetAllMetrics() *[]models.Metrics
	SaveMetricsInFile(filename string)
	RestoreMetricsFromFile(filename string)
}

type DBInterface interface {
	PingDB() error
}
