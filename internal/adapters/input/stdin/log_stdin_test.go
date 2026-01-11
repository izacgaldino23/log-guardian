package stdin_test

import (
	"bytes"
	"context"
	"errors"
	"log-guardian/internal/adapters/input/stdin"
	"log-guardian/internal/core/domain"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("forced read error")
}

func TestStdinIngestion(t *testing.T) {
	t.Run("SuccessRead", func(t *testing.T) {
		fakeInput := []string{"", "log1", "log2", "log3"}

		reader := bytes.NewReader([]byte(strings.Join(fakeInput, "\n")))
		stdin := stdin.NewStdinIngestion(reader)

		ctx := t.Context()

		output := make(chan domain.LogEvent)
		errChan := make(chan error)
		stdin.Read(ctx, output, errChan)

		outputCount := 0

		for i := 0; i < len(fakeInput)-1; i++ {
			select {
			case <-output:
				outputCount++
			case <-time.After(1 * time.Second):
				t.Fatal("The log didn't arrive to the channel")
			}
		}

		assert.Equal(t, len(fakeInput)-1, outputCount)
	})

	t.Run("ContextCanceled", func(t *testing.T) {
		fakeInput := []string{"log1", "log2", "log3"}

		reader := bytes.NewReader([]byte(strings.Join(fakeInput, "\n")))
		stdin := stdin.NewStdinIngestion(reader)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		output := make(chan domain.LogEvent)

		errChan := make(chan error)
		stdin.Read(ctx, output, errChan)

		<-output
		cancel()

		time.Sleep(50 * time.Millisecond)

		countRestante := 0
		for {
			select {
			case <-output:
				countRestante++
				if countRestante > 1 {
					t.Fatal("The goroutine is still running")
				}
			case <-time.After(100 * time.Millisecond):
				return
			}
		}
	})

	t.Run("ScannerError", func(t *testing.T) {
		reader := &errorReader{}
		stdin := stdin.NewStdinIngestion(reader)

		ctx := t.Context()

		output := make(chan domain.LogEvent)

		errChan := make(chan error)
		stdin.Read(ctx, output, errChan)

		err := <-errChan
		assert.Error(t, err)
	})
}
