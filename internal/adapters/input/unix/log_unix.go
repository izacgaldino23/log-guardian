package unix

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log-guardian/internal/core/domain"
	"log-guardian/internal/core/ports"
	"net"
	"strings"
	"time"
)

const (
	initialBufSize = 4096
	readDeadline   = 5 * time.Second
	maxTries       = 3
)

type UnixIngestion struct {
	socketPath     string
	connection     Conn
	timeout        time.Duration
	idGen          domain.IDGenerator
	maxMessageSize int
}

func NewUnixIngestion(conn Conn, idGen domain.IDGenerator) *UnixIngestion {
	return &UnixIngestion{
		connection:     conn,
		idGen:          idGen,
		maxMessageSize: 1024 * 1024,
	}
}

func (u *UnixIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdownCallback ports.IngestionShutdown) {
	go func() {
		<-ctx.Done()
	}()

	go func() {
		defer shutdownCallback.OnShutdown()

		u.Run(ctx, output, errChan)
	}()
}

func (u *UnixIngestion) Run(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	defer u.connection.Close()

	reader := bufio.NewReaderSize(u.connection, initialBufSize)

	for {
		err := u.connection.SetReadDeadline(time.Now().Add(readDeadline))
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

		if len(line) > u.maxMessageSize {
			u.SendError(ctx, fmt.Errorf("message too large: %d bytes", len(line)), errChan)
			return
		}

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
	event, _ := domain.NewLogEvent(domain.SOURCE_UNIX, msg, domain.LOG_LEVEL_INFO, nil, u.idGen)

	select {
	case <-ctx.Done():
	case output <- *event:
	}
}
