package agent

import (
	"bytes"
	"compress/gzip"
	"context"
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
	"sync"
	"time"

	"github.com/skdiver33/metrics-collector/internal/misc"
	"github.com/skdiver33/metrics-collector/internal/store"

	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
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
	signingKey     string
	rateLimit      uint
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
	agentFlags.StringVar(&newConfig.signingKey, "k", "", "key for signing data")
	agentFlags.UintVar(&newConfig.rateLimit, "l", 4, "amount sendings threads. default 4.")
	agentFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		newConfig.serverAddress = envServerAddr
	}

	envSigningKey, ok := os.LookupEnv("KEY")
	if ok {
		newConfig.signingKey = envSigningKey
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

	envRateLimit, ok := os.LookupEnv("RATE_LIMIT")
	if ok {
		limit, err := strconv.ParseUint(envRateLimit, 10, 32)
		newConfig.rateLimit = uint(limit)
		if err != nil {
			return nil, errors.New("can`t convert RATE_LIMIT env variable")
		}
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

func (agent *Agent) RuntimeMetricsUpdate() error {
	v, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("error get runtime metrics(memmory info). error:  %w", err)
	}
	metrics := models.Metrics{MType: models.Gauge}
	metrics.ID = "TotalMemory"
	val := float64(v.Total)
	metrics.Value = &val
	agent.metricStorage.UpdateMetrics(context.Background(), metrics)
	metrics.ID = "FreeMemory"
	val = float64(v.Free)
	metrics.Value = &val
	agent.metricStorage.UpdateMetrics(context.Background(), metrics)
	loadInfo, err := load.Avg()
	if err != nil {
		return fmt.Errorf("error get runtime metrics(load info). error:  %w", err)
	}
	metrics.ID = "CPUutilization1"
	val = loadInfo.Load1
	metrics.Value = &val
	agent.metricStorage.UpdateMetrics(context.Background(), metrics)
	return nil
}

func (agent *Agent) UpdateMetrics() error {
	memStat := runtime.MemStats{}
	runtime.ReadMemStats(&memStat)
	value := reflect.ValueOf(memStat)
	allMetrics := agent.metricStorage.GetAllMetrics(context.Background())

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
		if err := agent.metricStorage.UpdateMetrics(context.Background(), metrics); err != nil {
			return err
		}

	}
	return nil
}

func (agent *Agent) SendMetrics() error {
	requestPattern := "http://%s/update/%s/%s/%s"

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	allMetrics := agent.metricStorage.GetAllMetrics(context.Background())
	for _, metrics := range *allMetrics {

		response, err := client.Post(fmt.Sprintf(requestPattern, agent.config.serverAddress, metrics.MType, metrics.ID, metrics.GetMetricsValue()), "Content-Type: text/plain", nil)
		if err != nil {
			return fmt.Errorf("error send metrics %s. error:  %w", metrics.ID, err)
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("error update metrics %s on server. Response code: %d ", metrics.ID, response.StatusCode)
		}
	}
	return nil
}

