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
}

func NewLogFileIngestion(filePath string, factory WatcherFactory, opener fileOpener) *LogFileIngestion {
	return &LogFileIngestion{
		filePath:       filePath,
		watcherFactory: factory,
		fileOpener:     opener,
	}
}

// Read reads the file writes and sends the logs to the output channel
func (i *LogFileIngestion) Read(ctx context.Context, output chan<- domain.LogEvent, errChan chan<- error) {
	// Create watcher
	watcher, err := i.watcherFactory()
	if err != nil {
		errChan <- err
		return
	}

	// Open the file
	file, err := i.fileOpener(i.filePath)
	if err != nil {
		watcher.Close()

		errChan <- err
		return
	}

	// Seek to the end of the file
	_, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		file.Close()
		watcher.Close()

		errChan <- err
		return
	}

	go func() {
		defer watcher.Close()
		defer file.Close()

		// Add the file to the watcher
		err = watcher.Add(i.filePath)
		if err != nil {
			errChan <- err
			return
		}

		reader := bufio.NewReader(file)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events():
				if !ok {
					return
				}

				// Check if the event is a write
				if event.Has(fsnotify.Write) {
					for {
						line, err := reader.ReadString('\n')
						line = strings.TrimSuffix(line, "\n")
						if err != nil {
							if err == io.EOF {
								if line != "" {
									i.writeOnOutput(line, output)
								}
								break
							}
							errChan <- err
							return
						}

						if line == "" {
							continue
						}

						i.writeOnOutput(line, output)
					}
				}
			case err, ok := <-watcher.Errors():
				if !ok {
					return
				}
				errChan <- err
			}
		}
	}()
}

func (i *LogFileIngestion) writeOnOutput(msg string, output chan<- domain.LogEvent) {
	output <- domain.LogEvent{
		Timestamp: time.Now().Format(time.RFC3339),
		Source:    domain.SOURCE_FILE,
		Severity:  "INFO",
		Message:   msg,
	}
}
