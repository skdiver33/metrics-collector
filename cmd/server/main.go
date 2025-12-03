package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/skdiver33/metrics-collector/internal/grpcserver"
	"github.com/skdiver33/metrics-collector/internal/server"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	server, err := server.NewServer()
	if err != nil {
		log.Fatal(err.Error())
	}

	retCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	if server.Config.StorageDumpPath != "" {
		writeTicker := time.NewTicker(time.Duration(server.Config.StoreInterval) * time.Second)
		defer writeTicker.Stop()
		go func(ctx context.Context) {
			for {
				select {
				case <-writeTicker.C:
					server.WriteStorageDump()
				case <-ctx.Done():
					return
				}
			}
		}(retCtx)
	}

	srv := &http.Server{
		Addr:    server.Config.ListenAddress,
		Handler: server.HandlersRouter,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("error start http server: %s\n", err.Error())
			stop()
		}
	}()
	grpcServer := grpcserver.NewMetricsServer(server.Config, server.Storage)
	go func() {
		time.Sleep(time.Second)
		if err := grpcServer.Run(); err != nil {
			log.Printf("error start grpc server: %s\n", err.Error())
			stop()
		}
	}()
	<-retCtx.Done()
	stop()
	log.Println("Server shutdowning....")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
	grpcServer.Stop()
	server.Storage.CloseConnection()
	log.Println("Server stop")
}
