package server

import (
	"fmt"
	"net/http"
)

func ExampleMetricsHandler_GetAllMetrics() {
	//Run server
	go func() {
		server, err := NewServer()
		if err != nil {
			fmt.Printf("cannot create server %s", err.Error())
			return
		}
		http.ListenAndServe(server.Config.ListenAddress, server.HandlersRouter)
	}()
	resp, err := http.Get("localhost:8080/")
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
	fmt.Print(resp.StatusCode)
}

func ExampleMetricsHandler_SetMetrics() {
	//Run server
	go func() {
		server, err := NewServer()
		if err != nil {
			fmt.Printf("cannot create server %s", err.Error())
			return
		}
		http.ListenAndServe(server.Config.ListenAddress, server.HandlersRouter)
	}()
	resp, err := http.Post("localhost:8080/update/gauge/metricsName/123.4", "pain/text", nil)
	if err != nil {
		fmt.Printf("%s", err.Error())
	}
	defer resp.Body.Close()
	fmt.Print(resp.StatusCode)
}
