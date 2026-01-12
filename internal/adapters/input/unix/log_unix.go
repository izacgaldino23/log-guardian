package unix

import (
	"bufio"
	"context"
	"io"
	"log-guardian/internal/core/domain"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	initialBufSize = 64 * 1024
	readDeadline   = 2 * time.Second
)

type UnixIngestion struct {
	socketPath   string
	listenConfig net.ListenConfig
}

func NewUnixIngestion(socketPath string, lc net.ListenConfig) *UnixIngestion {
	return &UnixIngestion{
		socketPath:   socketPath,
		listenConfig: lc,
	}
}

func (u *UnixIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	if err := os.RemoveAll(u.socketPath); err != nil {
		u.SendError(ctx, err, errChan)
		return
	}

	listener, err := u.listenConfig.Listen(ctx, "unix", u.socketPath)
	if err != nil {
		u.SendError(ctx, err, errChan)
		return
	}

	defer os.Remove(u.socketPath)
	defer listener.Close()

	var wg sync.WaitGroup
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				wg.Wait()
				return
			default:
				u.SendError(ctx, err, errChan)
				continue
			}
		}

		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			u.Run(ctx, conn, output, errChan)
		}(conn)

	}
}

func (u *UnixIngestion) Run(ctx context.Context, conn net.Conn, output chan<- domain.LogEvent, errChan chan<- error) {
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
		return
	case output <- event:
	}
}
