package config

import (
	"context"
	"time"

	"github.com/sethvargo/go-envconfig"
)

// Config holds the application configuration.
type Config struct {
	RPCURL            string        `env:"RPC_URL,default=http://localhost:1317"`
	DockerHubUser     string        `env:"DOCKERHUB_USER,required"`
	DockerHubPassword string        `env:"DOCKERHUB_PASSWORD,required"`
	RepoPath          string        `env:"REPO_PATH,required"`
	SourcePrefix      string        `env:"SOURCE_PREFIX,default=release-"`
	TargetPrefix      string        `env:"TARGET_PREFIX,required"`
	PollInterval      time.Duration `env:"POLL_INTERVAL,default=1m"`

	HTTPMaxIdleConns        int `env:"HTTP_MAX_IDLE_CONNS,default=100"`
	HTTPMaxIdleConnsPerHost int `env:"HTTP_MAX_IDLE_CONNS_PER_HOST,default=10"`
	HTTPMaxConnsPerHost     int `env:"HTTP_MAX_CONNS_PER_HOST,default=10"`
}

// New loads the configuration from environment variables.
func New(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
