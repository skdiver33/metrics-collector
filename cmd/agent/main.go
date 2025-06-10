package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type Agent struct {
	metricStorage store.MemStorage
}

var (
	serverAddress  string
	pollInterval   uint
	reportInterval uint
)

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
	requestPattern := "http://%s/%s/%s/%s"

	allMMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}

	tr := &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: tr}

	//remake with new interface of metrics
	for _, name := range allMMetricsName {
		currentMetrics, err := agent.metricStorage.GetMetricsValue(name)
		if err != nil {
			fmt.Print(err.Error())
			return err
		}

		if currentMetrics.Value == nil && currentMetrics.Delta == nil {
			return errors.New("error! update metrics before send")
		}
		value := currentMetrics.GetMetricsValue()
		response, err := client.Post(fmt.Sprintf(requestPattern, serverAddress, currentMetrics.MType, name, value), "Content-Type: text/plain", nil)
		if err != nil {
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
	// poolInterval := 2
	// reportInterval := 10
	//period := reportInterval / pollInterval

	min := min(reportInterval, pollInterval)
	reportPeriod := reportInterval / min
	pollPeripd := pollInterval / min
	count := 0

	for {
		count++

		time.Sleep(time.Duration(min) * time.Second)
		if count%int(pollPeripd) == 0 {
			fmt.Printf("Poll %d\n", count/int(pollPeripd))
			if err := agent.UpdateMetrics(); err != nil {
				return err
			}
		}
		if count%int(reportPeriod) == 0 {
			fmt.Print("Send data\n")
			fmt.Printf("Send %d\n", count/int(reportPeriod))
			if err := agent.SendMetrics(); err != nil {
				return err
			}
		}
	}

}

func main() {
	agentFlags := flag.NewFlagSet("Agent flags", flag.ExitOnError)
	agentFlags.StringVar(&serverAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	agentFlags.UintVar(&reportInterval, "r", 10, "report interval in seconds. default 10.")
	agentFlags.UintVar(&pollInterval, "p", 2, "poll interval in seconds. default 2.")
	agentFlags.Parse(os.Args[1:])
	agent := Agent{}
	if err := agent.MainLoop(); err != nil {
		fmt.Print(err.Error())
	}
}
