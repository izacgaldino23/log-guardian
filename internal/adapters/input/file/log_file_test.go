package file_test

import (
	"context"
	"errors"
	"io"
	"log-guardian/internal/adapters/input/file"
	"log-guardian/internal/core/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func TestLogFileIngestion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                string
		filePath            string
		expectError         string
		fileBodyWrite       string
		shouldCancelContext bool
		watcherFactory      func() (file.FileWatcher, error)
		fileOpener          func(name string) (file.FileHandle, error)
	}{
		{
			name: "ShouldFailBecauseWatcherInitializationFails",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				return file.NewMockFileWatcher(ctrl), errors.New("error-creating-watcher")
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return nil, nil
			},
			expectError: "error-creating-watcher",
		},
		{
			name: "ShouldFailBecauseFileOpeningFails",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mock := file.NewMockFileWatcher(ctrl)
				mock.EXPECT().Close().AnyTimes()
				return mock, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return nil, errors.New("some-open-file-error")
			},
			expectError: "some-open-file-error",
		},
		{
			name: "ShouldFailBecauseFileSeekFails",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mock := file.NewMockFileWatcher(ctrl)
				mock.EXPECT().Close().AnyTimes()
				return mock, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				ctrl := gomock.NewController(t)
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), errors.New("some-seek-error"))
				mockFile.EXPECT().Close().Return(nil)
				return mockFile, nil
			},
			expectError: "some-seek-error",
		},
		{
			name: "ShouldFailBecauseFileWatcherAddFails",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add("file_path").Return(errors.New("some-watcher-add-error"))

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				ctrl := gomock.NewController(t)
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), nil)
				mockFile.EXPECT().Close().AnyTimes()
				return mockFile, nil
			},
			filePath:    "file_path",
			expectError: "some-watcher-add-error",
		},
		{
			name: "ShouldEndBecauseContextIsCanceled",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Events().AnyTimes()
				mockWatcher.EXPECT().Errors().AnyTimes()

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
			shouldCancelContext: true,
		},
		{
			name: "ShouldSkipBecauseEventIsEmpty",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Errors().AnyTimes()
				events := make(chan fsnotify.Event, 1)
				events <- fsnotify.Event{}
				close(events)
				mockWatcher.EXPECT().Events().AnyTimes().Return(events)

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
		},
		{
			name: "ShouldSkipBecauseEventIsNotWriteKind",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Errors().AnyTimes()
				events := make(chan fsnotify.Event, 1)
				events <- fsnotify.Event{
					Op: fsnotify.Create,
				}
				close(events)
				mockWatcher.EXPECT().Events().AnyTimes().Return(events)

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
		},
		{
			name:           "ShouldBreakTheLoopBecauseGotTheEOFError",
			watcherFactory: file.NewRealWatcherFactory(fsnotify.NewWatcher),
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
			fileBodyWrite: "hello\nworld",
		},
		{
			name: "ShouldFailBecauseReadStringFails",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Errors().AnyTimes()
				events := make(chan fsnotify.Event, 1)
				events <- fsnotify.Event{
					Op: fsnotify.Write,
				}
				close(events)
				mockWatcher.EXPECT().Events().AnyTimes().Return(events)

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				ctrl := gomock.NewController(t)
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), nil)
				mockFile.EXPECT().Read(gomock.Any()).Return(0, errors.New("some-read-error"))
				mockFile.EXPECT().Close().AnyTimes()

				return mockFile, nil
			},
			expectError: "some-read-error",
		},
		{
			name:           "ShouldContinueTheLoopIfTheLineStringIsEmpty",
			watcherFactory: file.NewRealWatcherFactory(fsnotify.NewWatcher),
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
			fileBodyWrite: "hello\nworld\n\n",
		},
		{
			name:           "ShouldOutputTheEventLogWithSuccess",
			watcherFactory: file.NewRealWatcherFactory(fsnotify.NewWatcher),
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
			fileBodyWrite: "hello\nworld\nfinal",
		},
		// should fail because watcher.Errors channel returns error
		{
			name: "ShouldFailBecauseWatcherErrorsChannelReturnsError",
			watcherFactory: func() (file.FileWatcher, error) {
				ctrl := gomock.NewController(t)
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Events().AnyTimes()
				errorChan := make(chan error, 1)
				errorChan <- errors.New("some-watch-error")
				close(errorChan)

				mockWatcher.EXPECT().Errors().AnyTimes().Return(errorChan)

				return mockWatcher, nil
			},
			fileOpener: func(name string) (file.FileHandle, error) {
				return os.Open(name)
			},
			expectError: "some-watch-error",
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			file_path, write, cleanup := setupTempFile(t)
			defer cleanup()

			if c.filePath != "" {
				file_path = c.filePath
			}

			logFileIngestion := file.NewLogFileIngestion(file_path, c.watcherFactory, c.fileOpener)

			errChan := make(chan error, 1)
			defer close(errChan)

			output := make(chan domain.LogEvent, 1)
			defer close(output)

			ctx, closeContext := context.WithCancel(context.Background())
			defer closeContext()

			logFileIngestion.Read(ctx, output, errChan)
			if c.shouldCancelContext {
				closeContext()
			}

			iterations := 1
			hasBody := c.fileBodyWrite != ""
			if hasBody {
				time.Sleep(100 * time.Millisecond)
				write(c.fileBodyWrite)
				lines := strings.Split(c.fileBodyWrite, "\n")
				newLinesCount := 0

				for _, line := range lines {
					if line != "" {
						newLinesCount++
					}
				}

				if newLinesCount > 1 {
					iterations = newLinesCount
				}
			}

			readCount := 0

			for i := 0; i < iterations; i++ {
				select {
				case <-output:
					readCount++
				case err := <-errChan:
					assert.EqualError(t, err, c.expectError)
				case <-time.After(500 * time.Millisecond):
					if c.expectError != "" || hasBody {
						t.Fatal("The result didn't arrive to the channel")
					}
				}
			}

			if hasBody && c.expectError == "" {
				assert.Equal(t, iterations, readCount)
			}
		})
	}
}

func TestWatcherFactoryWithError(t *testing.T) {
	newFsFake := func() (*fsnotify.Watcher, error) { return nil, errors.New("some-fake-error") }

	watcherFactory := file.NewRealWatcherFactory(newFsFake)

	_, err := watcherFactory()
	assert.EqualError(t, err, "some-fake-error")
}

// setupTempFile creates a temporary file and returns its path.
// returns the file path, a function to write to the file, and a cleanup function.
func setupTempFile(t *testing.T) (string, func(string), func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "log-guardian-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	tmpFilePath := filepath.Join(tmpDir, "test.log")

	err = os.WriteFile(tmpFilePath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	writeFunc := func(content string) {
		// Abrimos em modo Append
		f, err := os.OpenFile(tmpFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			t.Errorf("failed to open temp file for writing: %v", err)
			return
		}
		defer f.Close()

		if _, err := f.WriteString(content); err != nil {
			t.Errorf("failed to write to temp file: %v", err)
		}

		err = f.Sync()
		if err != nil {
			t.Errorf("failed to sync temp file: %v", err)
		}
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpFilePath, writeFunc, cleanup
}
