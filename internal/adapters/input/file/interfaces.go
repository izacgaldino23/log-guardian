package file

import (
	"io"

	"github.com/fsnotify/fsnotify"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=$GOPACKAGE

type (
	FileHandle interface {
		io.Reader
		io.Closer
		io.Seeker
	}

	FileSystem interface {
		Open(name string) (FileHandle, error)
	}
)

type (
	// FileWatcher define o comportamento necessário para o adaptador
	FileWatcher interface {
		Add(name string) error
		Close() error
		Events() <-chan fsnotify.Event
		Errors() <-chan error
	}

	// WatcherWrapper adapta a struct concreta do fsnotify para a nossa interface
	WatcherWrapper struct {
		*fsnotify.Watcher
	}
)

func (w *WatcherWrapper) Events() <-chan fsnotify.Event { return w.Watcher.Events }
func (w *WatcherWrapper) Errors() <-chan error          { return w.Watcher.Errors }
