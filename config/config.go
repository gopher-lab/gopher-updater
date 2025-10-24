package config

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/sethvargo/go-envconfig"
)

// Config holds the application configuration.
type Config struct {
	APIURL            *url.URL      `env:"API_URL,default=http://localhost:1317"`
	DockerHubUser     string        `env:"DOCKERHUB_USER"`
	DockerHubPassword string        `env:"DOCKERHUB_PASSWORD"`
	RepoPath          string        `env:"REPO_PATH,required"`
	SourcePrefix      string        `env:"SOURCE_PREFIX,default=release-"`
	TargetPrefix      string        `env:"TARGET_PREFIX,required"`
	PollInterval      time.Duration `env:"POLL_INTERVAL,default=1m"`
	DryRun            bool          `env:"DRY_RUN,default=false"`

	HTTPMaxIdleConns        int    `env:"HTTP_MAX_IDLE_CONNS,default=100"`
	HTTPMaxIdleConnsPerHost int    `env:"HTTP_MAX_IDLE_CONNS_PER_HOST,default=10"`
	HTTPMaxConnsPerHost     int    `env:"HTTP_MAX_CONNS_PER_HOST,default=10"`
	HTTPPort                string `env:"HTTP_PORT,default=8080"`
}

// New loads the configuration from environment variables.
func New(ctx context.Context) (*Config, error) {
	var cfg Config
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, err
	}

	if !cfg.DryRun {
		if cfg.DockerHubUser == "" {
			return nil, fmt.Errorf("DOCKERHUB_USER is required when not in dry-run mode")
		}
		if cfg.DockerHubPassword == "" {
			return nil, fmt.Errorf("DOCKERHUB_PASSWORD is required when not in dry-run mode")
		}
	}

	return &cfg, nil
}
