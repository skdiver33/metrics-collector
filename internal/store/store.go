package store

import (
	"errors"
	"fmt"
	"sync"

	"github.com/skdiver33/metrics-collector/models"
)

type MemStorage struct {
	Storage map[string]models.Metrics
	Mutex   *sync.Mutex
}

func NewMemStorage() (*MemStorage, error) {
	newStorage := MemStorage{}
	newStorage.Mutex = &sync.Mutex{}
	newStorage.Storage = make(map[string]models.Metrics)
	if err := newStorage.Initialize(); err != nil {
		return nil, err
	}
	return &newStorage, nil
}

type StorageInterface interface {
	AddMetrics(metricsName string, metricsValue models.Metrics) error
	UpdateMetrics(metricsName string, metricsValue models.Metrics) error
	GetMetrics(metricsName string) (models.Metrics, error)
	GetAllMetricsNames() ([]string, error)
}

func (inMemmory *MemStorage) Initialize() error {
	for _, metricsName := range models.GaugeMetricsNames {
		val := 0.0
		metrics := models.Metrics{ID: metricsName, MType: models.Gauge, Value: &val}
		if err := inMemmory.AddMetrics(metricsName, metrics); err != nil {
			fmt.Println("Error initialize storage.")
			return err
		}
	}
	for _, metricsName := range models.CounterMetricsNames {
		delta := int64(0)
		metrics := models.Metrics{ID: metricsName, MType: models.Counter, Delta: &delta}
		if err := inMemmory.AddMetrics(metricsName, metrics); err != nil {
			fmt.Println("Error initialize storage.")
			return err
		}
	}
	return nil
}

func (inMemmory *MemStorage) AddMetrics(metricsName string, metricsValue models.Metrics) error {
	inMemmory.Mutex.Lock()
	defer inMemmory.Mutex.Unlock()
	inMemmory.Storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetMetrics(metricsName string) (models.Metrics, error) {
	inMemmory.Mutex.Lock()
	defer inMemmory.Mutex.Unlock()
	metrics, ok := inMemmory.Storage[metricsName]
	if !ok {
		return metrics, errors.New("metrics with name not found")
	}
	return metrics, nil
}

func (inMemmory *MemStorage) UpdateMetrics(metricsName string, metricsValue models.Metrics) error {
	inMemmory.Mutex.Lock()
	defer inMemmory.Mutex.Unlock()
	inMemmory.Storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetAllMetricsNames() ([]string, error) {
	inMemmory.Mutex.Lock()
	defer inMemmory.Mutex.Unlock()
	allMetricsNames := make([]string, 0)
	for metricsName := range inMemmory.Storage {
		allMetricsNames = append(allMetricsNames, metricsName)
	}
	if len(allMetricsNames) == 0 {
		return allMetricsNames, errors.New("empty storage! initialize before use")
	}
	return allMetricsNames, nil
}
