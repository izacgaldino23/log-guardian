package file_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log-guardian/internal/adapters/input/file"
	"log-guardian/internal/core/domain"
	"log-guardian/internal/core/ports"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

type testCase struct {
	name                string
	filePath            string
	expectError         string
	fileBodyWrite       string
	shouldCancelContext bool
	watcherProvider     func() (file.FileWatcher, error)
	fileSystem          func() file.FileSystem
}

func TestLogFileIngestion(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	idGen := domain.NewMockIDGenerator(ctrl)
	idGen.EXPECT().Generate().AnyTimes().Return("some-id", nil)

	var (
		fileSystem      = file.OSFileSystem{}
		watcherProvider = file.WatcherProvider{}
	)

	fileOpener := func() file.FileSystem {
		return fileSystem
	}

	testCases := []testCase{
		{
			name: "ShouldFailBecauseFileOpeningFails",
			watcherProvider: func() (file.FileWatcher, error) {
				mock := file.NewMockFileWatcher(ctrl)
				mock.EXPECT().Close().AnyTimes()
				return mock, nil
			},
			fileSystem: func() file.FileSystem {
				fileSystemMock := file.NewMockFileSystem(ctrl)
				fileSystemMock.EXPECT().Open(gomock.Any()).Return(nil, errors.New("some-open-file-error"))
				return fileSystemMock
			},
			expectError: "some-open-file-error",
		},
		{
			name: "ShouldFailBecauseFileSeekFails",
			watcherProvider: func() (file.FileWatcher, error) {
				mock := file.NewMockFileWatcher(ctrl)
				mock.EXPECT().Close().AnyTimes()
				return mock, nil
			},
			fileSystem: func() file.FileSystem {
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), errors.New("some-seek-error"))
				mockFile.EXPECT().Close().Return(nil)

				fileSystemMock := file.NewMockFileSystem(ctrl)
				fileSystemMock.EXPECT().Open(gomock.Any()).Return(mockFile, nil)

				return fileSystemMock
			},
			expectError: "some-seek-error",
		},
		{
			name: "ShouldFailBecauseFileWatcherAddFails",
			watcherProvider: func() (file.FileWatcher, error) {
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add("file_path").Return(errors.New("some-watcher-add-error"))

				return mockWatcher, nil
			},
			fileSystem: func() file.FileSystem {
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), nil)
				mockFile.EXPECT().Close().AnyTimes()

				fileSystemMock := file.NewMockFileSystem(ctrl)
				fileSystemMock.EXPECT().Open(gomock.Any()).Return(mockFile, nil)

				return fileSystemMock
			},
			filePath:    "file_path",
			expectError: "some-watcher-add-error",
		},
		{
			name: "ShouldEndBecauseContextIsCanceled",
			watcherProvider: func() (file.FileWatcher, error) {
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()
				mockWatcher.EXPECT().Events().AnyTimes()
				mockWatcher.EXPECT().Errors().AnyTimes()

				return mockWatcher, nil
			},
			fileSystem:          fileOpener,
			shouldCancelContext: true,
		},
		{
			name: "ShouldSkipBecauseEventIsEmpty",
			watcherProvider: func() (file.FileWatcher, error) {
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
			fileSystem: fileOpener,
		},
		{
			name: "ShouldSkipBecauseEventIsNotWriteKind",
			watcherProvider: func() (file.FileWatcher, error) {
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
			fileSystem: fileOpener,
		},
		{
			name:            "ShouldBreakTheLoopBecauseGotTheEOFError",
			watcherProvider: watcherProvider.Create,
			fileSystem:      fileOpener,
			fileBodyWrite:   "hello\nworld",
		},
		{
			name: "ShouldFailBecauseReadStringFails",
			watcherProvider: func() (file.FileWatcher, error) {
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
			fileSystem: func() file.FileSystem {
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), nil)
				mockFile.EXPECT().Read(gomock.Any()).Return(0, errors.New("some-read-error"))
				mockFile.EXPECT().Close().AnyTimes()

				fileSystemMock := file.NewMockFileSystem(ctrl)
				fileSystemMock.EXPECT().Open(gomock.Any()).Return(mockFile, nil)

				return fileSystemMock
			},
			expectError: "some-read-error",
		},
		{
			name:            "ShouldContinueTheLoopIfTheLineStringIsEmpty",
			watcherProvider: watcherProvider.Create,
			fileSystem:      fileOpener,
			fileBodyWrite:   "hello\nworld\n\n",
		},
		{
			name:            "ShouldOutputTheEventLogWithSuccess",
			watcherProvider: watcherProvider.Create,
			fileSystem:      fileOpener,
			fileBodyWrite:   "hello\nworld\nfinal",
		},
		{
			name: "ShouldFailBecauseWatcherErrorsChannelReturnsError",
			watcherProvider: func() (file.FileWatcher, error) {
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
			fileSystem:  fileOpener,
			expectError: "some-watch-error",
		},
		{
			name: "ShouldFailBecauseWatcherErrorsChannelReturnsNilError",
			watcherProvider: func() (file.FileWatcher, error) {
				mockWatcher := file.NewMockFileWatcher(ctrl)
				mockWatcher.EXPECT().Close().AnyTimes()
				mockWatcher.EXPECT().Add(gomock.Any()).AnyTimes()

				errorChan := make(chan error, 1)
				errorChan <- nil
				close(errorChan)

				mockWatcher.EXPECT().Errors().AnyTimes().Return(errorChan)

				inputChan := make(chan fsnotify.Event, 1)
				inputChan <- fsnotify.Event{
					Op: fsnotify.Write,
				}
				close(inputChan)

				mockWatcher.EXPECT().Events().AnyTimes().Return(inputChan)

				return mockWatcher, nil
			},
			fileSystem: func() file.FileSystem {
				mockFile := file.NewMockFileHandle(ctrl)
				mockFile.EXPECT().Seek(gomock.Any(), io.SeekEnd).Return(int64(0), nil)

				input := "One result\n"
				r := bytes.NewBufferString(input)

				mockFile.EXPECT().Read(gomock.Any()).AnyTimes().DoAndReturn(func(b []byte) (int, error) {
					return r.Read(b)
				})

				mockFile.EXPECT().Close().AnyTimes()

				fileSystemMock := file.NewMockFileSystem(ctrl)
				fileSystemMock.EXPECT().Open(gomock.Any()).Return(mockFile, nil)

				return fileSystemMock
			},
		},
	}

	for _, c := range testCases {
		runTestLogFileIngestion(t, idGen, c)
	}
}

func runTestLogFileIngestion(t *testing.T, idGen domain.IDGenerator, c testCase) {
	t.Run(c.name, func(t *testing.T) {
		file_path, write, cleanup := setupTempFile(t)
		defer cleanup()

		if c.filePath != "" {
			file_path = c.filePath
		}

		provider, err := c.watcherProvider()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}
		defer provider.Close()

		logFileIngestion := file.NewLogFileIngestion(file_path, provider, c.fileSystem(), idGen)

		errChan := make(chan error, 1)
		defer close(errChan)

		output := make(chan domain.LogEvent, 1)
		defer close(output)

		ctx, closeContext := context.WithCancel(context.Background())
		defer closeContext()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		shutdownMock := ports.NewMockIngestionShutdown(ctrl)
		shutdownMock.EXPECT().OnShutdown().AnyTimes()

		logFileIngestion.Read(ctx, output, errChan, shutdownMock)
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
