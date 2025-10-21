package main

import (
	"log"
	"net/http"
	"time"

	_ "net/http/pprof"

	"github.com/skdiver33/metrics-collector/internal/server"
)

func main() {

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	server, err := server.NewServer()
	if err != nil {
		panic(err.Error())
	}
	if server.Config.StorageDumpPath != "" {
		go func() {
			for {
				time.Sleep(time.Duration(server.Config.StoreInterval) * time.Second)
				server.WriteStorageDump()
			}
		}()
	}

	if err := http.ListenAndServe(server.Config.ListenAddress, server.HandlersRouter); err != nil {
		panic(err.Error())
	}

}
