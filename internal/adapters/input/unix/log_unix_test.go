package unix_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/domain"
	"log-guardian/internal/core/ports"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

const validSocketPath = "/tmp/valid.sock"

type testCase struct {
	name                string
	socketPath          string
	expectedError       string
	input               string
	shouldCancelContext bool
	useNetPipe          bool
	mockConnection      func(address string, timeout time.Duration) unix.ConnectionProvider
	expectedOutput      []domain.LogEvent
}

func TestUnixIngestion_Read(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	idGen := domain.NewMockIDGenerator(ctrl)
	idGen.EXPECT().Generate().AnyTimes().Return("some-id", nil)

	testCases := []testCase{
		{
			name:       "ShouldFailWhenSetReadDeadline",
			socketPath: "/tmp/asdf.sock",
			mockConnection: func(address string, timeout time.Duration) unix.ConnectionProvider {
				mockConn := unix.NewMockConn(ctrl)
				mockConn.EXPECT().SetReadDeadline(gomock.Any()).Return(errors.New("some-set-read-dead-line-error"))
				mockConn.EXPECT().Close().AnyTimes()

				mockConnectionProvider := unix.NewMockConnectionProvider(ctrl)
				mockConnectionProvider.EXPECT().DialTimeout(gomock.Any(), address, timeout).Return(mockConn, nil)

				return mockConnectionProvider
			},
			expectedError: "some-set-read-dead-line-error",
		},
		{
			name:       "ShouldFailWhenGetReadTimeoutError",
			socketPath: "/tmp/asdf.sock",
			mockConnection: func(address string, timeout time.Duration) unix.ConnectionProvider {
				mockConn := unix.NewMockConn(ctrl)
				mockConn.EXPECT().SetReadDeadline(gomock.Any()).AnyTimes()
				mockConn.EXPECT().Close().AnyTimes()

				mockConn.EXPECT().Read(gomock.Any()).Return(0, context.DeadlineExceeded)

				input := "One result\n"
				r := bytes.NewBufferString(input)

				mockConn.EXPECT().Read(gomock.Any()).DoAndReturn(func(b []byte) (int, error) {
					return r.Read(b)
				})
				mockConn.EXPECT().Read(gomock.Any()).AnyTimes().Return(0, nil)

				mockConnectionProvider := unix.NewMockConnectionProvider(ctrl)
				mockConnectionProvider.EXPECT().DialTimeout(gomock.Any(), address, timeout).Return(mockConn, nil)

				return mockConnectionProvider
			},
			expectedOutput: []domain.LogEvent{
				{
					Source:   domain.SOURCE_UNIX,
					Severity: "INFO",
					Message:  "One result",
				},
			},
		},
		{
			name:       "ShouldFailWhenGetNonReadTimeoutError",
			socketPath: "/tmp/asdf.sock",
			mockConnection: func(address string, timeout time.Duration) unix.ConnectionProvider {
				mockConn := unix.NewMockConn(ctrl)
				mockConn.EXPECT().SetReadDeadline(gomock.Any()).AnyTimes()
				mockConn.EXPECT().Close().AnyTimes()

				mockConn.EXPECT().Read(gomock.Any()).AnyTimes().Return(0, errors.New("some-read-error"))

				mockConnectionProvider := unix.NewMockConnectionProvider(ctrl)
				mockConnectionProvider.EXPECT().DialTimeout(gomock.Any(), address, timeout).Return(mockConn, nil)

				return mockConnectionProvider
			},
			expectedError:       "some-read-error",
			shouldCancelContext: true,
		},
		{
			name:          "ShouldFailBecauseTheLineIsTooLong",
			socketPath:    validSocketPath,
			input:         func() string { return strings.Repeat("a", 1024*1024+1) + "\n" }(),
			useNetPipe:    true,
			expectedError: fmt.Sprintf("message too large: %v bytes", 1024*1024+1),
		},
		{
			name:       "ShouldReadEventsSuccessfully",
			socketPath: validSocketPath,
			input:      "first line\nsecond line\n\n",
			useNetPipe: true,
			expectedOutput: []domain.LogEvent{
				{
					Source:   domain.SOURCE_UNIX,
					Severity: "INFO",
					Message:  "first line",
				},
				{
					Source:   domain.SOURCE_UNIX,
					Severity: "INFO",
					Message:  "second line",
				},
			},
		},
	}

	for _, c := range testCases {
		validateUnixReadTestCase(t, c, idGen)
	}
}

func validateUnixReadTestCase(t *testing.T, c testCase, idGen domain.IDGenerator) {
	t.Run(c.name, func(t *testing.T) {
		var (
			server, client     net.Conn
			connectionProvider = c.mockConnection
		)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		localConnection := func(address string, timeout time.Duration) unix.ConnectionProvider {
			mockConnectionProvider := unix.NewMockConnectionProvider(ctrl)
			mockConnectionProvider.EXPECT().DialTimeout(gomock.Any(), address, timeout).Return(client, nil)

			return mockConnectionProvider
		}

		if c.useNetPipe {
			server, client = net.Pipe()
			defer server.Close()
			defer client.Close()

			connectionProvider = localConnection
		}

		connProvider := connectionProvider(c.socketPath, time.Second*1)

		unixIngest := unix.NewUnixIngestion(connProvider, idGen, c.socketPath, time.Second*1)

		done := make(chan struct{})
		output := make(chan domain.LogEvent, 10)
		errChan := make(chan error, 3)

		ctx, closeCtx := context.WithTimeout(context.Background(), 1*time.Second)
		defer closeCtx()

		shutdownMock := ports.NewMockIngestionShutdown(ctrl)
		shutdownMock.EXPECT().OnShutdown().AnyTimes()

		go func() {
			unixIngest.Read(ctx, output, errChan, shutdownMock)
			close(done)
		}()

		if c.input != "" {
			go func() {
				_, err := server.Write([]byte(c.input))
				if err != nil {
					t.Error(err)
				}
			}()
		}

		if c.shouldCancelContext {
			go func() {
				time.Sleep(10 * time.Millisecond)
				closeCtx()
			}()
		}

		outputs := []domain.LogEvent{}
		deadline := time.After(1 * time.Second)

		var receivedError error

	outer:
		for len(outputs) < len(c.expectedOutput) || len(c.expectedOutput) == 0 {
			select {
			case result := <-output:
				outputs = append(outputs, result)
			case receivedError = <-errChan:
				closeCtx()
				break outer
			case <-deadline:
				t.Fatal("The result didn't arrive to the channels")
			}
		}

		closeCtx()
		<-done

		if count := len(c.expectedOutput); count > 0 {
			assert.Equal(t, count, len(outputs))

			for i := range c.expectedOutput {
				var found bool
				for j := range outputs {
					if outputs[j].Message == c.expectedOutput[i].Message {
						found = true
						break
					}
				}

				assert.True(t, found)
			}
		}

		if c.expectedError != "" {
			assert.NotNil(t, receivedError)
			assert.Contains(t, receivedError.Error(), c.expectedError)
		}
	})
}
