package application

import (
	"context"
	"fmt"
	"log-guardian/internal/core/domain"
	"log-guardian/internal/core/ports"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type orchestrator struct {
	ingests struct {
		stdin ports.InputProvider
		file  ports.InputProvider
		unix  ports.InputProvider
	}
	config    *domain.RuntimeConfig
	wg        *sync.WaitGroup
	once      sync.Once
	ctx       context.Context
	ctxCancel context.CancelFunc
	signal    chan os.Signal
	outputs   []domain.LogEvent
	errors    []error
}

func NewOrchestrator(
	ctx context.Context,
	config *domain.RuntimeConfig,
	stdin ports.InputProvider,
	file ports.InputProvider,
	unix ports.InputProvider,
) *orchestrator {
	ctxWithCancel, cancel := context.WithCancel(ctx)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	orc := &orchestrator{
		config:    config,
		wg:        &sync.WaitGroup{},
		ctx:       ctxWithCancel,
		ctxCancel: cancel,
		signal:    signalChan,
	}

	orc.ingests.stdin = stdin
	orc.ingests.file = file
	orc.ingests.unix = unix

	return orc
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
	if o.ingests.stdin != nil {
		o.wg.Add(1)
		o.ingests.stdin.Read(o.ctx, output, errChan, o)
	}
}

func (o *orchestrator) watchFiles(output chan<- domain.LogEvent, errChan chan<- error) {
	if o.ingests.file != nil {
		o.wg.Add(1)
		o.ingests.file.Read(o.ctx, output, errChan, o)
	}
}

func (o *orchestrator) watchUnix(output chan<- domain.LogEvent, errChan chan<- error) {
	if o.ingests.unix != nil {
		o.wg.Add(1)
		o.ingests.unix.Read(o.ctx, output, errChan, o)
	}
}

func (o *orchestrator) Shutdown() {
	o.once.Do(func() {
		o.ctxCancel()
		o.wg.Wait()
	})
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
