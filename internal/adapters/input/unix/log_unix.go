package unix

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log-guardian/internal/core/domain"
	"net"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	initialBufSize = 4096
	readDeadline   = 5 * time.Second
	maxTries       = 3
	maxMessageSize = 1024 * 1024
)

type UnixIngestion struct {
	socketPath        string
	connectionFactory ConnectionFactory
	timeout           time.Duration
	rateLimiter *rate.Limiter
}

func NewUnixIngestion(socketPath string, connFactory ConnectionFactory, timeout time.Duration) *UnixIngestion {
	return &UnixIngestion{
		socketPath:        socketPath,
		timeout:           timeout,
		connectionFactory: connFactory,
		rateLimiter: rate.NewLimiter(rate.Limit(1000), 100),
	}
}

func (u *UnixIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	conn, err := u.connectionFactory("unix", u.socketPath, u.timeout)
	if err != nil {
		u.SendError(ctx, err, errChan)
		return
	}

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	go u.Run(ctx, conn, output, errChan)
}

func (u *UnixIngestion) Run(ctx context.Context, conn Conn, output chan<- domain.LogEvent, errChan chan<- error) {
	defer conn.Close()

	reader := bufio.NewReaderSize(conn, initialBufSize)

	for {
		err := conn.SetReadDeadline(time.Now().Add(readDeadline))
		if err != nil {
			u.SendError(ctx, err, errChan)
			return
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			if err != io.EOF {
				u.SendError(ctx, err, errChan)
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if len(line) > maxMessageSize {
			u.SendError(ctx, fmt.Errorf("message too large: %d bytes", len(line)), errChan)
			return
		}

		u.rateLimiter.Wait(ctx)

		u.Emit(ctx, line, output)
	}
}

func (u *UnixIngestion) SendError(ctx context.Context, err error, errChan chan<- error) {
	select {
	case <-ctx.Done():
	case errChan <- err:
	}
}

func (u *UnixIngestion) Emit(ctx context.Context, msg string, output chan<- domain.LogEvent) {
	event := domain.LogEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    domain.SOURCE_UNIX,
		Severity:  "INFO",
		Message:   msg,
	}

	select {
	case <-ctx.Done():
	case output <- event:
	}
}
