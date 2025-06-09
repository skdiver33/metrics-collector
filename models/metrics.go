package models

import (
	"errors"
	"strconv"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// NOTE: Не усложняем пример, вводя иерархическую вложенность структур.
// Органичиваясь плоской моделью.
// Delta и Value объявлены через указатели,
// что бы отличать значение "0", от не заданного значения
// и соответственно не кодировать в структуру.
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

func (metrics *Metrics) GetMetricsValue() string {
	value := ""
	switch metrics.MType {
	case Counter:
		{
			if metrics.Delta == nil {
				return "Value not defined"
			}
			value = strconv.FormatInt(*metrics.Delta, 10)
		}
	case Gauge:
		{
			if metrics.Value == nil {
				return "Value not defined"
			}
			value = strconv.FormatFloat(*metrics.Value, 'f', -1, 64)
		}
	}
	return value
}

func (metrics *Metrics) SetMetricsValue(newValue string) error {
	switch metrics.MType {
	case Counter:
		{
			value, err := strconv.Atoi(newValue)
			if err != nil {
				return errors.New("wrong metrics type")

			}
			*metrics.Delta += int64(value)
		}
	case Gauge:
		{
			value, err := strconv.ParseFloat(newValue, 64)
			if err != nil {
				return errors.New("wrong metrics type")

			}
			*metrics.Value = float64(value)
		}
	}
	return nil
}

var GaugeMetricsNames = []string{"Alloc",
	"BuckHashSys",
	"Frees",
	"GCCPUFraction",
	"GCSys",
	"HeapAlloc",
	"HeapIdle",
	"HeapInuse",
	"HeapObjects",
	"HeapReleased",
	"HeapSys",
	"LastGC",
	"Lookups",
	"MCacheInuse",
	"MCacheSys",
	"MSpanInuse",
	"MSpanSys",
	"Mallocs",
	"NextGC",
	"NumForcedGC",
	"NumGC",
	"OtherSys",
	"PauseTotalNs",
	"StackInuse",
	"StackSys",
	"Sys",
	"TotalAlloc",
	"RandomValue"}
var CounterMetricsNames = []string{"PollCount"}
