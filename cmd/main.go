package main

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
	"time"
)

func init() {
	// load config
	_, err := domain.LoadConfigs()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config := domain.GetConfig()

	uuidGenerator := infra.NewUUIDGenerator()

	outputChan := make(chan domain.LogEvent, 100)
	errChan := make(chan error, 10)

	if config.StdinConfig.Enabled {
		watchStdin(ctx, config, uuidGenerator, outputChan, errChan)
	}

	if config.FileConfig.Enabled {
		watchFiles(ctx, config, uuidGenerator, outputChan, errChan)
	}

	if config.UnixConfig.Enabled {
		watchUnix(ctx, config, uuidGenerator, outputChan, errChan)
	}

	fmt.Println("Log Guardian is running")

	for {
		select {
		case event := <-outputChan:
			fmt.Printf("Log received: [%s] %s\n", event.Severity, event.Message)
		case <-ctx.Done():
			return
		case err := <-errChan:
			close(outputChan)
			log.Fatal(err)
		}
	}
}

func watchStdin(ctx context.Context, config *domain.RuntimeConfig, idGen domain.IDGenerator, output chan<- domain.LogEvent, errChan chan<- error) {
	stdinIngest := stdin.NewStdinIngestion(os.Stdin, idGen)

	stdinIngest.Read(ctx, output, errChan)
}

func watchFiles(ctx context.Context, config *domain.RuntimeConfig, idGen domain.IDGenerator, output chan<- domain.LogEvent, errChan chan<- error) {
	var fileIngest *file.LogFileIngestion
	if len(config.FileConfig.Folders) > 0 {
		// TODO: support for folders
		// TODO: support for ignore files
		// running only on the first index

		watcherProvider := file.WatcherProvider{}
		fileOpener := file.OSFileSystem{}

		fileWatcher, err := watcherProvider.Create()
		if err != nil {
			log.Fatal(err)
		}

		fileIngest = file.NewLogFileIngestion(config.FileConfig.Folders[0].FolderPath, fileWatcher, fileOpener, idGen)
	}

	fileIngest.Read(ctx, output, errChan)
}

func watchUnix(ctx context.Context, config *domain.RuntimeConfig, idGen domain.IDGenerator, output chan<- domain.LogEvent, errChan chan<- error) {
	var unixIngest *unix.UnixIngestion
	if len(config.UnixConfig.Sockets) > 0 {
		// TODO: support for many connections
		// running only on the first index
		connection, err := unix.NewUnixConnectionProvider(config.UnixConfig.Sockets[0].Address, time.Duration(config.UnixConfig.Sockets[0].Timeout))
		if err != nil {
			log.Fatal(err)
		}

		unixIngest := unix.NewUnixIngestion(connection, idGen)

		unixIngest.Read(ctx, output, errChan)
	}

	unixIngest.Read(ctx, make(chan domain.LogEvent), make(chan error))
}
