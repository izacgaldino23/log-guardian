package file

import (
	"bufio"
	"context"
	"io"
	"log-guardian/internal/core/domain"
	"strings"

	"github.com/fsnotify/fsnotify"
)

type LogFileIngestion struct {
	filePath       string
	fileSystem     FileSystem
	fileWatcher        FileWatcher
	file           FileHandle
	idGen          domain.IDGenerator
}

func NewLogFileIngestion(filePath string, fileWatcher FileWatcher, opener FileSystem, idGen domain.IDGenerator) *LogFileIngestion {
	return &LogFileIngestion{
		filePath:       filePath,
		fileWatcher: fileWatcher,
		fileSystem:     opener,
		idGen:          idGen,
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

	lf.file, err = lf.fileSystem.Open(lf.filePath)
	if err != nil {
		lf.fileWatcher.Close()

		return err
	}

	_, err = lf.file.Seek(0, io.SeekEnd)
	if err != nil {
		lf.file.Close()
		lf.fileWatcher.Close()

		return err
	}

	return nil
}

func (lf *LogFileIngestion) run(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	defer lf.fileWatcher.Close()
	defer lf.file.Close()

	// Add the file to the watcher
	err := lf.fileWatcher.Add(lf.filePath)
	if err != nil {
		errChan <- err
		return
	}

	reader := bufio.NewReader(lf.file)

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-lf.fileWatcher.Events():
			if !ok {
				return
			}

			// Check if the event is a write
			if event.Has(fsnotify.Write) {
				lf.handleWrite(reader, output, errChan)
			}
		case err := <-lf.fileWatcher.Errors():
			if err == nil {
				continue
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
	event, _ := domain.NewLogEvent(domain.SOURCE_FILE, msg, domain.LOG_LEVEL_INFO, nil, lf.idGen)
	output <- *event
}
