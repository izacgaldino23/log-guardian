package main

import (
	"context"
	"log"
	"log-guardian/internal/adapters/infra"
	"log-guardian/internal/adapters/input/file"
	"log-guardian/internal/adapters/input/stdin"
	"log-guardian/internal/adapters/input/unix"
	"log-guardian/internal/core/application"
	"log-guardian/internal/core/domain"
	"os"
	"time"
)

var config *domain.RuntimeConfig

func init() {
	var err error

	// load config
	config, err = domain.LoadConfigs()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stdinIngest, unixIngest, fileIngest := createIngestions()

	orchestrator := application.NewOrchestrator(ctx, config, stdinIngest, fileIngest, unixIngest)
	orchestrator.Execute()
}

func createIngestions() (*stdin.StdinIngestion, *unix.UnixIngestion, *file.LogFileIngestion) {
	idGen := infra.NewUUIDGenerator()

	// stdin
	stdinIngest := stdin.NewStdinIngestion(os.Stdin, idGen)

	// unix
	socket := config.Ingests.Unix.Sockets[0]
	duration := time.Duration(socket.Timeout) * time.Millisecond

	connectionProvider := unix.NewUnixConnectionProvider()

	unixIngest := unix.NewUnixIngestion(connectionProvider, idGen, socket.Address, duration)

	// file
	watcherProvider := file.WatcherProvider{}
	fileOpener := file.OSFileSystem{}

	fileWatcher, err := watcherProvider.Create()
	if err != nil {
		log.Fatal(err)
	}

	fileIngest := file.NewLogFileIngestion(config.Ingests.File.Folders[0].FolderPath, fileWatcher, fileOpener, idGen)

	return stdinIngest, unixIngest, fileIngest
}
