package input

import (
	"bufio"
	"context"
	"io"
	"log-guardian/internal/core/domain"
	"time"
)

type StdinIngestion struct {
	reader io.Reader
}

func NewStdinIngestion(reader io.Reader) *StdinIngestion {
	return &StdinIngestion{reader: reader}
}

// Read reads the input from stdin and sends the logs to the output channel
func (i *StdinIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	scanner := bufio.NewScanner(i.reader)

	go func() {
		for scanner.Scan() {
			line := scanner.Text()

			if line == "" {
				continue
			}

			event := domain.LogEvent{
				Timestamp: time.Now().Format(time.RFC3339),
				Source:    domain.SOURCE_STDIN,
				Severity:  "INFO",
				Message:   line,
			}

			select {
			case <-ctx.Done():
				return
			case output <- event:
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()
}
