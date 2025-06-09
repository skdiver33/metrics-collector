package main

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type MetricsHandler struct {
	metricsStorage store.MemStorage
}

func (handler *MetricsHandler) receiveMetricsHandler(rw http.ResponseWriter, request *http.Request) {

	metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	metricsValue := chi.URLParam(request, "metricsValue")

	//for testing
	if strings.Contains(metricsName, "testSetGet") {
		handler.metricsStorage.AddMetrics(metricsName, models.Metrics{MType: metricsType})
	}

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
		http.Error(rw, "metrics not found", http.StatusBadRequest)
		return
	}
	if err := currentMetrics.SetMetricsValue(metricsValue); err != nil {
		http.Error(rw, "error set up new value in metrics", http.StatusBadRequest)
		return
	}
	if err := handler.metricsStorage.UpdateMetricsValue(metricsName, currentMetrics); err != nil {
		http.Error(rw, "error update metrics on server", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)

}

func (handler *MetricsHandler) returnAllMetricsHandler(rw http.ResponseWriter, request *http.Request) {
	answer := "<!DOCTYPE html>\n<html>\n<head>\n<title> Known metrics </title>\n</head>\n"
	metricsNames, err := handler.metricsStorage.GetAllMetricsNames()
	if err != nil {
		http.Error(rw, "error get metrics name from storage", http.StatusInternalServerError)
		return
	}
	for _, name := range metricsNames {
		metrics, _ := handler.metricsStorage.GetMetricsValue(name)
		answer = answer + name + metrics.MType + metrics.GetMetricsValue() + "\n"
	}
	answer += "</html>"
	rw.Header().Set("Content-type", "text/html")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(answer))

}

func (handler *MetricsHandler) metricsInfoHandler(rw http.ResponseWriter, request *http.Request) {
	//metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	//answer := metricsName + metricsType
	metrics, err := handler.metricsStorage.GetMetricsValue(metricsName)
	if err != nil {
		http.Error(rw, "error get metrics from storage", http.StatusNotFound)
		return
	}
	//answer += metrics.GetMetricsValue()
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(metrics.GetMetricsValue()))
}

func MetricRouter() chi.Router {
	handler := MetricsHandler{}
	handler.metricsStorage.InitializeStorage()
	//	fmt.Println(handler.metricsStorage)
	r := chi.NewRouter()
	r.Route("/", func(r chi.Router) {
		r.Get("/", handler.returnAllMetricsHandler)
		r.Get("/value/{metricsType}/{metricsName}", handler.metricsInfoHandler)
		r.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.receiveMetricsHandler)
	})
	return r
}

func main() {
	if err := http.ListenAndServe("localhost:8080", MetricRouter()); err != nil {
		panic("Error start server")
	}

}
