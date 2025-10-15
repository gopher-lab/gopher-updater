package health

import (
	"context"
	"fmt"

	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/dockerhub"
)

// Checker performs readiness checks for the application.
type Checker struct {
	cosmosClient    cosmos.ClientInterface
	dockerhubClient dockerhub.ClientInterface
	repoPath        string
}

// NewChecker creates a new health checker.
func NewChecker(
	cosmosClient cosmos.ClientInterface,
	dockerhubClient dockerhub.ClientInterface,
	repoPath string,
) *Checker {
	return &Checker{
		cosmosClient:    cosmosClient,
		dockerhubClient: dockerhubClient,
		repoPath:        repoPath,
	}
}

// Ready checks if the application is ready to serve traffic.
// It verifies connectivity to both the Cosmos chain and DockerHub.
func (c *Checker) Ready(ctx context.Context) error {
	// Check Cosmos connection
	if _, err := c.cosmosClient.GetLatestBlockHeight(ctx); err != nil {
		return fmt.Errorf("cosmos connection failed: %w", err)
	}

	// Check DockerHub connection and authentication.
	// We check for a tag that is highly unlikely to exist.
	if _, err := c.dockerhubClient.TagExists(ctx, c.repoPath, "readiness-check"); err != nil {
		return fmt.Errorf("dockerhub connection failed: %w", err)
	}

	return nil
}
