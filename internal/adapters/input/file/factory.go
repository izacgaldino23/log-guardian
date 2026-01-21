package file

import (
	"os"

	"github.com/fsnotify/fsnotify"
)

type WatcherProvider struct{}

func (p *WatcherProvider) Create() (FileWatcher, error) {
	w, err := fsnotify.NewWatcher()
	return &WatcherWrapper{Watcher: w}, err
}

type OSFileSystem struct{}

func (OSFileSystem) Open(name string) (FileHandle, error) {
	return os.Open(name)
}
