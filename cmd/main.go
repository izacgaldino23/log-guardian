package main

import (
	"context"
	"log"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"time"
)

var config *domain.RuntimeConfig

func init() {
	var err error
	// load config
	config, err = domain.LoadConfigs()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orchestrator := application.NewOrchestrator(ctx, config, 5*time.Second)
	orchestrator.Execute()
}
