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
	"net/http"
	"os"
	"strconv"

	chi "github.com/go-chi/chi/v5"
	"github.com/skdiver33/metrics-collector/internal/audit"
	"github.com/skdiver33/metrics-collector/internal/store"
)

// ServerConfig - структура с конфигурацией сервера.
// generate:reset
type ServerConfig struct {
	ListenAddress   string
	StoreInterval   uint
	StorageDumpPath string
	IsDumpRestore   bool
	SQLDBAddress    string
	SigningKey      string
	AuditURL        string
	AuditFile       string
	keyFile         string
}

func newServerConfig() *ServerConfig {

	serverConfig := ServerConfig{}
	serverFlags := flag.NewFlagSet("Server config flags", flag.ContinueOnError)
	serverFlags.StringVar(&serverConfig.ListenAddress, "a", "localhost:8080", "adress for start server in form ip:port. default localhost:8080")
	serverFlags.UintVar(&serverConfig.StoreInterval, "i", 10, "store interval in seconds. default 300.")
	serverFlags.StringVar(&serverConfig.StorageDumpPath, "f", "", "path to file for storage dump. Default empty and disable.")
	serverFlags.StringVar(&serverConfig.SigningKey, "k", "", "key for check signing response body. Default empty")
	serverFlags.StringVar(&serverConfig.SQLDBAddress, "d", "", "DB connection string. Default - empty and disable.")
	serverFlags.BoolVar(&serverConfig.IsDumpRestore, "r", false, "use dump for restore storage state")
	serverFlags.StringVar(&serverConfig.AuditFile, "audit-file", "", "filename for audit file. Default empty")
	serverFlags.StringVar(&serverConfig.AuditURL, "audit-url", "", "url for audit service. Default empty")
	serverFlags.StringVar(&serverConfig.keyFile, "crypto-key", "../../keys/private.pem", "private key path. Default empty")
	serverFlags.Parse(os.Args[1:])

	envServerAddr, ok := os.LookupEnv("ADDRESS")
	if ok {
		serverConfig.ListenAddress = envServerAddr
	}

	envSigningKey, ok := os.LookupEnv("KEY")
	if ok {
		serverConfig.SigningKey = envSigningKey
	}
	envKeyFile, ok := os.LookupEnv("CRYPTO_KEY")
	if ok {
		serverConfig.keyFile = envKeyFile
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
	envAuditFile, ok := os.LookupEnv("AUDIT_FILE")
	if ok {
		serverConfig.AuditFile = envAuditFile
	}
	envAuditURL, ok := os.LookupEnv("AUDIT_URL")
	if ok {
		serverConfig.AuditURL = envAuditURL

	}

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

	if newServer.Storage == nil {
		log.Fatalf("fatal error. service storage nil, after creation")
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

	if newServer.Config.keyFile != "" {
		newHandler.privateKey, err = readPrivateKey(newServerConfig().keyFile)
		if err != nil {
			return nil, err
		}
	}

	newHandler.signingKey = newServer.Config.SigningKey

	newRouter := chi.NewRouter()
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
