package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

type Agent struct {
	metricStorage store.StorageInterface
	config        *AgentConfig
}

type AgentConfig struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func NewAgentConfig() (*AgentConfig, error) {

	newConfig := AgentConfig{}

	agentFlags := flag.NewFlagSet("Agent flags", flag.ContinueOnError)
	agentFlags.StringVar(&newConfig.serverAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	interval := uint(0)
	agentFlags.UintVar(&interval, "r", 10, "report interval in seconds. default 10.")
	newConfig.reportInterval = time.Duration(interval) * time.Second
	agentFlags.UintVar(&interval, "p", 2, "poll interval in seconds. default 2.")
	newConfig.pollInterval = time.Duration(interval) * time.Second
	agentFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		newConfig.serverAddress = envServerAddr
	}

	envPollINterval, ok := os.LookupEnv("POLL_INTERVAL")
	if ok {
		interval, err := strconv.ParseUint(envPollINterval, 10, 32)
		if err != nil {
			return nil, errors.New("can`t convert STORE_INTERVAL env variable")
		}
		newConfig.pollInterval = time.Duration(interval) * time.Second
	}

	envReportINterval, ok := os.LookupEnv("REPORT_INTERVAL")
	if ok {
		interval, err := strconv.ParseUint(envReportINterval, 10, 32)
		if err != nil {
			return nil, errors.New("can`t convert STORE_INTERVAL env variable")
		}
		newConfig.reportInterval = time.Duration(interval) * time.Second
	}

	return &newConfig, nil
}

func NewAgent(storage store.StorageInterface) (*Agent, error) {

	newAgent := Agent{}
	var err error
	if newAgent.config, err = NewAgentConfig(); err != nil {
		return nil, err
	}
	newAgent.metricStorage = storage
	return &newAgent, nil
}

func (agent *Agent) UpdateMetrics() error {
	memStat := runtime.MemStats{}
	runtime.ReadMemStats(&memStat)
	value := reflect.ValueOf(memStat)

	allMetrics := agent.metricStorage.GetAllMetrics()

	for _, metrics := range *allMetrics {

		switch metrics.MType {
		case models.Gauge:
			{
				fieldValue := value.FieldByName(metrics.ID)
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
						return errors.New("wrong data type in source of gauge metrics")
					}
				}
				metrics.Value = &newValue
			}
		case models.Counter:
			{
				newValue := int64(0)
				if metrics.Delta == nil {
					newValue = 1
				} else {
					newValue = *metrics.Delta + 1
				}

				metrics.Delta = &newValue
			}
		}
		if err := agent.metricStorage.UpdateMetrics(metrics.ID, metrics); err != nil {
			return err
		}

	}
	return nil
}

func (agent *Agent) SendMetrics() error {
	requestPattern := "http://%s/update/%s/%s/%s"

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	allMetrics := agent.metricStorage.GetAllMetrics()
	for _, metrics := range *allMetrics {

		response, err := client.Post(fmt.Sprintf(requestPattern, agent.config.serverAddress, metrics.MType, metrics.ID, metrics.GetMetricsValue()), "Content-Type: text/plain", nil)
		if err != nil {
			return fmt.Errorf("error send metrics %s. error:  %s", metrics.ID, err.Error())
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("error update metrics %s on server. Response code: %d ", metrics.ID, response.StatusCode)
		}
	}
	return nil
}

func (agent *Agent) SendJSONMetrics(useCompression bool) error {

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	allMetrics := agent.metricStorage.GetAllMetrics()
	for _, metrics := range *allMetrics {

		buf, err := json.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("error marshal metrics to JSON. error: %s", err.Error())
		}

		var requestBody bytes.Buffer

		if useCompression {
			zw := gzip.NewWriter(&requestBody)
			if _, err := zw.Write(buf); err != nil {
				return fmt.Errorf("error compress metrics %s. error: %s", metrics.ID, err.Error())

			}
			if err := zw.Close(); err != nil {
				return fmt.Errorf("error close zip writer. error: %s", err.Error())
			}
		} else {
			requestBody.Write(buf)
		}
		req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.serverAddress+"/update/", &requestBody)
		if err != nil {
			return fmt.Errorf("error! create request. error: %s", err.Error())
		}
		req.Header.Set("Content-Type", "application/json")
		if useCompression {
			req.Header.Set("Content-Encoding", "gzip")
		}
		response, err := client.Do(req)

		if err != nil {
			return fmt.Errorf("error send metrics %s error %s", metrics.ID, err.Error())
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("error update metrics %s on server. Response code %d ", metrics.ID, response.StatusCode)
		}

	}
	return nil
}

func (agent *Agent) MainLoop() {

	poolTicker := time.NewTicker(agent.config.pollInterval)
	defer poolTicker.Stop()

	reportTicker := time.NewTicker(agent.config.reportInterval)
	defer reportTicker.Stop()

	ch := make(chan bool)
	done := make(chan bool)

	go func() {
		for {
			select {
			case v := <-done:
				ch <- v
			case <-poolTicker.C:
				if err := agent.UpdateMetrics(); err != nil {
					log.Printf("error update metrics. error: %s", err.Error())
				}
			}
		}

	}()

	go func() {
		for {
			select {
			case v := <-done:
				ch <- v
			case <-reportTicker.C:
				if err := agent.SendJSONMetrics(false); err != nil {
					log.Printf("error send metrics. error: %s", err.Error())
				}
			}
		}

	}()
	<-ch
}
