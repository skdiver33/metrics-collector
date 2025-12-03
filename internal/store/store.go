package store

import (
	"context"
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

func (inMemmory *MemStorage) Initialize() error {
	baseCtx := context.Background()
	for _, metricsName := range models.GaugeMetricsNames {
		val := 0.0
		metrics := models.Metrics{ID: metricsName, MType: models.Gauge, Value: &val}
		if err := inMemmory.AddMetrics(baseCtx, metrics); err != nil {
			log.Println("Error initialize storage.")
			return err
		}
	}
	for _, metricsName := range models.CounterMetricsNames {
		delta := int64(0)
		metrics := models.Metrics{ID: metricsName, MType: models.Counter, Delta: &delta}
		if err := inMemmory.AddMetrics(baseCtx, metrics); err != nil {
			log.Println("Error initialize storage.")
			return err
		}
	}
	return nil
}

func (inMemmory *MemStorage) AddMetrics(ctx context.Context, metrics models.Metrics) error {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	inMemmory.Storage[metrics.ID] = metrics
	return nil
}

func (inMemmory *MemStorage) GetMetrics(ctx context.Context, metricsName string) (models.Metrics, error) {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	metrics, ok := inMemmory.Storage[metricsName]
	if !ok {
		return metrics, errors.New("metrics with name not found")
	}
	return metrics, nil
}

func (inMemmory *MemStorage) UpdateMetrics(ctx context.Context, metricsValue models.Metrics) error {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	inMemmory.Storage[metricsValue.ID] = metricsValue
	return nil
}

func (inMemmory *MemStorage) UpdateAllMetrics(ctx context.Context, allMetrics *[]models.Metrics) error {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	for _, metrics := range *allMetrics {
		newValue := metrics.GetMetricsValue()
		currentMetrics, ok := inMemmory.Storage[metrics.ID]
		if !ok {
			log.Print("error get metrics from map")
			return nil
		}
		currentMetrics.SetMetricsValue(newValue)
		inMemmory.Storage[metrics.ID] = currentMetrics
	}

	return nil
}

func (inMemmory *MemStorage) GetAllMetrics(ctx context.Context) *[]models.Metrics {
	inMemmory.mu.Lock()
	defer inMemmory.mu.Unlock()
	values := maps.Values(inMemmory.Storage)
	metricSlice := slices.Collect(values)
	return &metricSlice
}

func (inMemmory *MemStorage) CreateDBDump(ctx context.Context, filename string) {
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

func (inMemmory *MemStorage) RestoreDBDump(ctx context.Context, filename string) {
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
	for _, value := range readStorage {
		inMemmory.UpdateMetrics(context.Background(), value)
	}
}

func (inMemmory *MemStorage) CloseConnection() {
	clear(inMemmory.Storage)
}
