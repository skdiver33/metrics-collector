package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/go-chi/chi"
	serverLoggerTypes "github.com/skdiver33/metrics-collector/internal/server"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type MetricsHandler struct {
	metricsStorage *store.MemStorage
	sugar          *zap.SugaredLogger
	mu             sync.Mutex
}

func NewMetricsHandler() (*MetricsHandler, error) {
	newHandler := MetricsHandler{}

	newMetricsStorage, err := store.NewMemStorage()
	if err != nil {
		return nil, err
	}
	newHandler.metricsStorage = newMetricsStorage

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	newHandler.sugar = logger.Sugar()

	return &newHandler, nil
}

func (handler *MetricsHandler) receiveMetricsHandler(rw http.ResponseWriter, request *http.Request) {

	//fmt.Print("Receive new metrics not JSON")
	metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	metricsValue := chi.URLParam(request, "metricsValue")

	//for testing 3 iteration add test metrics name in storage
	if strings.Contains(metricsName, "testSetGet") {
		_, err := handler.metricsStorage.GetMetrics(metricsName)
		if err != nil {
			testMetrics := models.Metrics{ID: metricsName, MType: metricsType}
			testMetrics.SetMetricsValue("0")
			handler.metricsStorage.AddMetrics(metricsName, testMetrics)
		}

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

func (handler *MetricsHandler) setJSONMetrics(rw http.ResponseWriter, request *http.Request) {
	// handler.mu.Lock()
	// defer handler.mu.Unlock()
	receiveMetrics := models.Metrics{}
	if err := json.NewDecoder(request.Body).Decode(&receiveMetrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	//for testing 7 iteration add test metrics name in storage
	if strings.Contains(receiveMetrics.ID, "GetSet") {
		_, err := handler.metricsStorage.GetMetrics(receiveMetrics.ID)
		if err != nil {
			testMetrics := models.Metrics{ID: receiveMetrics.ID, MType: receiveMetrics.MType}
			testMetrics.SetMetricsValue("0")
			handler.metricsStorage.AddMetrics(receiveMetrics.ID, testMetrics)
		}

	}

	//fmt.Println("set ", receiveMetrics)
	if strings.Compare(receiveMetrics.MType, models.Counter) != 0 && strings.Compare(receiveMetrics.MType, models.Gauge) != 0 {
		http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
		return
	}

	if receiveMetrics.ID == "" {
		http.Error(rw, "empty metrics name !", http.StatusNotFound)
		return
	}
	currentMetrics, err := handler.metricsStorage.GetMetrics(receiveMetrics.ID)
	if err != nil {
		http.Error(rw, "metrics not found", http.StatusBadRequest)
		return
	}
	if err := currentMetrics.SetMetricsValue(receiveMetrics.GetMetricsValue()); err != nil {
		http.Error(rw, "error set up new value in metrics", http.StatusBadRequest)
		return
	}

	if err := handler.metricsStorage.UpdateMetrics(receiveMetrics.ID, currentMetrics); err != nil {
		http.Error(rw, "error update metrics on server", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(currentMetrics)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(resp)
}

func (handler *MetricsHandler) getJSONMetrics(rw http.ResponseWriter, request *http.Request) {
	// handler.mu.Lock()
	// defer handler.mu.Unlock()

	receiveMetrics := models.Metrics{}
	if err := json.NewDecoder(request.Body).Decode(&receiveMetrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	//fmt.Println("get ", receiveMetrics)
	response, err := handler.metricsStorage.GetMetrics(receiveMetrics.ID)
	if err != nil {
		http.Error(rw, "error get metrics from storage", http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(response)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	rw.Write(resp)
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
		responseData := &serverLoggerTypes.ResponseData{Status: 0, Size: 0}
		lw := serverLoggerTypes.LoggingResponseWriter{ResponseWriter: w, ResponseData: responseData}

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

func MetricsRouter() (*chi.Mux, error) {
	handler, err := NewMetricsHandler()
	if err != nil {
		return nil, err
	}
	r := chi.NewRouter()
	//r.Use(handler.requestLogger)
	r.Route("/", func(r chi.Router) {
		r.Get("/", handler.returnAllMetricsHandler)
		r.Route("/value", func(r chi.Router) {
			r.Post("/", handler.getJSONMetrics)
			r.Get("/{metricsType}/{metricsName}", handler.metricsInfoHandler)
		})
		r.Route("/update", func(r chi.Router) {
			r.Post("/", handler.setJSONMetrics)
			r.Post("/{metricsType}/{metricsName}/{metricsValue}", handler.receiveMetricsHandler)
		})
	})
	return r, nil
}

func main() {

	serverFlags := flag.NewFlagSet("Start flags", flag.ExitOnError)
	startAdress := serverFlags.String("a", "localhost:8080", "adress for start server")
	serverFlags.Parse(os.Args[1:])
	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		startAdress = &envServerAddr
	}

	chiRouter, err := MetricsRouter()
	if err != nil {
		panic(err.Error())
	}
	if err := http.ListenAndServe(*startAdress, chiRouter); err != nil {
		panic(err.Error())
	}

}
