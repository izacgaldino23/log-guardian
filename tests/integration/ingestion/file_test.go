//go:build integration_test
// +build integration_test

package ingestion_test

import (
	"context"
	"fmt"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFileIngestion(t *testing.T) {
	tests := []struct {
		name          string
		enabled       bool
		inputData     string
		expectedCount int
		expectedMsgs  []string
	}{
		{
			name:          "Enabled Single file multiple lines",
			enabled:       true,
			inputData:     "line 1\nline 2\n",
			expectedCount: 2,
			expectedMsgs:  []string{"line 1", "line 2"},
		},
		{
			name:          "Disabled No data should be captured",
			enabled:       false,
			inputData:     "ignored line\n",
			expectedCount: 0,
			expectedMsgs:  []string{},
		},
		{
			name:          "Enabled Empty file",
			enabled:       true,
			inputData:     "",
			expectedCount: 0,
			expectedMsgs:  []string{},
		},
		{
			name:          "Enabled Content without trailing newline",
			enabled:       true,
			inputData:     "single line no newline",
			expectedCount: 1,
			expectedMsgs:  []string{"single line no newline"},
		},
		{
			name:          "Enabled should return error when file does not exist",
			enabled:       true,
			inputData:     "single line no newline",
			expectedCount: 1,
			expectedMsgs:  []string{"single line no newline"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile, err := os.CreateTemp(os.TempDir(), "test_file_ingestion_*.log")
			assert.Nil(t, err)
			tempFile.Close()
			defer os.Remove(tempFile.Name())

			fileName := tempFile.Name()

			config := &domain.RuntimeConfig{
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						IngestionConfig: domain.IngestionConfig{Enabled: tt.enabled},
						Folders: []domain.FolderConfig{
							{FolderPath: fileName},
						},
					},
				},
			}

			ctx, cancel := context.WithCancel(context.Background())
			orc := application.NewOrchestrator(ctx, config, 5*time.Second)

			go func() {
				orc.Execute()
			}()

			time.Sleep(100 * time.Millisecond)
			if tt.inputData != "" {
				writeFile(fileName, tt.inputData)
			}

			time.Sleep(200 * time.Millisecond)
			cancel()

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

func TestFileIngestion_Error(t *testing.T) {
	fileName := "/some/path/that/does/not/exist.log"

	config := &domain.RuntimeConfig{
		Ingests: domain.Ingests{
			File: domain.FileConfig{
				IngestionConfig: domain.IngestionConfig{Enabled: true},
				Folders: []domain.FolderConfig{
					{FolderPath: fileName},
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	orc := application.NewOrchestrator(ctx, config, 5*time.Second)

	go func() {
		orc.Execute()
	}()

	time.Sleep(200 * time.Millisecond)
	cancel()

	errorsList := orc.GetErrors()
	assert.GreaterOrEqual(t, len(errorsList), 1)
	assert.Error(t, errorsList[0], fmt.Sprintf("open %s: no such file or directory", fileName))
}

func writeFile(filename string, content string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(content)
	return err
}
