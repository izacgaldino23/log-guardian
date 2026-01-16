package application

import (
	"context"
	"fmt"
	"log"
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
			fmt.Printf("Log received: [%s] %s\n", event.Severity, event.Message)
		case <-o.ctx.Done():
			break outer
		case err := <-errChan:
			log.Printf("Error: %s\n", err.Error())
		case <-o.signal:
			break outer
		}
	}

	fmt.Println("Log Guardian is shutting down")

	o.Shutdown()
}

func (o *orchestrator) watchStdin(output chan<- domain.LogEvent, errChan chan<- error) {
	stdinIngest := stdin.NewStdinIngestion(os.Stdin, o.idGen)

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
		connection, err := unix.NewUnixConnectionProvider(o.config.Ingests.Unix.Sockets[0].Address, time.Duration(o.config.Ingests.Unix.Sockets[0].Timeout))
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
