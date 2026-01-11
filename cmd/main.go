package main

import (
	"context"
	"fmt"
	"log"
	"log-guardian/internal/adapters/input/stdin"
	"log-guardian/internal/core/domain"
	"os"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logChan := make(chan domain.LogEvent, 100)
	errChan := make(chan error)

	stdin := stdin.NewStdinIngestion(os.Stdin)

	stdin.Read(ctx, logChan, errChan)

	fmt.Println("Log Guardian is running")

	for {
		select {
		case event := <-logChan:
			fmt.Printf("Log received: [%s] %s\n", event.Severity, event.Message)
		case <-ctx.Done():
			return
		case err := <-errChan:
			close(logChan)
			log.Fatal(err)
		}
	}
}
