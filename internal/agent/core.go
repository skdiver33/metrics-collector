// Package agent содержит реализацию агента сбора метрик.
package agent

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/skdiver33/metrics-collector/internal/misc"
	"github.com/skdiver33/metrics-collector/internal/store"

	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/skdiver33/metrics-collector/models"

	cryptoRand "crypto/rand"
)

type Agent struct {
	metricStorage store.StorageInterface
	config        *AgentConfig
	pubKey        *rsa.PublicKey
}

type AgentConfig struct {
	ServerAddress  string `json:"address" env:"ADDRESS"`
	PollInterval   uint   `json:"poll_interval" env:"POLL_INTERVAL"`
	ReportInterval uint   `json:"report_interval" env:"REPORT_INTERVAL"`
	KeyFile        string `json:"crypto_key" env:"CRYPTO_KEY"`
	SigningKey     string `env:"KEY"`
	RateLimit      uint   `env:"RATE_LIMIT"`
}

func NewAgentConfig() (*AgentConfig, error) {

	newConfig := AgentConfig{}
	var configPath string
	agentFlags := flag.NewFlagSet("Agent flags", flag.ContinueOnError)
	agentFlags.StringVar(&newConfig.ServerAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	agentFlags.UintVar(&newConfig.ReportInterval, "r", 10, "report interval in seconds. default 10.")
	agentFlags.UintVar(&newConfig.PollInterval, "p", 2, "poll interval in seconds. default 2.")
	agentFlags.StringVar(&newConfig.SigningKey, "k", "", "key for signing data")
	agentFlags.UintVar(&newConfig.RateLimit, "l", 4, "amount sendings threads. default 4.")
	agentFlags.StringVar(&newConfig.KeyFile, "crypto-key", "", "path to public key")
	agentFlags.StringVar(&configPath, "c", "", "path to config file")
	agentFlags.StringVar(&configPath, "config", "", "path to config file")
	agentFlags.Parse(os.Args[1:])

	if confPath, ok := os.LookupEnv("CONFIG"); ok {
		configPath = confPath
	}

	if len(configPath) != 0 {
		err := cleanenv.ReadConfig(configPath, &newConfig)
		if err != nil {
			log.Printf("error read config file. %s", err.Error())
		}
	}
	agentFlags.Parse(os.Args[1:])
	cleanenv.ReadEnv(&newConfig)

	return &newConfig, nil
}

func NewAgent(storage store.StorageInterface) (*Agent, error) {

	newAgent := Agent{}
	var err error
	if newAgent.config, err = NewAgentConfig(); err != nil {
		return nil, err
	}
	newAgent.metricStorage = storage
	if newAgent.config.KeyFile != "" {
		newAgent.pubKey, err = readPubKey(newAgent.config.KeyFile)
		if err != nil {
			return nil, err
		}
	}
	return &newAgent, nil
}

func readPubKey(filePath string) (*rsa.PublicKey, error) {
	keyBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(keyBytes)
	if pemBlock == nil {
		return nil, errors.New("error decode pem block")
	}
	key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("is not public key")
	}
	return rsaKey, nil

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

		response, err := client.Post(fmt.Sprintf(requestPattern, agent.config.ServerAddress, metrics.MType, metrics.ID, metrics.GetMetricsValue()), "Content-Type: text/plain", nil)
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

	jsonbuf, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("error marshal metrics to JSON. error: %w", err)
	}
	buf := make([]byte, len(jsonbuf))
	if agent.config.KeyFile != "" {
		buf, err = rsa.EncryptPKCS1v15(cryptoRand.Reader, agent.pubKey, jsonbuf)
		if err != nil {
			log.Fatal(err)
		}
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
	req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.ServerAddress+"/update/", &requestBody)
	if err != nil {
		return fmt.Errorf("error! create request. error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if useCompression {
		req.Header.Set("Content-Encoding", "gzip")
	}
	if agent.config.SigningKey != "" {
		bodyHash := misc.GetRequestHash(requestBody.Bytes(), agent.config.SigningKey)
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

	req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.ServerAddress+"/updates/", &requestBody)
	if err != nil {
		return fmt.Errorf("error! create request. error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if agent.config.SigningKey != "" {
		bodyHash := misc.GetRequestHash(requestBody.Bytes(), agent.config.SigningKey)
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
	for i := 0; i < int(agent.config.RateLimit); i++ {
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

	poolTicker := time.NewTicker(time.Duration(agent.config.PollInterval) * time.Second)
	defer poolTicker.Stop()

	reportTicker := time.NewTicker(time.Duration(agent.config.ReportInterval) * time.Second)
	defer reportTicker.Stop()

	done := make(chan bool)

	termCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-termCtx.Done():
				return
			case <-done:
				return
			case <-poolTicker.C:
				mu.Lock()
				if err := agent.UpdateMetrics(); err != nil {
					log.Printf("error update metrics. error: %s", err.Error())
					close(done)
					return
				}
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
		defer wg.Done()
		for {
			select {
			case <-termCtx.Done():
				return
			case <-done:
				return
			case <-reportTicker.C:
				mu.Lock()
				err := misc.RetriableErrorHandler(termCtx, agent.SendMetricsConsistently)
				mu.Unlock()
				if err != nil {
					log.Println("error send data to server. agent down.")
					close(done)
					return
				}

			}
		}

	}()
	wg.Wait()
}
