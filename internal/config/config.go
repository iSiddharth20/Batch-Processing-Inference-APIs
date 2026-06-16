package config

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Addr string `yaml:"address"`
}

type Config struct {
	Env        string `yaml:"env" env-required:"True" env-default:"dev"`
	DbPath     string `yaml:"db_path" env-required:"True"`
	HTTPServer `yaml:"http_server"`
}

func MustLoad() *Config {
	var configPath string

	flagConfigFile := flag.String("config", "", "path to configuration file")
	flag.Parse()

	configPath = *flagConfigFile
	if configPath == "" {
		log.Fatal("Config file path not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal("Config file does not exist at specified path")
	}

	var cfg Config
	slog.Info("obtained", slog.String("configPath", configPath))
	err := cleanenv.ReadConfig(configPath, &cfg)
	if err != nil {
		log.Fatalf("Cannot read config file: %s", err.Error())
	}

	return &cfg
}
