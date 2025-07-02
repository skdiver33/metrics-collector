package main

import (
	"github.com/skdiver33/metrics-collector/internal/agent"
	"github.com/skdiver33/metrics-collector/internal/store"
)

func main() {

	agentStorage, err := store.NewMemStorage()
	if err != nil {
		panic(err.Error())
	}
	agent := agent.NewAgent(agentStorage)

	agent.MainLoop()
}
