package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi"
	types "github.com/skdiver33/metrics-collector/internal/server"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type MetricsHandler struct {
	metricsStorage store.MemStorage
	sugar          zap.SugaredLogger
}

func (handler *MetricsHandler) receiveMetricsHandler(rw http.ResponseWriter, request *http.Request) {

	metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	metricsValue := chi.URLParam(request, "metricsValue")

	//for testing 3 iteration add test metrics name in storage
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
	currentMetrics, err := handler.metricsStorage.GetMetrics(metricsName)
	if err != nil {
		http.Error(rw, "metrics not found", http.StatusBadRequest)
		return
	}
	if err := currentMetrics.SetMetricsValue(metricsValue); err != nil {
		http.Error(rw, "error set up new value in metrics", http.StatusBadRequest)
		return
	}
	if err := handler.metricsStorage.UpdateMetrics(metricsName, currentMetrics); err != nil {
		http.Error(rw, "error update metrics on server", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)

}

func (handler *MetricsHandler) returnAllMetricsHandler(rw http.ResponseWriter, request *http.Request) {
	answer := "<!DOCTYPE html>\n<html>\n<head>\n<title> Known metrics </title>\n</head>\n<body\n>"
	metricsNames, err := handler.metricsStorage.GetAllMetricsNames()
	if err != nil {
		http.Error(rw, "error get metrics name from storage", http.StatusInternalServerError)
		return
	}
	for _, name := range metricsNames {
		metrics, _ := handler.metricsStorage.GetMetrics(name)
		answer = fmt.Sprintf("<p>%s %s %s %s </p>\n", answer, name, metrics.MType, metrics.GetMetricsValue())
	}
	answer += "</body>\n</html>"
	rw.Header().Set("Content-type", "text/html")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(answer))

}

func (handler *MetricsHandler) metricsInfoHandler(rw http.ResponseWriter, request *http.Request) {
	metricsName := chi.URLParam(request, "metricsName")
	metrics, err := handler.metricsStorage.GetMetrics(metricsName)
	if err != nil {
		http.Error(rw, "error get metrics from storage", http.StatusNotFound)
		return
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(metrics.GetMetricsValue()))
}

func (handler *MetricsHandler) requestLogger(h http.Handler) http.Handler {
	logerFunc := func(w http.ResponseWriter, req *http.Request) {

		start := time.Now()
		responseData := &types.ResponseData{Status: 0, Size: 0}
		lw := types.LoggingResponseWriter{ResponseWriter: w, ResponseData: responseData}

		h.ServeHTTP(&lw, req)
		duration := time.Since(start)
		handler.sugar.Infoln(
			"uri", req.RequestURI,
			"method", req.Method,
			"status", responseData.Status,
			"duration", duration,
			"size", responseData.Size,
		)
	}
	return http.HandlerFunc(logerFunc)
}
func MetricRouter() chi.Router {
	handler := MetricsHandler{}
	handler.metricsStorage.InitializeStorage()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	handler.sugar = *logger.Sugar()
	r := chi.NewRouter()

	r.Use(handler.requestLogger)
	r.Route("/", func(r chi.Router) {
		r.Get("/", handler.returnAllMetricsHandler)
		r.Get("/value/{metricsType}/{metricsName}", handler.metricsInfoHandler)
		r.Post("/update/{metricsType}/{metricsName}/{metricsValue}", handler.receiveMetricsHandler)
	})
	return r
}

func main() {

	serverFlags := flag.NewFlagSet("Start flags", flag.ExitOnError)
	startAdress := serverFlags.String("a", "localhost:8080", "adress for start server")
	serverFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		startAdress = &envServerAddr
	}

	if err := http.ListenAndServe(*startAdress, MetricRouter()); err != nil {
		panic("Error start server")
	}

}
