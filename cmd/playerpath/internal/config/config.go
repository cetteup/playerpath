package config

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/cetteup/playerpath/internal/domain/provider"
)

type Config struct {
	Database DatabaseConfig `yaml:"db"`
	Servers  []ServerConfig `yaml:"servers"`
}

type DatabaseConfig struct {
	Hostname     string `yaml:"host"`
	DatabaseName string `yaml:"dbname"`
	Username     string `yaml:"user"`
	Password     string `yaml:"passwd"`
}

type ServerConfig struct {
	IP       string            `yaml:"ip"`
	Provider provider.Provider `yaml:"provider"`
}

func LoadConfig(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}
