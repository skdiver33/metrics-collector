// Package server модуль core содержит реализацию веб-сервера.
// Предоставляет возможность запустить веб-сервер с заданными параметрами.
// Параметры имеют значения по умолчанию, а так же могут быть заданы через аргументы запуска
// или аргументы командной строки.
package server

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"log"
	"net"
	"net/http"
	"os"

	chi "github.com/go-chi/chi/v5"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/skdiver33/metrics-collector/internal/audit"
	"github.com/skdiver33/metrics-collector/internal/store"
)

// ServerConfig - структура с конфигурацией сервера.
// generate:reset
type ServerConfig struct {
	ListenAddress     string `json:"address" env:"ADDRESS"`
	GRPCListenAddress string `json:"grpc_address" env:"GRPC_ADDRESS"`
	StoreInterval     uint   `json:"store_interval" env:"STORE_INTERVAL"`
	StorageDumpPath   string `json:"store_file" env:"FILE_STORAGE_PATH"`
	IsDumpRestore     bool   `json:"restore" env:"RESTORE"`
	SQLDBAddress      string `json:"database_dsn" env:"DATABASE_DSN"`
	SigningKey        string `env:"KEY"`
	AuditURL          string `env:"AUDIT_URL"`
	AuditFile         string `env:"AUDIT_FILE"`
	KeyFile           string `json:"crypto_key" env:"CRYPTO_KEY"`
	TrustedSubnet     string `json:"trusted_subnet" env:"TRUSTED_SUBNET"`
}

func newServerConfig() *ServerConfig {

	serverConfig := ServerConfig{}
	var configPath string

	serverFlags := flag.NewFlagSet("Server config flags", 0)
	serverFlags.StringVar(&serverConfig.ListenAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	serverFlags.StringVar(&serverConfig.GRPCListenAddress, "grpc", "localhost:3080", "adress for start grpc server in form ip:port. default localhost:3080")
	serverFlags.UintVar(&serverConfig.StoreInterval, "i", 10, "store interval in seconds. default 300.")
	serverFlags.StringVar(&serverConfig.StorageDumpPath, "f", "", "path to file for storage dump. Default empty and disable.")
	serverFlags.StringVar(&serverConfig.SigningKey, "k", "", "key for check signing response body. Default empty")
	serverFlags.StringVar(&serverConfig.SQLDBAddress, "d", "", "DB connection string. Default - empty and disable.")
	serverFlags.BoolVar(&serverConfig.IsDumpRestore, "r", false, "use dump for restore storage state")
	serverFlags.StringVar(&serverConfig.AuditFile, "audit-file", "", "filename for audit file. Default empty")
	serverFlags.StringVar(&serverConfig.AuditURL, "audit-url", "", "url for audit service. Default empty")
	serverFlags.StringVar(&serverConfig.KeyFile, "crypto-key", "", "private key path. Default empty")
	serverFlags.StringVar(&serverConfig.TrustedSubnet, "t", "", "trusted subnet")
	serverFlags.StringVar(&configPath, "c", "", "path to config file")
	serverFlags.StringVar(&configPath, "config", "", "path to config file")
	serverFlags.Parse(os.Args[1:])

	if confPath, ok := os.LookupEnv("CONFIG"); ok {
		configPath = confPath
	}
	if len(configPath) != 0 {
		err := cleanenv.ReadConfig(configPath, &serverConfig)
		if err != nil {
			log.Printf("error read config file. %s", err.Error())
		}
	}
	serverFlags.Parse(os.Args[1:])
	cleanenv.ReadEnv(&serverConfig)

	return &serverConfig
}

// Server - структура агрегирующая необходимые для запуска сервера компоненты.
// generate:reset
type Server struct {
	// Конфиг сервера
	Config *ServerConfig
	// Заданное хранилище для хранения метрик
	Storage store.StorageInterface
	// Обработчик http запросов
	HandlersRouter http.Handler
}

func readPrivateKey(filePath string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(keyBytes)
	if pemBlock == nil {
		return nil, errors.New("error decode pem block")
	}
	key, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// NewServer - возвращает указатель на новый сервер.
func NewServer() (*Server, error) {
	newServer := Server{}
	newServer.Config = newServerConfig()
	if newServer.Config.SQLDBAddress != "" {
		newStorage, err := store.NewSQLStorage(newServer.Config.SQLDBAddress)
		if err != nil {
			return nil, err
		}
		newServer.Storage = newStorage

	} else {
		newStorage, err := store.NewMemStorage()
		if err != nil {
			return nil, err
		}
		newServer.Storage = newStorage
	}

	if newServer.Config.StorageDumpPath != "" && newServer.Config.IsDumpRestore {
		newServer.Storage.RestoreDBDump(context.Background(), newServer.Config.StorageDumpPath)
	}

	if newServer.Storage == nil {
		return nil, errors.New("fatal error. service storage nil, after creation")
	}

	newHandler, err := NewMetricsHandler(newServer.Storage)
	if err != nil {
		return nil, err
	}

	if newServer.Config.AuditFile != "" {
		fo := audit.NewFileSubscriber(newServer.Config.AuditFile)
		newHandler.auditor.Register(fo)
	}

	if newServer.Config.AuditURL != "" {
		uo := audit.NewURLSubscriber(newServer.Config.AuditURL)
		newHandler.auditor.Register(uo)
	}

	if newServer.Config.TrustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(newServerConfig().TrustedSubnet)
		if err != nil {
			return nil, err
		}
		newHandler.trustNet = ipNet
	}

	if newServer.Config.KeyFile != "" {
		newHandler.privateKey, err = readPrivateKey(newServerConfig().KeyFile)
		if err != nil {
			return nil, err
		}
	}

	newHandler.signingKey = newServer.Config.SigningKey

	newRouter := chi.NewRouter()
	newRouter.Use(newHandler.CheckingIpOnTrust)
	newRouter.Use(newHandler.RequestLogger)
	newRouter.Use(newHandler.SigningHandle)
	newRouter.Use(newHandler.GzipHandle)
	newRouter.Use(newHandler.DecryptHandle)
	newRouter.Route("/", func(r chi.Router) {
		r.Get("/", newHandler.GetAllMetrics)
		r.Get("/ping", newHandler.PingDB)
		r.Route("/value", func(r chi.Router) {
			r.Post("/", newHandler.GetJSONMetrics)
			r.Get("/{metricsType}/{metricsName}", newHandler.GetMetrics)
		})
		r.Post("/updates/", newHandler.SetBunchMetrics)
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
