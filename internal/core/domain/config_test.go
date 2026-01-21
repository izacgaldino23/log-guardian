package domain_test

import (
	"log-guardian/internal/core/domain"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeConfig_Validate(t *testing.T) {
	tests := []struct {
		name              string
		config            *domain.RuntimeConfig
		setupTempDir      bool
		expectedError     error
		expectedErrorType error
	}{
		{
			name: "valid config with zero shutdown timeout",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: 0,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: false,
					},
				},
			},
			expectedError: domain.ErrInvalidShutdownTimeout,
		},
		{
			name: "valid config with negative shutdown timeout",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: -5,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: false,
					},
				},
			},
			expectedError: domain.ErrInvalidShutdownTimeout,
		},
		{
			name: "valid config with positive shutdown timeout and no folders",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: 10,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: false,
						Folders: []domain.FolderConfig{},
					},
				},
			},
			expectedError: nil,
		},
		{
			name: "valid config with existing folder",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: 5,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: true,
						Folders: []domain.FolderConfig{
							{FolderPath: "./test-existing-folder"},
						},
					},
				},
			},
			setupTempDir:  true,
			expectedError: nil,
		},
		{
			name: "invalid config with non-existent folder",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: 5,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: true,
						Folders: []domain.FolderConfig{
							{FolderPath: "./non-existent-folder"},
						},
					},
				},
			},
			expectedErrorType: domain.ErrFolderPathNotFound,
		},
		{
			name: "config with unclean folder path gets cleaned",
			config: &domain.RuntimeConfig{
				ShutdownTimeout: 5,
				Ingests: domain.Ingests{
					File: domain.FileConfig{
						Enabled: true,
						Folders: []domain.FolderConfig{
							{FolderPath: "./test-folder/../test-existing-folder"},
						},
					},
				},
			},
			setupTempDir:  true,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var tempDir string
			if tt.setupTempDir {
				tempDir = t.TempDir()
				for i := range tt.config.Ingests.File.Folders {
					if tt.config.Ingests.File.Folders[i].FolderPath == "./test-existing-folder" ||
						tt.config.Ingests.File.Folders[i].FolderPath == "./test-folder/../test-existing-folder" {
						tt.config.Ingests.File.Folders[i].FolderPath = tempDir
					}
				}
			}

			err := tt.config.Validate()

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else if tt.expectedErrorType != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrorType.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.setupTempDir && err == nil {
				for _, folder := range tt.config.Ingests.File.Folders {
					cleanedPath := filepath.Clean(folder.FolderPath)
					assert.Equal(t, cleanedPath, folder.FolderPath)
				}
			}
		})
	}
}

func TestLoadConfigs(t *testing.T) {
	tests := []struct {
		name          string
		setupEnvVars  map[string]string
		expectedError error
	}{
		{
			name:          "load with default config",
			expectedError: nil,
		},
		{
			name: "load with custom shutdown timeout",
			setupEnvVars: map[string]string{
				"APP_SHUTDOWN_TIMEOUT": "15",
			},
			expectedError: nil,
		},
		{
			name: "load with file ingest enabled",
			setupEnvVars: map[string]string{
				"APP_INGESTS_FILE_ENABLED": "true",
			},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for key, value := range tt.setupEnvVars {
				t.Setenv(key, value)
			}

			config, err := domain.LoadConfigs()

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)

				assert.Greater(t, config.ShutdownTimeout, 0)

				if _, exists := tt.setupEnvVars["APP_SHUTDOWN_TIMEOUT"]; exists {
					assert.Equal(t, 15, config.ShutdownTimeout)
				} else {
					assert.Equal(t, 5, config.ShutdownTimeout)
				}

				if enabled, exists := tt.setupEnvVars["APP_INGESTS_FILE_ENABLED"]; exists && enabled == "true" {
					assert.True(t, config.Ingests.File.Enabled)
				} else {
					assert.False(t, config.Ingests.File.Enabled)
				}
			}
		})
	}
}

