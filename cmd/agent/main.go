package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type Agent struct {
	metricStorage store.MemStorage
}

type MetricsCollector interface {
	UpdateMetrics() error
	SendMetrics() error
}

func (agent *Agent) UpdateMetrics() error {
	memStat := runtime.MemStats{}
	runtime.ReadMemStats(&memStat)
	value := reflect.ValueOf(memStat)
	allMMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}
	for _, name := range allMMetricsName {
		currentMetrics, err := agent.metricStorage.GetMetricsValue(name)
		if err != nil {
			fmt.Printf("Error get current value metrics for name %s\n", name)
			return err
		}
		switch currentMetrics.MType {
		case models.Gauge:
			{
				fieldValue := value.FieldByName(name)
				newValue := 0.0

				if !fieldValue.IsValid() {
					newValue = rand.Float64()
				} else {
					switch fieldValue.Kind() {
					case reflect.Float64:
						newValue = float64(fieldValue.Float())
					case reflect.Uint64, reflect.Uint32:
						newValue = float64(fieldValue.Uint())
					default:
						fmt.Printf("unhandled kind %s", fieldValue.Kind())
						return errors.New("wrong data type in source of gauge metrics")
					}
				}
				currentMetrics.Value = &newValue

			}
		case models.Counter:
			{
				newValue := int64(0)
				if currentMetrics.Delta == nil {
					newValue = 1
				} else {
					newValue = *currentMetrics.Delta + 1
				}

				currentMetrics.Delta = &newValue
			}
		}
		if err := agent.metricStorage.UpdateMetricsValue(name, currentMetrics); err != nil {
			return err
		}

	}
	return nil
}

func (agent *Agent) SendMetrics() error {
	requestPattern := "http://localhost:8080/update/%s/%s/%s"

	allMMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}

	for _, name := range allMMetricsName {
		currentMetrics, err := agent.metricStorage.GetMetricsValue(name)
		if err != nil {
			fmt.Print(err.Error())
			return err
		}
		var value string
		if currentMetrics.Value == nil && currentMetrics.Delta == nil {
			return errors.New("error! update metrics before send")
		}
		if currentMetrics.Value != nil {
			value = strconv.FormatFloat(*currentMetrics.Value, 'f', -1, 64)
		} else {
			value = strconv.Itoa(int(*currentMetrics.Delta))
		}
		response, err := http.Post(fmt.Sprintf(requestPattern, currentMetrics.MType, name, value), "Content-Type: text/plain", nil)
		if err != nil {
			fmt.Println(err)
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			//return error send metrics to server
			return errors.New("error update metrics on server!!! Response code not 200")
		}
	}
	return nil
}

func (agent *Agent) MainLoop() error {
	if err := agent.metricStorage.InitializeStorage(); err != nil {
		return err
	}
	poolInterval := 2
	reportInterval := 10
	period := reportInterval / poolInterval
	for {
		time.Sleep(time.Duration(poolInterval) * time.Second)
		if err := agent.UpdateMetrics(); err != nil {
			return err
		}
		val, err := agent.metricStorage.GetMetricsValue("PollCount")
		if err != nil {
			return err
		}
		if *val.Delta%int64(period) == 0 {
			fmt.Print("Send data")
			if err := agent.SendMetrics(); err != nil {
				return err
			}
		}
		fmt.Println(*val.Delta)
	}

}

func main() {
	agent := Agent{}
	if err := agent.MainLoop(); err != nil {
		fmt.Print(err.Error())
	}
}
