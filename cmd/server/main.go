package main

import (
	"net/http"
	"time"

	"github.com/skdiver33/metrics-collector/internal/server"
)

func main() {

	server, err := server.NewServer()
	if err != nil {
		panic(err.Error())
	}
	go func() {
		for {
			time.Sleep(time.Duration(server.Config.StoreInterval) * time.Second)
			server.WriteStorageDump()
		}
	}()

	if err := http.ListenAndServe(server.Config.ListenAddress, server.HandlersRouter); err != nil {
		panic(err.Error())
	}

}
