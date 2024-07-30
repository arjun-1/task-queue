package queue

import (
	"context"
	"net/url"

	"github.com/sethvargo/go-envconfig"
)

type AppConfig struct {
	PostgresURL url.URL `env:"POSTGRES_URL"`
}

func AppConfigFromEnv(ctx context.Context) (*AppConfig, error) {
	var c AppConfig
	if err := envconfig.Process(ctx, &c); err != nil {
		return nil, err
	}

	return &c, nil
}
