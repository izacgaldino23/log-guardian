package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"log-guardian/internal/core/ports"

	"go.uber.org/mock/gomock"
)

func TestNewOrchestrator(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	if orc == nil {
		t.Fatal("Expected orchestrator to be created")
	}

	// Since we can't access the config directly from the orchestrator,
	// we'll test the behavior through the execution
}

func TestOrchestrator_Execute_WithStdinOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock the stdin Read method
	stdin.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(50 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestOrchestrator_Execute_WithFileOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: true},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock the file Read method
	file.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(50 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestOrchestrator_Execute_WithUnixOnly(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: true},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock the unix Read method
	unix.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			// Simulate some work
			time.Sleep(100 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(50 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestOrchestrator_Execute_WithAllIngests(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: true},
			Unix:  domain.UnixConfig{Enabled: true},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock all Read methods
	stdin.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			time.Sleep(50 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	file.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			time.Sleep(50 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	unix.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			time.Sleep(50 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()

	// Give some time for shutdown to complete
	time.Sleep(200 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}

func TestOrchestrator_Execute_WithError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock the stdin Read method to send an error
	stdin.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			errChan <- errors.New("test error")
			time.Sleep(50 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(100 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}

	if errors[0].Error() != "test error" {
		t.Errorf("Expected 'test error', got '%s'", errors[0].Error())
	}
}

func TestOrchestrator_Execute_WithLogEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Mock the stdin Read method to send log events
	stdin.EXPECT().Read(gomock.Any(), gomock.Any(), gomock.Any(), orc).DoAndReturn(
		func(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error, shutdown ports.IngestionShutdown) {
			// Send some test log events
			logEvent := domain.LogEvent{
				ID:        "test-id",
				Timestamp: time.Now(),
				Source:    "stdin",
				Severity:  domain.LOG_LEVEL_INFO,
				Message:   "test message",
				Metadata:  make(map[string]interface{}),
			}
			output <- logEvent

			time.Sleep(50 * time.Millisecond)
			shutdown.OnShutdown()
		},
	)

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(100 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 1 {
		t.Errorf("Expected 1 output, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}

	if len(outputs) > 0 {
		if outputs[0].ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", outputs[0].ID)
		}
		if outputs[0].Source != "stdin" {
			t.Errorf("Expected source 'stdin', got '%s'", outputs[0].Source)
		}
		if outputs[0].Message != "test message" {
			t.Errorf("Expected message 'test message', got '%s'", outputs[0].Message)
		}
	}
}

func TestOrchestrator_OnShutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Test that calling OnShutdown panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("OnShutdown should panic")
		}
	}()
	orc.OnShutdown()
}

func TestOrchestrator_Shutdown(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Test that shutdown doesn't panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Shutdown should not panic")
		}
	}()

	orc.Shutdown()
}

func TestOrchestrator_GetOutput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Initially, outputs should be empty
	outputs := orc.GetOutput()
	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs initially, got %d", len(outputs))
	}
}

func TestOrchestrator_GetErrors(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: false},
			File:  domain.FileConfig{Enabled: false},
			Unix:  domain.UnixConfig{Enabled: false},
		},
	}

	stdin := ports.NewMockInputProvider(ctrl)
	file := ports.NewMockInputProvider(ctrl)
	unix := ports.NewMockInputProvider(ctrl)

	orc := application.NewOrchestrator(ctx, config, stdin, file, unix)

	// Initially, errors should be empty
	errors := orc.GetErrors()
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors initially, got %d", len(errors))
	}
}

func TestOrchestrator_Execute_WithNilProviders(t *testing.T) {
	ctx := context.Background()
	config := &domain.RuntimeConfig{
		ShutdownTimeout: 5,
		Ingests: domain.Ingests{
			Stdin: domain.StdinConfig{Enabled: true},
			File:  domain.FileConfig{Enabled: true},
			Unix:  domain.UnixConfig{Enabled: true},
		},
	}

	// Create orchestrator with nil providers
	orc := application.NewOrchestrator(ctx, config, nil, nil, nil)

	// Execute should not panic even with nil providers
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute should not panic with nil providers")
		}
	}()

	// Execute in a goroutine and cancel after a short time
	go orc.Execute()
	time.Sleep(50 * time.Millisecond)
	orc.Shutdown()

	// Give some time for shutdown to complete
	time.Sleep(100 * time.Millisecond)

	outputs := orc.GetOutput()
	errors := orc.GetErrors()

	if len(outputs) != 0 {
		t.Errorf("Expected 0 outputs, got %d", len(outputs))
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}
}
