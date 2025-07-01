package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	serverHandlers "github.com/skdiver33/metrics-collector/internal/server"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"
)

// func MetricsRouter() (*chi.Mux, error) {
// 	newMetricsStorage, err := store.NewMemStorage()
// 	if err != nil {
// 		return nil, err
// 	}

// 	handler, err := serverHandlers.NewMetricsHandler(newMetricsStorage)
// 	if err != nil {
// 		return nil, err
// 	}
// 	r := chi.NewRouter()
// 	r.Use(handler.RequestLogger)
// 	r.Use(handler.GzipHandle)
// 	r.Route("/", func(r chi.Router) {
// 		r.Get("/", handler.GetAllMetrics)
// 		r.Route("/value", func(r chi.Router) {
// 			r.Post("/", handler.GetJSONMetrics)
// 			r.Get("/{metricsType}/{metricsName}", handler.GetMetrics)
// 		})
// 		r.Route("/update", func(r chi.Router) {
// 			r.Post("/", handler.SetJSONMetrics)
// 			r.Post("/{metricsType}/{metricsName}/{metricsValue}", handler.SetMetrics)
// 		})
// 	})
// 	return r, nil
// }

type ServerConfig struct {
	listenAddress   string
	storeInterval   uint
	storageDumpPath string
	isDumpRestore   bool
}

func newServerConfig() *ServerConfig {

	serverConfig := ServerConfig{}
	serverFlags := flag.NewFlagSet("Server config flags", flag.PanicOnError)
	serverFlags.StringVar(&serverConfig.listenAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	serverFlags.UintVar(&serverConfig.storeInterval, "i", 10, "store interval in seconds. default 300.")
	serverFlags.StringVar(&serverConfig.storageDumpPath, "f", "/tmp/storage_dump.json", "path to file for storage dump")
	serverFlags.BoolVar(&serverConfig.isDumpRestore, "r", false, "use dump for restore storage state")

	// serverConfig.listenAddress = *serverFlags.String("a", "localhost:8080", "adress for start server")
	// serverConfig.storeInterval = *serverFlags.Uint("i", 300, "store interval, default 300 seconds")
	// serverConfig.storageDumpPath = *serverFlags.String("f", "./storage_dump.json", "path to file for storage dump")
	// serverConfig.isDumpRestore = *serverFlags.Bool("r", false, "use dump for restore storage state")
	serverFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		serverConfig.listenAddress = envServerAddr
	}

	envStoreINterval, ok := os.LookupEnv("STORE_INTERVAL")
	if ok {
		interval, err := strconv.ParseUint(envStoreINterval, 10, 32)
		if err != nil {
			panic("can`t convert STORE_INTERVAL env variable")
		}
		serverConfig.storeInterval = uint(interval)
	}

	envFileStoragePAth, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok {
		serverConfig.storageDumpPath = envFileStoragePAth
	}

	envIsRestoreFlag, ok := os.LookupEnv("RESTORE")
	if ok {
		isRestore, err := strconv.ParseBool(envIsRestoreFlag)
		if err != nil {
			panic("can`t convert RESTORE env variable")
		}
		serverConfig.isDumpRestore = isRestore
	}

	return &serverConfig
}

type Server struct {
	config  *ServerConfig
	storage store.StorageInterface
	router  http.Handler
}

func NewServer() (*Server, error) {
	newServer := Server{}

	serverConfig := newServerConfig()
	newServer.config = serverConfig

	newStorage, err := store.NewMemStorage()
	if err != nil {
		return nil, err
	}
	newServer.storage = newStorage

	if serverConfig.isDumpRestore {
		if _, err := os.Stat(serverConfig.storageDumpPath); err == nil {
			newServer.readStorageDump()
		} else if !errors.Is(err, os.ErrNotExist) {
			panic(err.Error())
		}

	}

	newHandler, err := serverHandlers.NewMetricsHandler(newStorage)
	if err != nil {
		return nil, err
	}
	newRouter := chi.NewRouter()
	//newRouter.Use(newHandler.RequestLogger)
	newRouter.Use(newHandler.GzipHandle)
	newRouter.Route("/", func(r chi.Router) {
		r.Get("/", newHandler.GetAllMetrics)
		r.Route("/value", func(r chi.Router) {
			r.Post("/", newHandler.GetJSONMetrics)
			r.Get("/{metricsType}/{metricsName}", newHandler.GetMetrics)
		})
		r.Route("/update", func(r chi.Router) {
			r.Post("/", newHandler.SetJSONMetrics)
			r.Post("/{metricsType}/{metricsName}/{metricsValue}", newHandler.SetMetrics)
		})
	})
	newServer.router = newRouter
	return &newServer, nil

}

func (server *Server) writeStorageDump() error {
	storage, ok := server.storage.(*store.MemStorage)
	if !ok {
		return errors.New("error get storage from server")
	}
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()
	data, err := json.Marshal(storage.Storage)
	if err != nil {
		panic("error convert to JSON all metrics")
	}
	err = os.WriteFile(server.config.storageDumpPath, data, 0666)
	if err != nil {
		panic("error write data to file")
	}
	return nil
}

func (server *Server) readStorageDump() error {
	readData, err := os.ReadFile(server.config.storageDumpPath)
	if err != nil {
		panic("cannot read data from file")
	}
	readStorage := make(map[string]models.Metrics)
	err = json.Unmarshal(readData, &readStorage)
	if err != nil {
		panic("cannot convert data from JSON")
	}
	for name, value := range readStorage {
		server.storage.UpdateMetrics(name, value)
	}
	return nil
}

func main() {

	server, err := NewServer()
	if err != nil {
		panic(err.Error())
	}
	fmt.Println(server.storage)
	go func() {
		for {
			time.Sleep(time.Duration(server.config.storeInterval) * time.Second)
			server.writeStorageDump()
		}
	}()

	if err := http.ListenAndServe(server.config.listenAddress, server.router); err != nil {
		panic(err.Error())
	}

}
