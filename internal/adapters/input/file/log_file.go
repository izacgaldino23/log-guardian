package file

import (
	"bufio"
	"context"
	"io"
	"log-guardian/internal/core/domain"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type LogFileIngestion struct {
	filePath       string
	watcherFactory WatcherFactory
	fileOpener     fileOpener
	watcher        FileWatcher
	file           FileHandle
}

func NewLogFileIngestion(filePath string, factory WatcherFactory, opener fileOpener) *LogFileIngestion {
	return &LogFileIngestion{
		filePath:       filePath,
		watcherFactory: factory,
		fileOpener:     opener,
	}
}

// Read reads the file writes and sends the logs to the output channel
func (lf *LogFileIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	err := lf.setup()
	if err != nil {
		errChan <- err
		return
	}

	go lf.run(ctx, output, errChan)
}

func (lf *LogFileIngestion) setup() error {
	var err error

	lf.watcher, err = lf.watcherFactory()
	if err != nil {
		return err
	}

	lf.file, err = lf.fileOpener(lf.filePath)
	if err != nil {
		lf.watcher.Close()

		return err
	}

	_, err = lf.file.Seek(0, io.SeekEnd)
	if err != nil {
		lf.file.Close()
		lf.watcher.Close()

		return err
	}

	return nil
}

func (lf *LogFileIngestion) run(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	defer lf.watcher.Close()
	defer lf.file.Close()

	// Add the file to the watcher
	err := lf.watcher.Add(lf.filePath)
	if err != nil {
		errChan <- err
		return
	}

	reader := bufio.NewReader(lf.file)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-lf.watcher.Events():
			if !ok {
				return
			}

			// Check if the event is a write
			if event.Has(fsnotify.Write) {
				lf.handleWrite(reader, output, errChan)
			}
		case err, ok := <-lf.watcher.Errors():
			if !ok {
				return
			}
			errChan <- err
			return
		}
	}
}

func (lf *LogFileIngestion) handleWrite(reader *bufio.Reader, output chan<- domain.LogEvent, errChan chan<- error) {
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		if err != nil {
			if err == io.EOF {
				if line != "" {
					lf.emit(line, output)
				}
				break
			}
			errChan <- err
			return
		}

		if line == "" {
			continue
		}

		lf.emit(line, output)
	}
}

func (lf *LogFileIngestion) emit(msg string, output chan<- domain.LogEvent) {
	output <- domain.LogEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    domain.SOURCE_FILE,
		Severity:  "INFO",
		Message:   msg,
	}
}
