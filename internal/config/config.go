package config

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type HTTPServer struct {
	Addr string `yaml:"address"`
}

type BatchLimits struct {
	MaxPromptsPerBatch    int `yaml:"max_prompts_per_batch" env-default:"1000"`
	MaxPromptCharacters   int `yaml:"max_prompt_characters" env-default:"1000"`
	MaxResponseCharacters int `yaml:"max_response_characters" env-default:"1000"`
}

type Concurrency struct {
	WorkerPoolSize              int `yaml:"worker_pool_size" env-default:"4"`
	MaxConcurrentInferenceCalls int `yaml:"max_concurrent_inference_calls" env-default:"4"`
}

type Retry struct {
	MaxPromptRetries int `yaml:"max_prompt_retries" env-default:"3"`
	MaxBatchRetries  int `yaml:"max_batch_retries" env-default:"2"`
}

type Backoff struct {
	BaseDelay time.Duration `yaml:"base_delay" env-default:"200ms"`
	MaxDelay  time.Duration `yaml:"max_delay" env-default:"5s"`
}

type Timeouts struct {
	InferenceCallTimeout time.Duration `yaml:"inference_call_timeout" env-default:"10s"`
	CompletedJobTTL      time.Duration `yaml:"completed_job_ttl" env-default:"5m"`
}

type Config struct {
	Env         string `yaml:"env" env-required:"True" env-default:"dev"`
	DbPath      string `yaml:"db_path" env-required:"True"`
	HTTPServer  `yaml:"http_server"`
	BatchLimits `yaml:"batch_limits"`
	Concurrency `yaml:"concurrency"`
	Retry       `yaml:"retry"`
	Backoff     `yaml:"backoff"`
	Timeouts    `yaml:"timeouts"`
}

func Load(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file does not exist at %q", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}
	return &cfg, nil
}

func MustLoad() *Config {
	flagConfigFile := flag.String("config", "", "path to configuration file")
	flag.Parse()

	configPath := *flagConfigFile
	if configPath == "" {
		log.Fatal("Config file path not set")
	}

	slog.Info("obtained", slog.String("configPath", configPath))
	cfg, err := Load(configPath)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	return cfg
}
