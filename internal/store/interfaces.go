package store

import (
	"context"

	"github.com/skdiver33/metrics-collector/models"
)

type StorageInterface interface {
	AddMetrics(ctx context.Context, metricsValue models.Metrics) error
	UpdateMetrics(ctx context.Context, metricsValue models.Metrics) error
	GetMetrics(ctx context.Context, metricsName string) (models.Metrics, error)
	//	GetAllMetricsNames() ([]string, error)
	GetAllMetrics(ctx context.Context) *[]models.Metrics
	CreateDBDump(ctx context.Context, filename string)
	RestoreDBDump(ctx context.Context, filename string)
}

type DBInterface interface {
	PingDB(ctx context.Context) error
}
