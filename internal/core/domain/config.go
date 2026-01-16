package domain

import (
	"os"

	"gopkg.in/yaml.v3"
)

var config *RuntimeConfig

type RuntimeConfig struct {
	StdinConfig StdinConfig `yaml:"stdin"`
	FileConfig  FileConfig  `yaml:"file"`
	UnixConfig  UnixConfig  `yaml:"unix"`
}

type ingestionConfig struct {
	Enabled bool `yaml:"enabled"`
}

type StdinConfig struct {
	ingestionConfig
}

type FileConfig struct {
	ingestionConfig

	Folders []FolderConfig `yaml:"folders"`
}

type FolderConfig struct {
	FolderPath  string   `yaml:"folder_path"`
	IgnoreFiles []string `yaml:"ignore_files"`
}

type UnixConfig struct {
	ingestionConfig

	Sockets []UnixSocket `yaml:"sockets"`
}

type UnixSocket struct {
	Address string `yaml:"address"`
	Timeout int    `yaml:"timeout"`
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
