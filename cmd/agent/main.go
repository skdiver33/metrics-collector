package main

import (
	"fmt"
	"log"

	"github.com/skdiver33/metrics-collector/internal/agent"
	"github.com/skdiver33/metrics-collector/internal/store"
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
