package config

import (
	"golang.org/x/exp/slog"
	"log"
	"os"
	"time"
	"url-shortener/internal/lib/logger/handlers/slogpretty"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		HTTPServer  `yaml:"http_server"`
		Env         string `yaml:"env" env-default:"local"`
		StoragePath string `yaml:"storage_path" env-required:"true"`
		Log         Log    `yaml:"log"`
	}
	HTTPServer struct {
		Address     string        `yaml:"address" env-default:"localhost:8080" env:"HTTP_SERVER_ADDR"`
		Timeout     time.Duration `yaml:"timeout" env-default:"4s"`
		IdleTimeout time.Duration `yaml:"idle_timeout" env-default:"60s"`
		User        string        `yaml:"user" env-required:"true"`
		Password    string        `yaml:"password" env-required:"true" env:"HTTP_SERVER_PASSWORD"`
	}
	Log struct {
		Slog Slog `yaml:"slog"`
	}
	Slog struct {
		Level     slog.Level              `yaml:"level"`
		AddSource bool                    `yaml:"add_source"`
		Format    slogpretty.FieldsFormat `yaml:"format"` // json, text or pretty
		Pretty    PrettyLog               `yaml:"pretty"`
	}

	PrettyLog struct {
		FieldsFormat slogpretty.FieldsFormat `yaml:"fields_format"` // json, json-indent or yaml
		Emoji        bool                    `yaml:"emoji"`
		TimeLayout   string                  `yaml:"time_layout"`
	}
)

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
