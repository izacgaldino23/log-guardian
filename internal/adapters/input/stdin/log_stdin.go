package stdin

import (
	"bufio"
	"context"
	"io"
	"log-guardian/internal/core/domain"
)

type StdinIngestion struct {
	reader io.Reader
	idGen  domain.IDGenerator
}

func NewStdinIngestion(reader io.Reader, idGen domain.IDGenerator) *StdinIngestion {
	return &StdinIngestion{
		reader: reader,
		idGen:  idGen,
	}
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

			event, _ := domain.NewLogEvent(domain.SOURCE_STDIN, line, domain.LOG_LEVEL_INFO, nil, i.idGen)

			select {
			case <-ctx.Done():
				return
			case output <- *event:
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()
}
