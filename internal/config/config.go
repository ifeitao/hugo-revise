package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Versioning struct {
	DateFormat string
}

type Config struct {
	Versioning Versioning
}

func defaultConfig() Config {
	return Config{
		Versioning: Versioning{
			DateFormat: "2006-01-02",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := defaultConfig()
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType(detectType(path))
	v.SetDefault("versioning.date_format", cfg.Versioning.DateFormat)

	if _, err := os.Stat(path); err == nil {
		if err := v.ReadInConfig(); err != nil {
			return cfg, err
		}
	}

	cfg.Versioning.DateFormat = v.GetString("versioning.date_format")
	return cfg, nil
}

func detectType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return "toml"
	}
}

const LogDirectory = ".hugo-revise"

func EnsureLogDir() error {
	return os.MkdirAll(LogDirectory, 0o755)
}
