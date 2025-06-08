package store

import (
	"fmt"

	"github.com/skdiver33/metrics-collector/models"
)

type MemStorage struct {
	Storage map[string]models.Metrics
}

type Storage interface {
	AddMetrics(metricsName string, metricsValue models.Metrics) error
	UpdateMetricsValue(metricsName string, metricsValue models.Metrics) error
	GetMetricsValue(metricsName string) (models.Metrics, error)
	GetAllMetricsNames() []string
	RemoveMetrics(metricsName string) error
}

type storageError struct {
	description string
}

func (e storageError) Error() string {
	return fmt.Sprintf("Storage Error!. %s", e.description)
}

func (inMemmory *MemStorage) AddMetrics(metricsName string, metricsValue models.Metrics) error {
	_, err := inMemmory.GetMetricsValue(metricsName)
	if err == nil {
		return &storageError{"Metrics Already exist in storage"}
	}

	inMemmory.Storage[metricsName] = metricsValue
	return nil

}

func (inMemmory *MemStorage) GetMetricsValue(metricsName string) (models.Metrics, error) {
	metrics, ok := inMemmory.Storage[metricsName]
	if !ok {
		metrics = models.Metrics{}
		return metrics, &storageError{"Metrics with name not found!"}
	}
	return metrics, nil

}

func (inMemmory *MemStorage) UpdateMetricsValue(metricsName string, metricsValue models.Metrics) error {
	_, err := inMemmory.GetMetricsValue(metricsName)

	if err != nil {
		return &storageError{fmt.Sprintf("Error update value! %s", err.Error())}
	}
	inMemmory.Storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetAllMetricsNames() []string {
	allMetricsNames := make([]string, 0)
	for metricsName := range inMemmory.Storage {
		allMetricsNames = append(allMetricsNames, metricsName)

	}
	return allMetricsNames
}
