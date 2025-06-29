package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net"
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
	metricStorage *store.MemStorage
	config        *AgentConfig
}

func NewAgent() (*Agent, error) {
	newAgent := Agent{}

	config := AgentConfig{}
	agentFlags := flag.NewFlagSet("Agent flags", flag.PanicOnError)
	agentFlags.StringVar(&config.serverAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	agentFlags.UintVar(&config.reportInterval, "r", 10, "report interval in seconds. default 10.")
	agentFlags.UintVar(&config.pollInterval, "p", 2, "poll interval in seconds. default 2.")

	agentFlags.Parse(os.Args[1:])
	config.ParseEnvVariable()
	newAgent.config = &config

	newStorage, err := store.NewMemStorage()
	if err != nil {
		return nil, err
	}
	newAgent.metricStorage = newStorage

	return &newAgent, nil
}

type AgentConfig struct {
	serverAddress  string
	pollInterval   uint
	reportInterval uint
}

func (config *AgentConfig) ParseEnvVariable() {
	envNames := []string{"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL"}
	for index, envName := range envNames {
		varValue, ok := os.LookupEnv(envName)
		if !ok {
			continue
		}
		switch index {
		case 0:
			config.serverAddress = varValue
		case 1:
			{
				interval, err := strconv.Atoi(varValue)
				if err != nil || interval < 0 {
					fmt.Println("error value in environment variable REPORT_INTERVAL. Must be uint.")
					return
				}
				config.reportInterval = uint(interval)
			}
		case 2:
			{
				interval, err := strconv.Atoi(varValue)
				if err != nil || interval < 0 {
					fmt.Println("error value in environment variable POLL_INTERVAL. Must be uint.")
					return
				}
				config.pollInterval = uint(interval)
			}
		}

	}

}

type MetricsCollector interface {
	UpdateMetrics() error
	SendMetrics() error
}

func (agent *Agent) UpdateMetrics() error {
	memStat := runtime.MemStats{}
	runtime.ReadMemStats(&memStat)
	value := reflect.ValueOf(memStat)
	allMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}
	for _, name := range allMetricsName {
		currentMetrics, err := agent.metricStorage.GetMetrics(name)
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
		if err := agent.metricStorage.UpdateMetrics(name, currentMetrics); err != nil {
			return err
		}

	}
	return nil
}

func (agent *Agent) SendMetrics() error {
	requestPattern := "http://%s/update/%s/%s/%s"

	allMMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}

	tr := &http.Transport{
		//ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: tr}

	for _, name := range allMMetricsName {
		currentMetrics, err := agent.metricStorage.GetMetrics(name)
		if err != nil {
			fmt.Print(err.Error())
			return err
		}

		if currentMetrics.Value == nil && currentMetrics.Delta == nil {
			return errors.New("error! update metrics before send")
		}
		value := currentMetrics.GetMetricsValue()
		response, err := client.Post(fmt.Sprintf(requestPattern, agent.config.serverAddress, currentMetrics.MType, name, value), "Content-Type: text/plain", nil)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return errors.New("error update metrics on server!!! Response code not 200")
		}
	}
	return nil
}

func (agent *Agent) SendJSONMetrics() error {

	allMMetricsName, err := agent.metricStorage.GetAllMetricsNames()
	if err != nil {
		return err
	}

	tr := &http.Transport{
		//ResponseHeaderTimeout: 10 * time.Second,
		// MaxIdleConns:          1,
		// IdleConnTimeout: 30 * time.Second,
	}
	client := &http.Client{Transport: tr}
	for _, name := range allMMetricsName {
		currentMetrics, erro := agent.metricStorage.GetMetrics(name)
		if erro != nil {
			fmt.Print(erro.Error())
			return erro
		}
		buf, err := json.Marshal(currentMetrics)
		if err != nil {
			return errors.New("error! json marshaling")
		}
		//fmt.Println("Send data ", string(buf))
		requestBody := bytes.NewBuffer(buf)

		req, err := http.NewRequest(http.MethodPost, "http://"+agent.config.serverAddress+"/update/", requestBody)
		if err != nil {
			return errors.New("error! create request")
		}
		req.Header.Set("Content-Type", "application/json")
		req.Close = true

		//time.Sleep(1 * time.Second)
		response, err := client.Do(req)

		if err != nil {
			fmt.Printf("Client error send data %s \n", err.Error())
			//continue
			return err
		}
		defer response.Body.Close()

		answer, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("client error read body %s", err.Error())
		}
		fmt.Println(string(answer))

		if response.StatusCode != http.StatusOK {
			return errors.New("error update metrics on server!!! Response code not 200")
		}

	}
	return nil
}

func (agent *Agent) MainLoop() error {

	min := min(agent.config.reportInterval, agent.config.pollInterval)
	reportPeriod := agent.config.reportInterval / min
	pollPeriod := agent.config.pollInterval / min
	count := 0

	for {
		if val := isServerAvailabble(); val {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	for {
		count++

		time.Sleep(time.Duration(min) * time.Second)
		if count%int(pollPeriod) == 0 {
			if err := agent.UpdateMetrics(); err != nil {
				return err
			}
		}
		if count%int(reportPeriod) == 0 {
			if err := agent.SendJSONMetrics(); err != nil {
				return err
			}
		}
	}

}

func isServerAvailabble() bool {

	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		//conn.Close()
		return false
	}
	conn.Close()
	return true
}

func main() {
	agent, err := NewAgent()
	if err != nil {
		panic(err.Error())

	}

	if err := agent.MainLoop(); err != nil {
		panic(err.Error())
	}
}

// client := resty.New()
// 	for _, name := range allMMetricsName {
// 		currentMetrics, erro := agent.metricStorage.GetMetrics(name)
// 		if erro != nil {
// 			fmt.Print(erro.Error())
// 			return erro
// 		}
// 		//time.Sleep(100 * time.Millisecond)
// 		result := models.Metrics{}
// 		fmt.Println("send metrics  ", currentMetrics)
// 		var (
// 			resp *resty.Response
// 			err  error
// 		)
// 		resp, err = client.R().EnableTrace().
// 			SetHeader("Content-Type", "application/json").
// 			SetBody(currentMetrics).
// 			SetResult(&result). // or SetResult(&AuthSuccess{}).
// 			Post("http://localhost:8080/update/")

// 		client.R().TraceInfo()
// 		fmt.Println("post resp  ", resp, "   ", err)
// 		if err != nil {
// 			if err == io.EOF {
// 				continue
// 			}
// 			fmt.Println(err.Error())
// 			return err
// 		}
