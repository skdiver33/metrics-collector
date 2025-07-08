package server

import (
	"flag"
	"net/http"
	"os"
	"strconv"

	chi "github.com/go-chi/chi/v5"
	"github.com/skdiver33/metrics-collector/internal/store"
)

type ServerConfig struct {
	ListenAddress   string
	StoreInterval   uint
	StorageDumpPath string
	IsDumpRestore   bool
}

func newServerConfig() *ServerConfig {

	serverConfig := ServerConfig{}
	serverFlags := flag.NewFlagSet("Server config flags", flag.ContinueOnError)
	serverFlags.StringVar(&serverConfig.ListenAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	serverFlags.UintVar(&serverConfig.StoreInterval, "i", 10, "store interval in seconds. default 300.")
	serverFlags.StringVar(&serverConfig.StorageDumpPath, "f", "/tmp/storage_dump.json", "path to file for storage dump")
	serverFlags.BoolVar(&serverConfig.IsDumpRestore, "r", false, "use dump for restore storage state")
	serverFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		serverConfig.ListenAddress = envServerAddr
	}

	envStoreINterval, ok := os.LookupEnv("STORE_INTERVAL")
	if ok {
		interval, err := strconv.ParseUint(envStoreINterval, 10, 32)
		if err != nil {
			panic("can`t convert STORE_INTERVAL env variable")
		}
		serverConfig.StoreInterval = uint(interval)
	}

	envFileStoragePAth, ok := os.LookupEnv("FILE_STORAGE_PATH")
	if ok {
		serverConfig.StorageDumpPath = envFileStoragePAth
	}

	envIsRestoreFlag, ok := os.LookupEnv("RESTORE")
	if ok {
		isRestore, err := strconv.ParseBool(envIsRestoreFlag)
		if err != nil {
			panic("can`t convert RESTORE env variable")
		}
		serverConfig.IsDumpRestore = isRestore
	}

	return &serverConfig
}

type Server struct {
	Config         *ServerConfig
	Storage        store.StorageInterface
	HandlersRouter http.Handler
}

func NewServer() (*Server, error) {

	newServer := Server{}

	newServer.Config = newServerConfig()

	newStorage, err := store.NewMemStorage()
	if err != nil {
		panic("error initialize storage in server")
	}
	newServer.Storage = newStorage

	if newServer.Config.IsDumpRestore {
		newServer.Storage.RestoreMetricsFromFile(newServer.Config.StorageDumpPath)
	}

	newHandler, err := NewMetricsHandler(newStorage)
	if err != nil {
		return nil, err
	}
	newRouter := chi.NewRouter()
	newRouter.Use(newHandler.RequestLogger)
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
	newServer.HandlersRouter = newRouter
	return &newServer, nil

}

func (server *Server) WriteStorageDump() {
	server.Storage.SaveMetricsInFile(server.Config.StorageDumpPath)
}
