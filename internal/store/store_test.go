package store_test

import (
	"context"
	"math/rand/v2"
	"strconv"
	"testing"

	memstore "github.com/skdiver33/metrics-collector/internal/store"
	model "github.com/skdiver33/metrics-collector/models"
)

func Benchmark_AddMetrics(b *testing.B) {
	b.StopTimer()
	count := 1000
	storage, err := memstore.NewMemStorage()
	if err != nil {
		b.Error("error init storage")
	}
	testMetrics := make([]model.Metrics, count)
	for i := 0; i < count; i++ {
		val := rand.Float64()
		name := "Tets" + strconv.Itoa(rand.Int())
		testMetrics = append(testMetrics, model.Metrics{ID: name, MType: model.Gauge, Value: &val})
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, metrics := range testMetrics {
			storage.AddMetrics(context.Background(), metrics)
		}
	}
}

func Benchmark_GetMetrics(b *testing.B) {
	b.StopTimer()
	count := 1000
	storage, err := memstore.NewMemStorage()
	if err != nil {
		b.Error("error init storage")
	}
	testMetrics := make([]model.Metrics, count)
	for i := 0; i < count; i++ {
		val := rand.Float64()
		name := "Tets" + strconv.Itoa(rand.Int())
		storage.AddMetrics(context.Background(), model.Metrics{ID: name, MType: model.Gauge, Value: &val})
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		for _, metrics := range testMetrics {
			storage.GetMetrics(context.Background(), metrics.ID)
		}
	}
}

func Benchmark_UpdateMetrics(b *testing.B) {
	count := 1000
	storage, err := memstore.NewMemStorage()
	if err != nil {
		b.Error("error init storage")
	}
	testMetrics := make([]model.Metrics, count)
	for i := 0; i < count; i++ {
		val := rand.Float64()
		name := "Tets" + strconv.Itoa(rand.Int())
		testMetrics = append(testMetrics, model.Metrics{ID: name, MType: model.Gauge, Value: &val})
	}
	for _, metrics := range testMetrics {
		storage.AddMetrics(context.Background(), metrics)
	}
	for _, metrics := range testMetrics {
		metrics.SetMetricsValue(strconv.FormatFloat(rand.Float64(), 'f', 6, 64))
	}
	b.Run("update 1 metrics", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, metrics := range testMetrics {
				storage.UpdateMetrics(context.Background(), metrics)
			}
		}
	})
	b.Run("update all metrics", func(b *testing.B) {
		b.StartTimer()
		for i := 0; i < b.N; i++ {
			storage.UpdateAllMetrics(context.Background(), &testMetrics)
		}
	})

}
