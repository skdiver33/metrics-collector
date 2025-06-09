package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/skdiver33/metrics-collector/models"
)

func MetricsHandler(rw http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		http.Error(rw, "Only Post requests are allowed!", http.StatusMethodNotAllowed)
		return
	}

	metrics, ok := strings.CutPrefix(request.URL.Path, "/update/")
	if !ok {
		http.Error(rw, "internal server error", http.StatusNotFound)
		return
	}
	metrics, _ = strings.CutSuffix(metrics, "/")

	metricsData := strings.Split(metrics, "/")
	fmt.Print(metricsData)
	if len(metricsData) != 3 {
		http.Error(rw, "Not all metrics data defined!", http.StatusNotFound)
		return
	}
	metricsType := metricsData[0]
	metricsName := metricsData[1]
	metricsValue := metricsData[2]

	if strings.Compare(metricsType, models.Counter) != 0 && strings.Compare(metricsType, models.Gauge) != 0 {
		http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
		return
	}

	if metricsName == "" {
		http.Error(rw, "Not all metrics data defined!", http.StatusNotFound)
		return
	}
	switch metricsType {
	case models.Counter:
		{
			if _, err := strconv.Atoi(metricsValue); err != nil {
				http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
				return
			}
		}
	case models.Gauge:
		{
			if _, err := strconv.ParseFloat(metricsValue, 64); err != nil {
				http.Error(rw, "Wrong metrics type", http.StatusBadRequest)
				return
			}
		}
	}
	rw.Header().Set("Content-type", "text/plain")
	rw.WriteHeader(http.StatusOK)

}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/update/", MetricsHandler)
	if err := http.ListenAndServe("localhost:8080", mux); err != nil {
		panic("Error start server")
	}
}
