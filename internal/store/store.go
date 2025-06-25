package store

import (
	"errors"
	"fmt"

	"github.com/skdiver33/metrics-collector/models"
)

type MemStorage struct {
	storage map[string]models.Metrics
	//mu      sync.RWMutex
}

type Storage interface {
	InitializeStorage() error
	AddMetrics(metricsName string, metricsValue models.Metrics) error
	UpdateMetrics(metricsName string, metricsValue models.Metrics) error
	GetMetrics(metricsName string) (models.Metrics, error)
	GetAllMetricsNames() []string
}

func (inMemmory *MemStorage) InitializeStorage() error {
	inMemmory.storage = make(map[string]models.Metrics)
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

	_, err := inMemmory.GetMetrics(metricsName)
	if err == nil {
		return errors.New("metrics Already exist in storage")
	}
	//inMemmory.mu.Lock()
	//defer inMemmory.mu.Unlock()
	inMemmory.storage[metricsName] = metricsValue
	return nil

}

func (inMemmory *MemStorage) GetMetrics(metricsName string) (models.Metrics, error) {
	//inMemmory.mu.Lock()
	//defer inMemmory.mu.Unlock()
	metrics, ok := inMemmory.storage[metricsName]
	if !ok {
		metrics = models.Metrics{}
		return metrics, errors.New("metrics with name not found")
	}
	return metrics, nil

}

func (inMemmory *MemStorage) UpdateMetrics(metricsName string, metricsValue models.Metrics) error {

	_, err := inMemmory.GetMetrics(metricsName)

	if err != nil {
		message := fmt.Sprintf("Error update value %s", err.Error())
		return errors.New(message)
	}
	//inMemmory.mu.Lock()
	//defer inMemmory.mu.Unlock()
	inMemmory.storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetAllMetricsNames() ([]string, error) {

	allMetricsNames := make([]string, 0)
	//inMemmory.mu.Lock()
	//defer inMemmory.mu.Unlock()
	for metricsName := range inMemmory.storage {
		allMetricsNames = append(allMetricsNames, metricsName)

	}
	if len(allMetricsNames) == 0 {
		return allMetricsNames, errors.New("empty storage! initialize before use")
	}
	return allMetricsNames, nil
}