func (agent *Agent) SendJSONMetrics(metrics *models.Metrics) error {

	useCompression := true

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	buf, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshal metrics to JSON. error: %w", err)
	}

	var requestBody bytes.Buffer

	if useCompression {
		zw := gzip.NewWriter(&requestBody)
		if _, err := zw.Write(buf); err != nil {
			return fmt.Errorf("error compress metrics %s. error: %w", metrics.ID, err)

		}
		if err := zw.Close(); err != nil {
			return fmt.Errorf("error close zip writer. error: %w", err)
		}
	} else {
		requestBody.Write(buf)
	}
	req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.serverAddress+"/update/", &requestBody)
	if err != nil {
		return fmt.Errorf("error! create request. error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if useCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}
	if agent.config.signingKey != "" {
		bodyHash := misc.GetRequestHash(requestBody.Bytes(), agent.config.signingKey)
		req.Header.Set("HashSHA256", bodyHash)
	}
	response, err := client.Do(req)

	if err != nil {
		return misc.NewRetrialableError(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error update metrics %s on server. Response code %d ", metrics.ID, response.StatusCode)
	}

	return nil
}

func (agent *Agent) SendBunchMetrics() error {

	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	allMetrics := agent.metricStorage.GetAllMetrics(context.Background())

	buf, err := json.Marshal(allMetrics)
	if err != nil {
		return fmt.Errorf("error marshal all metrics to JSON. error: %w", err)
	}

	var requestBody bytes.Buffer
	zw := gzip.NewWriter(&requestBody)
	if _, err := zw.Write(buf); err != nil {
		return fmt.Errorf("error compress all metrics. error: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("error close zip writer. error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.serverAddress+"/updates/", &requestBody)
	if err != nil {
		return fmt.Errorf("error! create request. error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if agent.config.signingKey != "" {
		bodyHash := misc.GetRequestHash(requestBody.Bytes(), agent.config.signingKey)
		req.Header.Set("HashSHA256", bodyHash)
	}
	response, err := client.Do(req)
	if err != nil {
		return misc.NewRetrialableError(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error update all metrics on server. Response code %d ", response.StatusCode)
	}

	return nil
}

func (agent *Agent) SendMetricsConsistently() error {
	allMetrics := agent.metricStorage.GetAllMetrics(context.Background())
	for _, metrics := range *allMetrics {
		err := agent.SendJSONMetrics(&metrics)
		if err != nil {
			return err
		}
	}
	return nil
}

type Result struct {
	status string
	err    error
}

func (agent *Agent) sendOneMetrics(jobs <-chan models.Metrics, result chan<- Result) {
	res := Result{}
	for metrics := range jobs {
		err := agent.SendJSONMetrics(&metrics)
		if err != nil {
			res.err = err
			result <- res
		}
		res.status = "Ok"
		result <- res
	}
}

func (agent *Agent) SendMetricsParallel() error {
	allMetrics := agent.metricStorage.GetAllMetrics(context.Background())
	numJobs := len(*allMetrics)
	metricsChannel := make(chan models.Metrics, numJobs)
	resultChannel := make(chan Result, numJobs)
	for i := 0; i < int(agent.config.rateLimit); i++ {
		go agent.sendOneMetrics(metricsChannel, resultChannel)
	}

	for _, metrics := range *allMetrics {
		metricsChannel <- metrics
	}
	close(metricsChannel)

	for i := 0; i < numJobs; i++ {
		res := <-resultChannel
		if res.err != nil {
			return res.err
		}
	}
	return nil
}

func (agent *Agent) MainLoop() {
	var mu sync.Mutex

	poolTicker := time.NewTicker(agent.config.pollInterval)
	defer poolTicker.Stop()

	reportTicker := time.NewTicker(agent.config.reportInterval)
	defer reportTicker.Stop()

	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {

		for {
			select {
			case <-done:
				wg.Done()
				return
			case <-poolTicker.C:
				mu.Lock()
				if err := agent.UpdateMetrics(); err != nil {
					log.Printf("error update metrics. error: %s", err.Error())
					close(done)
					return
				}
				mu.Unlock()
			}
		}

	}()
	wg.Add(1)
	go func() {

		for {
			select {
			case <-done:
				wg.Done()
				return
			case <-poolTicker.C:
				mu.Lock()
				if err := agent.RuntimeMetricsUpdate(); err != nil {
					log.Printf("error runtime update metrics. error: %s", err.Error())
					close(done)
					return
				}
				mu.Unlock()

			}
		}

	}()

	wg.Add(1)
	go func() {

		for {
			select {
			case <-done:
				wg.Done()
				return
			case <-reportTicker.C:
				mu.Lock()
				err := misc.RetriableErrorHandler(agent.SendMetricsParallel)
				mu.Unlock()
				if err != nil {
					log.Println("error send data to server. agent down.")
					close(done)
					wg.Done()
					return
				}

			}
		}

	}()
	wg.Wait()
}