func TestConfigStructures(t *testing.T) {
	t.Run("RuntimeConfig fields", func(t *testing.T) {
		config := &domain.RuntimeConfig{
			ShutdownTimeout: 10,
			Ingests: domain.Ingests{
				Stdin: domain.StdinConfig{
					Enabled: true,
				},
				File: domain.FileConfig{
					Enabled: false,
					Folders: []domain.FolderConfig{
						{
							FolderPath:  "/var/log",
							IgnoreFiles: []string{"*.tmp", "debug.log"},
						},
					},
				},
				Unix: domain.UnixConfig{
					Enabled: true,
					Sockets: []domain.UnixSocket{
						{
							Address: "/var/log/app.sock",
							Timeout: 30,
						},
					},
				},
			},
		}

		assert.Equal(t, 10, config.ShutdownTimeout)
		assert.True(t, config.Ingests.Stdin.Enabled)
		assert.False(t, config.Ingests.File.Enabled)
		assert.True(t, config.Ingests.Unix.Enabled)
		assert.Len(t, config.Ingests.File.Folders, 1)
		assert.Len(t, config.Ingests.Unix.Sockets, 1)
		assert.Equal(t, "/var/log", config.Ingests.File.Folders[0].FolderPath)
		assert.Equal(t, "*.tmp", config.Ingests.File.Folders[0].IgnoreFiles[0])
		assert.Equal(t, "debug.log", config.Ingests.File.Folders[0].IgnoreFiles[1])
		assert.Equal(t, "/var/log/app.sock", config.Ingests.Unix.Sockets[0].Address)
		assert.Equal(t, int64(30), config.Ingests.Unix.Sockets[0].Timeout)
	})

	t.Run("IngestionConfig embedding", func(t *testing.T) {
		stdinConfig := domain.StdinConfig{
			Enabled: true,
		}
		assert.True(t, stdinConfig.Enabled)

		fileConfig := domain.FileConfig{
			Enabled: false,
		}
		assert.False(t, fileConfig.Enabled)

		unixConfig := domain.UnixConfig{
			Enabled: true,
		}
		assert.True(t, unixConfig.Enabled)
	})
}

func TestConfigValidationEdgeCases(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		config := &domain.RuntimeConfig{}
		err := config.Validate()
		assert.ErrorIs(t, err, domain.ErrInvalidShutdownTimeout)
	})

	t.Run("config with only shutdown timeout", func(t *testing.T) {
		config := &domain.RuntimeConfig{
			ShutdownTimeout: 5,
			Ingests: domain.Ingests{
				File: domain.FileConfig{
					Enabled: false,
				},
			},
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("config with multiple folders one invalid", func(t *testing.T) {
		tempDir := t.TempDir()
		config := &domain.RuntimeConfig{
			ShutdownTimeout: 5,
			Ingests: domain.Ingests{
				File: domain.FileConfig{
					Enabled: true,
					Folders: []domain.FolderConfig{
						{FolderPath: tempDir},
						{FolderPath: "/non/existent/path"},
					},
				},
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non")
		assert.Contains(t, err.Error(), "existent")
		assert.Contains(t, err.Error(), "path")
	})

	t.Run("config with relative path that exists", func(t *testing.T) {
		// TODO: Refactor this test when we get the folder searching for files
		t.Skip()
		currentDir, err := os.Getwd()
		require.NoError(t, err)

		config := &domain.RuntimeConfig{
			ShutdownTimeout: 5,
			Ingests: domain.Ingests{
				File: domain.FileConfig{
					Enabled: true,
					Folders: []domain.FolderConfig{
						{FolderPath: "."},
					},
				},
			},
		}

		err = config.Validate()
		assert.NoError(t, err)
		assert.Equal(t, currentDir, config.Ingests.File.Folders[0].FolderPath)
	})
}

func TestConfigConstants(t *testing.T) {
	assert.Equal(t, "invalid shutdown timeout", domain.ErrInvalidShutdownTimeout.Error())
	assert.Equal(t, "folder path not found", domain.ErrFolderPathNotFound.Error())
	assert.Equal(t, "invalid config file", domain.ErrInvalidConfigFile.Error())
}
