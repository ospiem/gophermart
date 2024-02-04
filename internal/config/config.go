package config

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	Endpoint          string `env:"RUN_ADDRESS"`
	DSN               string `env:"DATABASE_URI"`
	AccrualSysAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	LogLevel          string `env:"LOG_LEVEL"`
	JWTSecretKey      string `env:"SECRET_KEY"`
	Pagination        int    `env:"DB_PAGINATION"`
	WorkersNum        int    `env:"WORKERS_NUMBER"`
}

// TODO: separate configs
func New() (Config, error) {
	var c Config
	err := env.Parse(&c)
	if err != nil {
		return Config{}, fmt.Errorf("cannot parse environment variables: %w", err)
	}
	parseFlag(&c)
	return c, nil
}

func parseFlag(c *Config) {
	var ep, dsn, accrualEp string
	flag.StringVar(&ep, "a", "", "set service endpoint")
	flag.StringVar(&dsn, "d", "", "set DSN endpoint")
	flag.StringVar(&accrualEp, "r", "", "set accrual system endpoint")
	flag.StringVar(&c.LogLevel, "l", "info", "set log level")
	flag.IntVar(&c.Pagination, "pagination", 10, "set pagination for DB pagination")
	flag.IntVar(&c.WorkersNum, "w", 3, "set number of workers")

	flag.Parse()

	if ep != "" {
		c.Endpoint = ep
	}
	if dsn != "" {
		c.DSN = dsn
	}
	if accrualEp != "" {
		c.AccrualSysAddress = accrualEp
	}
}
