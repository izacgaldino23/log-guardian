//go:build integration_test

package ingestion_test

import (
	"context"
	"io"
	"log-guardian/internal/adapters/infra"
	"log-guardian/internal/adapters/input/stdin"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStdinIngestion(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		inputData      string
		expectedCount  int
		expectedMsgs   []string
		executionDelay time.Duration
	}{
		{
			name:          "Success Ingest multiple lines",
			enabled:       true,
			inputData:     "test data\nsecond line\n",
			expectedCount: 2,
			expectedMsgs:  []string{"test data", "second line"},
		},
		{
			name:          "Disabled Should not ingest any data",
			enabled:       false,
			inputData:     "Ignored data\n",
			expectedCount: 0,
			expectedMsgs:  []string{},
		},
		{
			name:          "Success Empty input",
			enabled:       true,
			inputData:     "",
			expectedCount: 0,
			expectedMsgs:  []string{},
		},
		{
			name:          "Success Long line without newline at end",
			enabled:       true,
			inputData:     "Unique line without newline at end",
			expectedCount: 1,
			expectedMsgs:  []string{"Unique line without newline at end"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &domain.RuntimeConfig{
				Ingests: domain.Ingests{
					Stdin: domain.StdinConfig{
						Enabled: tt.enabled,
					},
				},
			}

			pr, pw := io.Pipe()
			ctx, cancel := context.WithCancel(context.Background())

			idGen := infra.NewUUIDGenerator()

			// stdin
			stdinIngest := stdin.NewStdinIngestion(pr, idGen)

			orc := application.NewOrchestrator(ctx, config, stdinIngest, nil, nil)

			go func() {
				orc.Execute()
			}()

			if tt.inputData != "" {
				go func() {
					pw.Write([]byte(tt.inputData))
					pw.Close()
				}()
			} else {
				pw.Close()
			}

			time.Sleep(100 * time.Millisecond)
			cancel()
			pr.Close()

			outputs := orc.GetOutput()
			assert.Len(t, outputs, tt.expectedCount)

			for i, msg := range tt.expectedMsgs {
				if i < len(outputs) {
					assert.Equal(t, msg, outputs[i].Message)
				}
			}
		})
	}
}
