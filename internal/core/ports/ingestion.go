package ports

import (
	"context"
	"log-guardian/internal/core/domain"
)

type InputProvider interface {
	Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error)
}
