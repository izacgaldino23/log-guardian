package main

import (
	"context"
	"log"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"time"
)

func init() {
	// load config
	_, err := domain.LoadConfigs()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orchestrator := application.NewOrchestrator(ctx, domain.GetConfig(), 5*time.Second)
	orchestrator.Execute()
}
