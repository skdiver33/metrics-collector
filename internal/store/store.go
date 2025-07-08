package store

import (
	"encoding/json"
	"errors"
	"log"
	"maps"
	"os"
	"slices"
	"sync"

	"github.com/skdiver33/metrics-collector/models"
)

type MemStorage struct {
	Storage map[string]models.Metrics
	mu      *sync.Mutex
}

func NewMemStorage() (*MemStorage, error) {
	newStorage := MemStorage{}
	newStorage.mu = &sync.Mutex{}
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
	GetAllMetrics() *[]models.Metrics
	SaveMetricsInFile(filename string)
	RestoreMetricsFromFile(filename string)
}

func (inMemmory *MemStorage) Initialize() error {
	for _, metricsName := range models.GaugeMetricsNames {
		val := 0.0
		metrics := models.Metrics{ID: metricsName, MType: models.Gauge, Value: &val}
		if err := inMemmory.AddMetrics(metricsName, metrics); err != nil {
			log.Println("Error initialize storage.")
			return err
		}
	}
	for _, metricsName := range models.CounterMetricsNames {
		delta := int64(0)
		metrics := models.Metrics{ID: metricsName, MType: models.Counter, Delta: &delta}
		if err := inMemmory.AddMetrics(metricsName, metrics); err != nil {
			log.Println("Error initialize storage.")
			return err
		}
	}
	return nil
}

func (inMemmory *MemStorage) AddMetrics(metricsName string, metricsValue models.Metrics) error {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	inMemmory.Storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetMetrics(metricsName string) (models.Metrics, error) {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	metrics, ok := inMemmory.Storage[metricsName]
	if !ok {
		return metrics, errors.New("metrics with name not found")
	}
	return metrics, nil
}

func (inMemmory *MemStorage) UpdateMetrics(metricsName string, metricsValue models.Metrics) error {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	inMemmory.Storage[metricsName] = metricsValue
	return nil
}

func (inMemmory *MemStorage) GetAllMetricsNames() ([]string, error) {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	allMetricsNames := make([]string, 0)
	for metricsName := range inMemmory.Storage {
		allMetricsNames = append(allMetricsNames, metricsName)
	}
	if len(allMetricsNames) == 0 {
		return allMetricsNames, errors.New("empty storage! initialize before use")
	}
	return allMetricsNames, nil
}

func (inMemmory *MemStorage) GetAllMetrics() *[]models.Metrics {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	values := maps.Values(inMemmory.Storage)
	metricSlice := slices.Collect(values)
	return &metricSlice
}

func (inMemmory *MemStorage) SaveMetricsInFile(filename string) {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	data, err := json.Marshal(inMemmory.Storage)
	if err != nil {
		log.Printf("error convert to JSON all metrics. error: %s", err.Error())
	}
	err = os.WriteFile(filename, data, 0666)
	if err != nil {
		log.Printf("error write metrics to file. error: %s", err.Error())
	}
}

func (inMemmory *MemStorage) RestoreMetricsFromFile(filename string) {
	if _, err := os.Stat(filename); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("file with dump %s not exist", filename)
		return
	}

	readData, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("cannot read data from file. error: %s", err.Error())
		return
	}
	readStorage := make(map[string]models.Metrics)
	err = json.Unmarshal(readData, &readStorage)
	if err != nil {
		log.Printf("cannot Unmarshal read data. error: %s", err.Error())
		return
	}
	for name, value := range readStorage {
		inMemmory.UpdateMetrics(name, value)
	}
}
