package unix_test

import (
	"context"
	"errors"
	"fmt"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/domain"
	"net"
	"strings"
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

// NewClient
func NewClient(socketPath string) (*UnixClient, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, err
	}
	return &UnixClient{
		socketPath: socketPath,
		conn:       conn,
	}, nil
}

func (c *UnixClient) Write(msg string) error {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	_, err := fmt.Fprint(c.conn, msg)
	return err
}

func (c *UnixClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

type testCase struct {
	name                string
	socketPath          string
	network             string
	expectedError       string
	input               string
	shouldCancelContext bool
	mockListenerFactory unix.ListenerFactory
	expectedOutput      []domain.LogEvent
}

func TestUnixIngestion_Read(t *testing.T) {
	ctrl := gomock.NewController(t)

	testCases := []testCase{
		{
			name:       "ShouldFailInvalidSocketPath",
			socketPath: ".",
			mockListenerFactory: func(network, path string) (unix.Listener, error) {
				return nil, nil
			},
			expectedError: "RemoveAll .: invalid argument",
		},
		{
			name:       "ShouldFailWhenGetListenerFromFactory",
			socketPath: "/tmp/asdf.sock",
			mockListenerFactory: func(network, path string) (unix.Listener, error) {
				return nil, errors.New("some-listener-creation-error")
			},
			expectedError: "some-listener-creation-error",
		},
		{
			name:       "ShouldFailWhenCallAccept",
			socketPath: "/tmp/asdf.sock",
			mockListenerFactory: func(network, path string) (unix.Listener, error) {
				mockListener := unix.NewMockListener(ctrl)
				mockListener.EXPECT().Accept().AnyTimes().Return(nil, errors.New("some-listener-accept-error"))
				mockListener.EXPECT().Close().AnyTimes()

				return mockListener, nil
			},
			expectedError: "some-listener-accept-error",
		},
		{
			name:       "ShouldFailWhenSetReadDeadline",
			socketPath: "/tmp/asdf.sock",
			mockListenerFactory: func(network, path string) (unix.Listener, error) {
				mockListener := unix.NewMockListener(ctrl)
				mockListener.EXPECT().Accept().Times(3).Return(&net.UnixConn{}, nil)
				mockListener.EXPECT().Close().AnyTimes()

				return mockListener, nil
			},
			expectedError: "invalid argument",
		},
		{
			name:       "ShouldFailWhenGetReadTimeoutError",
			socketPath: "/tmp/asdf.sock",
			mockListenerFactory: func(network, path string) (unix.Listener, error) {
				mockReader := unix.NewMockConn(ctrl)
				mockReader.EXPECT().Read(gomock.Any()).AnyTimes().Return(0, errors.New("some-read-error"))
				mockReader.EXPECT().SetReadDeadline(gomock.Any()).AnyTimes()
				mockReader.EXPECT().Close().AnyTimes()

				mockListener := unix.NewMockListener(ctrl)
				mockListener.EXPECT().Accept().Times(3).Return(mockReader, nil)
				mockListener.EXPECT().Close().AnyTimes()

				return mockListener, nil
			},
			expectedError:       "some-read-error",
			shouldCancelContext: true,
		},
		{
			name:                "ShouldReadEventsSuccessfully",
			socketPath:          validSocketPath,
			mockListenerFactory: unix.NewNetListenerFactory(),
			input:               "first line\nsecond line",
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
		unixIngest := unix.NewUnixIngestion(c.socketPath, c.mockListenerFactory)

		output := make(chan domain.LogEvent, 1)
		defer close(output)

		errChan := make(chan error, 3)
		defer close(errChan)

		ctx, closeCtx := context.WithCancel(context.Background())
		defer closeCtx()

		var (
			client *UnixClient
			err    error
		)

		shouldWrite := c.socketPath == validSocketPath && c.input != ""

		unixIngest.Read(ctx, output, errChan)
		if c.shouldCancelContext {
			closeCtx()
		}

		if c.expectedOutput != nil {
			// if expect some output, errChan can't receive an error
			// add validation for this case
			select {
			case err := <-errChan:
				closeCtx()
				assert.NoError(t, err)
			default:
			}

		}

		if c.socketPath == validSocketPath {
			time.Sleep(50 * time.Millisecond)
			client, err = NewClient(c.socketPath)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
		}

		if shouldWrite && client != nil {
			err = client.Write(c.input)
			if err != nil {
				t.Fatal(err)
			}
		}

		outputCount := 0
		outputs := make([]domain.LogEvent, 0)

		select {
		case result := <-output:
			outputCount++
			outputs = append(outputs, result)
		case err := <-errChan:
			closeCtx()
			assert.EqualError(t, err, c.expectedError)
		case <-time.After(1 * time.Second):
			t.Fatal("The result didn't arrive to the channels")
		}

		if shouldWrite {
			assert.Equal(t, len(c.expectedOutput), outputCount)

			for i := range c.expectedOutput {
				var found bool
				for j := range outputs {
					if outputs[j].Message == c.expectedOutput[i].Message {
						found = true
					}
				}

				assert.True(t, found)
			}
		}
	})
}
