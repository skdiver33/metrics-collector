package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type MetricsHandler struct {
	metricsStorage store.MemStorage
}

func (handler *MetricsHandler) ReceiveMetrics(rw http.ResponseWriter, request *http.Request) {

	metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	metricsValue := chi.URLParam(request, "metricsValue")

	if strings.Compare(metricsType, models.Counter) != 0 && strings.Compare(metricsType, models.Gauge) != 0 {
		http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
		return
	}

	if metricsName == "" {
		http.Error(rw, "Not all metrics data defined!", http.StatusNotFound)
		return
	}
	currentMetrics, err := handler.metricsStorage.GetMetricsValue(metricsName)
	if err != nil {
		http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
		return
	}
	switch metricsType {
	case models.Counter:
		{
			value, err := strconv.Atoi(metricsValue)
			if err != nil {
				http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
				return
			}
			*currentMetrics.Delta += int64(value)
		}
	case models.Gauge:
		{
			value, err := strconv.ParseFloat(metricsValue, 64)
			if err != nil {
				http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
				return
			}
			*currentMetrics.Value = float64(value)
		}
	}
	if err := handler.metricsStorage.UpdateMetricsValue(metricsName, currentMetrics); err != nil {
		http.Error(rw, "error update metrics on server", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)

}

func main() {
	handler := MetricsHandler{}
	handler.metricsStorage.InitializeStorage()
	r := chi.NewRouter()
	r.Route("/update", func(r chi.Router) {
		r.Post("/{metricsType}/{metricsName}/metricsValue", handler.ReceiveMetrics)
	})
	if err := http.ListenAndServe("localhost:8080", r); err != nil {
		panic("Error start server")
	}

}
