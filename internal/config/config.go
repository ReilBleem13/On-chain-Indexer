package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	SolanaEndpoint   string `env:"SOLANA_ENDPOINT" env-required:"true"`
	PumpFunProgramID string `env:"PUMPFUN_PROGRAM_ID" env-required:"true"`
}

var config *Config

func AppConfig() *Config {
	return config
}

func New() (*Config, error) {
	godotenv.Load(".env")

	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	config = &cfg
	return config, nil
}
