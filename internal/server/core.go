package server

import (
	"context"
	"flag"
	"log"
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
	SQLDBAddress    string
}

func newServerConfig() *ServerConfig {

	serverConfig := ServerConfig{}
	serverFlags := flag.NewFlagSet("Server config flags", flag.ContinueOnError)
	serverFlags.StringVar(&serverConfig.ListenAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	serverFlags.UintVar(&serverConfig.StoreInterval, "i", 10, "store interval in seconds. default 300.")
	serverFlags.StringVar(&serverConfig.StorageDumpPath, "f", "", "path to file for storage dump. Default empty and disable.")
	serverFlags.StringVar(&serverConfig.SQLDBAddress, "d", "", "DB connection string. Default - empty and disable.")
	//serverFlags.StringVar(&serverConfig.SQLDBAddress, "d", "host=localhost user=metricsuser password=secret dbname=metrics sslmode=disable", "DB connection string. Default - empty and disable.")
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

	envDBAddr, ok := os.LookupEnv("DATABASE_DSN")
	if ok {
		serverConfig.SQLDBAddress = envDBAddr
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

	if newServer.Config.SQLDBAddress != "" {
		newStorage, err := store.NewSQLStorage(newServer.Config.SQLDBAddress)
		if err != nil {
			log.Fatalf("error initialize storage in server %s", err.Error())
		}
		newServer.Storage = newStorage

	} else {
		newStorage, err := store.NewMemStorage()
		if err != nil {
			log.Fatalf("error initialize storage in server %s", err.Error())
		}
		newServer.Storage = newStorage
	}

	if newServer.Config.StorageDumpPath != "" && newServer.Config.IsDumpRestore {
		newServer.Storage.RestoreDBDump(context.Background(), newServer.Config.StorageDumpPath)
	}

	newHandler, err := NewMetricsHandler(newServer.Storage)
	if err != nil {
		return nil, err
	}
	newRouter := chi.NewRouter()
	newRouter.Use(newHandler.RequestLogger)
	newRouter.Use(newHandler.GzipHandle)
	newRouter.Route("/", func(r chi.Router) {
		r.Get("/", newHandler.GetAllMetrics)
		r.Get("/ping", newHandler.PingDB)
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
	server.Storage.CreateDBDump(context.Background(), server.Config.StorageDumpPath)
}
