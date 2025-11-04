// Модуль handlers содержит обработчики входящих запросов, middlewares используемые сервером.

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
	audit "github.com/skdiver33/metrics-collector/internal/audit"
	"github.com/skdiver33/metrics-collector/internal/misc"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
	"go.uber.org/zap"
)

// MetricsHandler - описывает обобщенный обработчик запросов. Обработчик может выполнять логгирование запросов, взаимодействует с храниоищем метрик, позволяет выполнять шифрование данных.
type MetricsHandler struct {
	metricsStorage store.StorageInterface
	logger         *zap.SugaredLogger
	signingKey     string
	auditor        *audit.AuditEvent
}

// NewMetricsHandler - создает новый обработчик запросов, взаимодеййствующий с переданным хранилищем.
func NewMetricsHandler(storage store.StorageInterface) (*MetricsHandler, error) {
	newHandler := MetricsHandler{}
	newHandler.metricsStorage = storage
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()
	newHandler.logger = logger.Sugar()
	newHandler.auditor = audit.NewAuditEvent()

	return &newHandler, nil
}

//****************************** Endpoint Handlers **************************************

// SetMetrics - метод устанавливающий значение метрики переданной в параметрах запроса.
// Тип запроса - POST,  URL запроса: /update/metricsType/metricsName/metricsValue
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
	auditData := make([]string, 0)
	auditData = append(auditData, metricsName)
	handler.auditor.Update(auditData, request.RemoteAddr)

	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)

}

// SetJSONMetrics - обработчик обновяющий значение метрики переданной в параметрах запроса. Метрика передается в формате JSON.
// Тип запроса - POST,  URL запроса: /update

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

	auditData := make([]string, 0)
	auditData = append(auditData, receiveMetrics.ID)
	handler.auditor.Update(auditData, request.RemoteAddr)

	rw.Header().Set("Content-Type", "application/json")
	rw.Write(resp)
}

// GetMetrics - обработчик позволяющий получить значение метрики. Название метрики передается в параметре запроса.
// Тип запроса - GET,  URL запроса: /metricsType/metricsName
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

// GetJSONMetrics - обработчик позволяющий получить значение метрики. Название метрики передается в параметре запроса.
// Запрос передается в формате JSON. Ответ отправляется в формате JSON.
// Тип запроса - POST,  URL запроса: /value/
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

// GetAllMetrics - обработчик позволяющий получить все метрики, хранящиеся в хранилище.
// Ответ отправляется в формате JSON.
// Тип запроса - GET,  URL запроса: /
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

// PingDB - обработчик позволяющий проверить подключение к БД, при использовании ее в качестве хранилища.
// При установки подключения к БД возвращается StatusCode = 200.
// Тип запроса - GET,  URL запроса: /ping

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

func (handler *MetricsHandler) SetBunchMetrics(rw http.ResponseWriter, request *http.Request) {

	var receiveMetrics []models.Metrics
	if err := json.NewDecoder(request.Body).Decode(&receiveMetrics); err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if err := handler.metricsStorage.UpdateAllMetrics(request.Context(), &receiveMetrics); err != nil {
		log.Printf("error update all metrics in storage %s", err.Error())
		http.Error(rw, "", http.StatusBadRequest)
	}

	auditData := make([]string, 0)
	for _, metric := range receiveMetrics {
		auditData = append(auditData, metric.ID)
	}
	handler.auditor.Update(auditData, request.RemoteAddr)

	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)
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

//****************************** Signinig  Handler **************************************

type SignigWriter struct {
	http.ResponseWriter
	key string
}

func (w SignigWriter) Write(b []byte) (int, error) {
	if w.key != "" {
		hash := misc.GetRequestHash(b, w.key)
		w.Header().Set("HashSHA256", hash)
	}
	//	w.WriteHeader(http.StatusOK)
	return w.ResponseWriter.Write(b)

}

func (handler *MetricsHandler) SigningHandle(next http.Handler) http.Handler {
	signigKey := handler.signingKey
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if receiveHash := r.Header.Get("HashSHA256"); receiveHash != "" && signigKey != "" {

			body, err := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(body))
			if err != nil {
				log.Println("error read body for check sign")
				return
			}
			hash := misc.GetRequestHash(body, signigKey)
			if hash != receiveHash {
				log.Println("receive hash and calculate hash does not match")
				http.Error(w, "", http.StatusBadRequest)
				return
			}
		}
		next.ServeHTTP(SignigWriter{key: signigKey, ResponseWriter: w}, r)

		//next.ServeHTTP(w, r)

	})
}

//********************** Compress Handler *******************************************

type gzipWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (w gzipWriter) Write(b []byte) (int, error) {
	typeForGzip := []string{"application/json", "text/html"}
	contentTypes := strings.Join(w.Header().Values("Content-Type"), " ")
	if len(b) > 4096 {
		for _, value := range typeForGzip {
			if strings.Contains(contentTypes, value) {
				w.Header().Set("Content-Encoding", "gzip")
				//w.WriteHeader(http.StatusOK)
				return w.Writer.Write(b)
			}
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

//***********************************************************************************
