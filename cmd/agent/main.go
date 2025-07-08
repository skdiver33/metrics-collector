package main

import (
	"log"

	"github.com/skdiver33/metrics-collector/internal/agent"
	"github.com/skdiver33/metrics-collector/internal/store"
)

func main() {

	agentStorage, err := store.NewMemStorage()
	if err != nil {
		log.Fatal(err.Error())
	}
	agent, err := agent.NewAgent(agentStorage)
	if err != nil {
		log.Fatal(err.Error())
	}

	agent.MainLoop()
}
