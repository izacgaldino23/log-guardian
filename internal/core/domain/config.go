package domain

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

var (
	ErrInvalidShutdownTimeout = errors.New("invalid shutdown timeout")
	ErrFolderPathNotFound     = errors.New("folder path not found")
	ErrInvalidConfigFile      = errors.New("invalid config file")
)

type RuntimeConfig struct {
	ShutdownTimeout int     `yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
	Ingests         Ingests `yaml:"ingests" mapstructure:"ingests"`
}

type Ingests struct {
	Stdin StdinConfig `yaml:"stdin"`
	File  FileConfig  `yaml:"file"`
	Unix  UnixConfig  `yaml:"unix"`
}

type StdinConfig struct {
	Enabled bool `yaml:"enabled"`
}

type FileConfig struct {
	Enabled bool `yaml:"enabled"`

	Folders []FolderConfig `yaml:"folders"`
}

type FolderConfig struct {
	FolderPath  string   `yaml:"folder_path"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

type UnixConfig struct {
	Enabled bool `yaml:"enabled"`

	Sockets []UnixSocket `yaml:"sockets"`
}

type UnixSocket struct {
	Address string `yaml:"address"`
	Timeout int64  `yaml:"timeout"`
}

func (c *RuntimeConfig) Validate() error {
	if c.ShutdownTimeout <= 0 {
		return ErrInvalidShutdownTimeout
	}

	for i, folder := range c.Ingests.File.Folders {
		cleanedPath := filepath.Clean(folder.FolderPath)
		c.Ingests.File.Folders[i].FolderPath = cleanedPath

		if _, err := os.Stat(cleanedPath); os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFolderPathNotFound, cleanedPath)
		}
	}

	return nil
}

func setupViper(c *RuntimeConfig) (err error) {
	v := viper.New()

	v.SetEnvPrefix("APP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Set defaults
	v.SetDefault("shutdown_timeout", 5)

	v.SetDefault("ingests.stdin.enabled", false)
	v.SetDefault("ingests.file.enabled", false)
	v.SetDefault("ingests.unix.enabled", false)

	v.SetDefault("ingests.file.folders", []FolderConfig{})
	v.SetDefault("ingests.unix.sockets", []UnixSocket{})

	// Load config from file
	v.SetConfigFile("./config.yaml")

	err = viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("%w: %s", ErrInvalidConfigFile, err)
		}

		log.Println("Config file not found, using defaults")
	}

	// Unmarshal to struct
	err = v.Unmarshal(c)
	if err != nil {
		return err
	}

	return
}

// LoadConfigs loads the configuration file
func LoadConfigs() (*RuntimeConfig, error) {
	config := &RuntimeConfig{}

	err := setupViper(config)
	if err != nil {
		return nil, err
	}

	err = config.Validate()
	if err != nil {
		return nil, err
	}

	return config, nil
}
