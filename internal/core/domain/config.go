package domain

import (
	"os"

	"gopkg.in/yaml.v3"
)

var config *RuntimeConfig

type RuntimeConfig struct {
	ShutdownTimeout int     `yaml:"shutdown_timeout"`
	Ingests         Ingests `yaml:"ingests"`
}

type Ingests struct {
	Stdin StdinConfig `yaml:"stdin"`
	File  FileConfig  `yaml:"file"`
	Unix  UnixConfig  `yaml:"unix"`
}

type IngestionConfig struct {
	Enabled bool `yaml:"enabled"`
}

type StdinConfig struct {
	IngestionConfig
}

type FileConfig struct {
	IngestionConfig

	Folders []FolderConfig `yaml:"folders"`
}

type FolderConfig struct {
	FolderPath  string   `yaml:"folder_path"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

type UnixConfig struct {
	IngestionConfig

	Sockets []UnixSocket `yaml:"sockets"`
}

type UnixSocket struct {
	Address string `yaml:"address"`
	Timeout int64  `yaml:"timeout"`
}

// LoadConfigs loads the configuration file
func LoadConfigs() (*RuntimeConfig, error) {
	config = &RuntimeConfig{}

	// search for the config file
	configData, err := os.ReadFile("config.yaml")
	if err != nil {
		return nil, err
	}

	// pase the config
	err = yaml.Unmarshal(configData, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func GetConfig() *RuntimeConfig {
	return config
}
