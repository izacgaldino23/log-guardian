package application

import (
	"context"
	"fmt"
	"io"
	"log-guardian/internal/adapters/infra"
	"log-guardian/internal/adapters/input/file"
	"log-guardian/internal/adapters/input/stdin"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/domain"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type orchestrator struct {
	config          *domain.RuntimeConfig
	idGen           domain.IDGenerator
	wg              *sync.WaitGroup
	shutdownTimeout time.Duration
	ctx             context.Context
	ctxCancel       context.CancelFunc
	signal          chan os.Signal
	Input           io.Reader
	outputs         []domain.LogEvent
	errors          []error
}

func NewOrchestrator(ctx context.Context, config *domain.RuntimeConfig, shutdownTimeout time.Duration) *orchestrator {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	return &orchestrator{
		config:          config,
		idGen:           infra.NewUUIDGenerator(),
		wg:              &sync.WaitGroup{},
		shutdownTimeout: shutdownTimeout,
		ctx:             ctxWithCancel,
		ctxCancel:       cancel,
		signal:          signalChan,
	}
}

func (o *orchestrator) Execute() {
	outputChan := make(chan domain.LogEvent, 100)
	errChan := make(chan error, 10)

	if o.config.Ingests.Stdin.Enabled {
		o.watchStdin(outputChan, errChan)
	}

	if o.config.Ingests.File.Enabled {
		o.watchFiles(outputChan, errChan)
	}

	if o.config.Ingests.Unix.Enabled {
		o.watchUnix(outputChan, errChan)
	}

	fmt.Println("Log Guardian is running")

outer:
	for {
		select {
		case event := <-outputChan:
			o.outputs = append(o.outputs, event)
		case <-o.ctx.Done():
			break outer
		case err := <-errChan:
			o.errors = append(o.errors, err)
		case <-o.signal:
			break outer
		}
	}

	fmt.Println("Log Guardian is shutting down")

	o.Shutdown()
}

func (o *orchestrator) watchStdin(output chan<- domain.LogEvent, errChan chan<- error) {
	input := o.Input
	if input == nil {
		input = os.Stdin
	}

	stdinIngest := stdin.NewStdinIngestion(input, o.idGen)

	o.wg.Add(1)
	stdinIngest.Read(o.ctx, output, errChan, o)
}

func (o *orchestrator) watchFiles(output chan<- domain.LogEvent, errChan chan<- error) {
	var fileIngest *file.LogFileIngestion
	if len(o.config.Ingests.File.Folders) > 0 {
		// TODO: support for folders
		// TODO: support for ignore files
		// running only on the first index

		watcherProvider := file.WatcherProvider{}
		fileOpener := file.OSFileSystem{}

		fileWatcher, err := watcherProvider.Create()
		if err != nil {
			errChan <- err
			return
		}

		fileIngest = file.NewLogFileIngestion(o.config.Ingests.File.Folders[0].FolderPath, fileWatcher, fileOpener, o.idGen)

		o.wg.Add(1)
		fileIngest.Read(o.ctx, output, errChan, o)
	}
}

func (o *orchestrator) watchUnix(output chan<- domain.LogEvent, errChan chan<- error) {
	var unixIngest *unix.UnixIngestion
	if len(o.config.Ingests.Unix.Sockets) > 0 {
		// TODO: support for many connections
		// running only on the first index
		socket := o.config.Ingests.Unix.Sockets[0]
		duration := time.Duration(socket.Timeout) * time.Millisecond
		connection, err := unix.NewUnixConnectionProvider(socket.Address, duration)
		if err != nil {
			errChan <- err
			return
		}

		unixIngest = unix.NewUnixIngestion(connection, o.idGen)

		o.wg.Add(1)
		unixIngest.Read(o.ctx, output, errChan, o)
	}
}

func (o *orchestrator) Shutdown() {
	o.ctxCancel()
	o.wg.Wait()
}

func (s *orchestrator) OnShutdown() {
	s.wg.Done()
}

func (o *orchestrator) GetOutput() []domain.LogEvent {
	return o.outputs
}

func (o *orchestrator) GetErrors() []error {
	return o.errors
}
