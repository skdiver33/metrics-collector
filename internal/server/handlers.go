package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
	"go.uber.org/zap"
)

type MetricsHandler struct {
	metricsStorage store.StorageInterface
	logger         *zap.SugaredLogger
}

func NewMetricsHandler(storage store.StorageInterface) (*MetricsHandler, error) {
	newHandler := MetricsHandler{}
	newHandler.metricsStorage = storage
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	newHandler.logger = logger.Sugar()

	return &newHandler, nil
}

//****************************** Endpoint Handlers **************************************

func (handler *MetricsHandler) SetMetrics(rw http.ResponseWriter, request *http.Request) {

	metricsType := chi.URLParam(request, "metricsType")
	metricsName := chi.URLParam(request, "metricsName")
	metricsValue := chi.URLParam(request, "metricsValue")

	if metricsType != models.Counter && metricsType != models.Gauge {
		log.Print("wrong metrics type")
		http.Error(rw, "wrong metrics type", http.StatusBadRequest)
		return
	}

	currentMetrics, err := handler.metricsStorage.GetMetrics(request.Context(), metricsName)
	if err != nil {
		currentMetrics = models.Metrics{ID: metricsName, MType: metricsType}
		currentMetrics.SetMetricsValue("0")
		handler.metricsStorage.AddMetrics(request.Context(), currentMetrics)
	}

	if err := currentMetrics.SetMetricsValue(metricsValue); err != nil {
		log.Print("error set up new value in metrics")
		http.Error(rw, "", http.StatusBadRequest)
		return
	}
	if err := handler.metricsStorage.UpdateMetrics(request.Context(), currentMetrics); err != nil {
		log.Print("error update metrics on server")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)
}

func (handler *MetricsHandler) SetJSONMetrics(rw http.ResponseWriter, request *http.Request) {

	receiveMetrics := models.Metrics{}
	if err := json.NewDecoder(request.Body).Decode(&receiveMetrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if receiveMetrics.MType != models.Counter && receiveMetrics.MType != models.Gauge {
		log.Print("Wrong metrics type")
		http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
		return
	}

	if receiveMetrics.ID == "" {
		log.Print("empty metrics name ")
		http.Error(rw, "empty metrics name !", http.StatusNotFound)
		return
	}

	currentMetrics, err := handler.metricsStorage.GetMetrics(request.Context(), receiveMetrics.ID)
	if err != nil {
		currentMetrics = models.Metrics{ID: receiveMetrics.ID, MType: receiveMetrics.MType}
		currentMetrics.SetMetricsValue("0")
		handler.metricsStorage.AddMetrics(request.Context(), currentMetrics)
	}

	newValue := receiveMetrics.GetMetricsValue()
	if err := currentMetrics.SetMetricsValue(newValue); err != nil {
		log.Print("error set up new value in metrics")
		http.Error(rw, "error set up new value in metrics", http.StatusBadRequest)
		return
	}
	if err := handler.metricsStorage.UpdateMetrics(request.Context(), currentMetrics); err != nil {
		log.Print("error update metrics in storage")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(currentMetrics)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(resp)
}

func (handler *MetricsHandler) GetMetrics(rw http.ResponseWriter, request *http.Request) {
	metricsName := chi.URLParam(request, "metricsName")
	metrics, err := handler.metricsStorage.GetMetrics(request.Context(), metricsName)
	if err != nil {
		log.Print("error get metrics from storage")
		http.Error(rw, "error get metrics from storage", http.StatusNotFound)
		return
	}

	rw.Header().Set("Content-type", "text/plain")
	rw.Write([]byte(metrics.GetMetricsValue()))
}

func (handler *MetricsHandler) GetJSONMetrics(rw http.ResponseWriter, request *http.Request) {

	receiveMetrics := models.Metrics{}
	if err := json.NewDecoder(request.Body).Decode(&receiveMetrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := handler.metricsStorage.GetMetrics(request.Context(), receiveMetrics.ID)
	if err != nil {
		log.Print("error get metrics from storage")
		http.Error(rw, "error get metrics from storage", http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(response)
	if err != nil {
		log.Print("error Marshal response")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(resp)
}

func (handler *MetricsHandler) GetAllMetrics(rw http.ResponseWriter, request *http.Request) {
	answer := "<!DOCTYPE html>\n<html>\n<head>\n<title> Known metrics </title>\n</head>\n<body\n>"
	metrics := handler.metricsStorage.GetAllMetrics(request.Context())
	if metrics == nil {
		log.Print("error get metrics form storage in GetAllMetrics")
		http.Error(rw, "", http.StatusInternalServerError)
		return
	}

	for _, curMetr := range *metrics {
		answer = fmt.Sprintf("<p>%s %s %s %s </p>\n", answer, curMetr.ID, curMetr.MType, curMetr.GetMetricsValue())
	}
	answer += "</body>\n</html>"

	rw.Header().Set("Content-type", "text/html")
	rw.Write([]byte(answer))
}

//************************* Logger Handler *********************************************

type (
	ResponseData struct {
		Status int
		Size   int
	}
	LoggingResponseWriter struct {
		http.ResponseWriter
		ResponseData *ResponseData
	}
)

func (r *LoggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.ResponseData.Size += size
	return size, err
}

func (r *LoggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.ResponseData.Status = statusCode
}

func (handler *MetricsHandler) RequestLogger(h http.Handler) http.Handler {
	logerFunc := func(w http.ResponseWriter, req *http.Request) {

		start := time.Now()
		responseData := &ResponseData{Status: 0, Size: 0}
		lw := LoggingResponseWriter{ResponseWriter: w, ResponseData: responseData}

		h.ServeHTTP(&lw, req)

		duration := time.Since(start)
		handler.logger.Infoln(
			"uri", req.RequestURI,
			"method", req.Method,
			"status", responseData.Status,
			"duration", duration,
			"size", responseData.Size,
		)
	}
	return http.HandlerFunc(logerFunc)
}

//********************** Compress Handler *******************************************

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	typeForGzip := []string{"application/json", "text/html"}
	contentTypes := strings.Join(w.Header().Values("Content-Type"), " ")
	for _, value := range typeForGzip {
		if strings.Contains(contentTypes, value) {
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
			return w.Writer.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

func (handler *MetricsHandler) GzipHandle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Println("error create gzip")
				return
			}
			decompressBody, err := io.ReadAll(gz)
			if err != nil {
				log.Println("error decompress body")
				return
			}
			gz.Close()
			r.Body = io.NopCloser(bytes.NewReader(decompressBody))
			r.ContentLength = int64(len(decompressBody))

		}

		//support compression client check
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}
		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		defer gz.Close()

		next.ServeHTTP(gzipWriter{ResponseWriter: w, Writer: gz}, r)
	})
}

func (handler *MetricsHandler) PingDB(rw http.ResponseWriter, request *http.Request) {
	switch value := handler.metricsStorage.(type) {
	case store.DBInterface:
		{
			err := value.PingDB(request.Context())
			if err != nil {
				log.Print("DB not connected")
				http.Error(rw, "", http.StatusInternalServerError)
			}
			rw.WriteHeader(http.StatusOK)
			return
		}
	default:
		http.Error(rw, "", http.StatusBadRequest)

	}
}
