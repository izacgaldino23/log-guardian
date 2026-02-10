//go:build integration_test

package ingestion_test

import (
	"context"
	"log-guardian/internal/adapters/infra"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"net"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnixSocketIngestion(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix Socket test on Windows")
	}

	socketPath := "/tmp/log-guardian-external.sock"

	tests := []struct {
		name          string
		enabled       bool
		createSocket  bool
		inputData     string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "Enabled - Successful connection and read",
			enabled:       true,
			createSocket:  true,
			inputData:     "log message\n",
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "Enabled - Socket does not exist",
			enabled:       true,
			createSocket:  false,
			inputData:     "",
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "Disabled - No connection attempted",
			enabled:       false,
			createSocket:  true,
			inputData:     "ignored\n",
			expectedCount: 0,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = os.Remove(socketPath)

			if tt.createSocket {
				listener, err := net.Listen("unix", socketPath)
				require.NoError(t, err)
				defer listener.Close()

				if tt.enabled {
					go func() {
						conn, err := listener.Accept()
						if err == nil {
							defer conn.Close()
							conn.Write([]byte(tt.inputData))
							time.Sleep(50 * time.Millisecond)
						}
					}()
				}
			}

			config := &domain.RuntimeConfig{
				Ingests: domain.Ingests{
					Unix: domain.UnixConfig{
						Enabled: tt.enabled,
						Sockets: []domain.UnixSocket{
							{
								Address: socketPath,
								Timeout: 100,
							},
						},
					},
				},
			}

			socket := config.Ingests.Unix.Sockets[0]
			duration := time.Duration(socket.Timeout) * time.Millisecond

			connectionProvider := unix.NewUnixConnectionProvider()

			idGen := infra.NewUUIDGenerator()
			unixIngest := unix.NewUnixIngestion(connectionProvider, idGen, socket.Address, duration)

			ctx, cancel := context.WithCancel(context.Background())
			orc := application.NewOrchestrator(ctx, config, nil, nil, unixIngest)

			go orc.Execute()

			time.Sleep(250 * time.Millisecond)
			cancel()

			outputs := orc.GetOutput()
			assert.Len(t, outputs, tt.expectedCount)

			errorsList := orc.GetErrors()
			if tt.expectError {
				assert.GreaterOrEqual(t, len(errorsList), 1)
				if len(errorsList) > 0 {
					assert.Error(t, errorsList[0])
				}
			} else {
				assert.Len(t, errorsList, 0)
			}

			if tt.enabled && tt.createSocket && len(outputs) > 0 {
				assert.Equal(t, "log message", outputs[0].Message)
			}
		})
	}
}
