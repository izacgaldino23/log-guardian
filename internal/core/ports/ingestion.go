package ports

import (
	"context"
	"log-guardian/internal/core/domain"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type IngestionShutdown interface {
	OnShutdown()
}

type InputProvider interface {
	Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown IngestionShutdown)
}
