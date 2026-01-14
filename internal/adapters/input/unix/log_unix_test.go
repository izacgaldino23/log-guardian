package unix_test

import (
	"bytes"
	"context"
	"errors"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/domain"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

type UnixClient struct {
	socketPath string
	conn       net.Conn
}

const validSocketPath = "/tmp/valid.sock"

type testCase struct {
	name                  string
	socketPath            string
	network               string
	expectedError         string
	input                 string
	shouldCancelContext   bool
	useNetPipe            bool
	mockConnectionFactory func(server net.Conn) unix.ConnectionFactory
	expectedOutput        []domain.LogEvent
}

func TestUnixIngestion_Read(t *testing.T) {
	ctrl := gomock.NewController(t)
	t.Parallel()

	testCases := []testCase{
		{
			name:       "ShouldFailInvalidSocketPath",
			socketPath: ".",
			mockConnectionFactory: func(_ net.Conn) unix.ConnectionFactory {
				return func(network, path string, timeout time.Duration) (unix.Conn, error) {
					return net.DialTimeout(network, path, timeout)
				}
			},
			expectedError: "dial unix .: connect: An invalid argument was supplied.",
		},
		{
			name:       "ShouldFailWhenGetListenerFromFactory",
			socketPath: "/tmp/asdf.sock",
			mockConnectionFactory: func(_ net.Conn) unix.ConnectionFactory {
				return func(network, path string, timeout time.Duration) (unix.Conn, error) {
					return nil, errors.New("some-listener-creation-error")
				}
			},
			expectedError: "some-listener-creation-error",
		},
		{
			name:       "ShouldFailWhenSetReadDeadline",
			socketPath: "/tmp/asdf.sock",
			mockConnectionFactory: func(_ net.Conn) unix.ConnectionFactory {
				return func(network, path string, timeout time.Duration) (unix.Conn, error) {
					mockConn := unix.NewMockConn(ctrl)
					mockConn.EXPECT().SetReadDeadline(gomock.Any()).Return(errors.New("some-set-read-dead-line-error"))
					mockConn.EXPECT().Close().AnyTimes()

					return mockConn, nil
				}
			},
			expectedError: "some-set-read-dead-line-error",
		},
		{
			name:       "ShouldFailWhenGetReadTimeoutError",
			socketPath: "/tmp/asdf.sock",
			mockConnectionFactory: func(_ net.Conn) unix.ConnectionFactory {
				return func(network, path string, timeout time.Duration) (unix.Conn, error) {
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

					return mockConn, nil
				}
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
			mockConnectionFactory: func(_ net.Conn) unix.ConnectionFactory {
				return func(network, path string, timeout time.Duration) (unix.Conn, error) {
					mockConn := unix.NewMockConn(ctrl)
					mockConn.EXPECT().SetReadDeadline(gomock.Any()).AnyTimes()
					mockConn.EXPECT().Close().AnyTimes()

					mockConn.EXPECT().Read(gomock.Any()).AnyTimes().Return(0, errors.New("some-read-error"))

					return mockConn, nil
				}
			},
			expectedError:       "some-read-error",
			shouldCancelContext: true,
		},
		{
			name:       "ShouldFailWhenGetErrorCreatingConnection",
			socketPath: validSocketPath,
			mockConnectionFactory: func(client net.Conn) unix.ConnectionFactory {
				return unix.NewNetConnectionFactory(func(_, _ string, _ time.Duration) (unix.Conn, error) {
					return nil, errors.New("some-connection-creation-error")
				})
			},
			expectedError: "some-connection-creation-error",
		},
		{
			name:       "ShouldReadEventsSuccessfully",
			socketPath: validSocketPath,
			mockConnectionFactory: func(client net.Conn) unix.ConnectionFactory {
				if client != nil {
					return unix.NewNetConnectionFactory(func(_, _ string, _ time.Duration) (unix.Conn, error) {
						return client, nil
					})
				}

				return nil
			},
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
		validateUnixReadTestCase(t, c)
	}
}

func validateUnixReadTestCase(t *testing.T, c testCase) {
	t.Run(c.name, func(t *testing.T) {
		var (
			server, client net.Conn
		)

		if c.useNetPipe {
			server, client = net.Pipe()
			defer server.Close()
			defer client.Close()
		}

		unixIngest := unix.NewUnixIngestion(c.socketPath, c.mockConnectionFactory(client), time.Second*1)

		done := make(chan struct{})
		output := make(chan domain.LogEvent, 10)
		errChan := make(chan error, 3)

		ctx, closeCtx := context.WithTimeout(context.Background(), 1*time.Second)
		defer closeCtx()

		go func() {
			unixIngest.Read(ctx, output, errChan)
			close(done)
		}()

		if c.input != "" {
			go func() {
				server.Write([]byte(c.input))
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
		for {
			if len(outputs) == len(c.expectedOutput) && len(c.expectedOutput) > 0 {
				break
			}

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
			assert.Equal(t, c.expectedError, receivedError.Error())
		}
	})
}
