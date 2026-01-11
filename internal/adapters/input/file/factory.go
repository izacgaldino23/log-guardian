package file

func NewRealWatcherFactory(newWatcher newFSNotifyWatcher) WatcherFactory {
	return func() (FileWatcher, error) {
		w, err := newWatcher()
		if err != nil {
			return nil, err
		}
		return &WatcherWrapper{Watcher: w}, nil
	}
}
