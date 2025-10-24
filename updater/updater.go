package updater

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gopher-lab/gopher-updater/config"
	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/dockerhub"
	"github.com/gopher-lab/gopher-updater/pkg/xlog"
)

// Updater is responsible for monitoring the chain and retagging images.
type Updater struct {
	cosmosClient    cosmos.ClientInterface
	dockerhubClient dockerhub.ClientInterface
	cfg             *config.Config
}

// New creates a new Updater.
func New(
	cosmosClient cosmos.ClientInterface,
	dockerhubClient dockerhub.ClientInterface,
	cfg *config.Config,
) *Updater {
	return &Updater{
		cosmosClient:    cosmosClient,
		dockerhubClient: dockerhubClient,
		cfg:             cfg,
	}
}

// Run starts the updater loop. It checks for upgrades periodically.
func (u *Updater) Run(ctx context.Context) error {
	ticker := time.NewTicker(u.cfg.PollInterval)
	defer ticker.Stop()

	xlog.Info("performing initial check for software upgrade proposal")
	if err := u.CheckAndProcessUpgrade(ctx); err != nil {
		xlog.Error("error checking for upgrade on initial check", "err", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			xlog.Info("checking for software upgrade proposal")
			if err := u.CheckAndProcessUpgrade(ctx); err != nil {
				xlog.Error("error checking for upgrade", "err", err)
			}
		}
	}
}

// CheckAndProcessUpgrade fetches all passed upgrade plans and processes the next available one.
func (u *Updater) CheckAndProcessUpgrade(ctx context.Context) error {
	plans, err := u.cosmosClient.GetUpgradePlans(ctx)
	if err != nil {
		return fmt.Errorf("failed to get upgrade plans: %w", err)
	}

	if len(plans) == 0 {
		xlog.Info("no passed software upgrade proposals found")
		return nil
	}

	currentHeight, err := u.cosmosClient.GetLatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block height: %w", err)
	}

	var pendingPlans []cosmos.Plan
	for _, plan := range plans {
		proposalHeight, err := strconv.ParseInt(plan.Height, 10, 64)
		if err != nil {
			xlog.Error("failed to parse upgrade height, skipping plan", "plan", plan.Name, "height", plan.Height, "err", err)
			continue
		}
		upgradeHeight := proposalHeight - 1

		if currentHeight >= upgradeHeight {
			targetTag := u.cfg.TargetPrefix + plan.Name
			exists, err := u.dockerhubClient.TagExists(ctx, u.cfg.RepoPath, targetTag)
			if err != nil {
				return fmt.Errorf("failed to check if target tag exists for plan %s: %w", plan.Name, err)
			}
			if !exists {
				pendingPlans = append(pendingPlans, plan)
			}
		}
	}

	if len(pendingPlans) == 0 {
		xlog.Info("no pending upgrades to process")
		return nil
	}

	// Sort by height to process the oldest pending upgrade first
	sort.Slice(pendingPlans, func(i, j int) bool {
		h1, _ := strconv.ParseInt(pendingPlans[i].Height, 10, 64)
		h2, _ := strconv.ParseInt(pendingPlans[j].Height, 10, 64)
		return h1 < h2
	})

	nextPlan := pendingPlans[0]
	xlog.Info("found pending upgrade to process", "plan", nextPlan.Name, "height", nextPlan.Height)

	return u.processUpgrade(ctx, &nextPlan)
}

func (u *Updater) processUpgrade(ctx context.Context, plan *cosmos.Plan) error {
	sourceTag := u.cfg.SourcePrefix + plan.Name
	targetTag := u.cfg.TargetPrefix + plan.Name

	xlog.Info("retagging image", "repo", u.cfg.RepoPath, "source", sourceTag, "target", targetTag)

	if u.cfg.DryRun {
		xlog.Info("dry run enabled, skipping retag")
		return nil
	}

	err := u.dockerhubClient.RetagImage(ctx, u.cfg.RepoPath, sourceTag, targetTag)
	if err != nil {
		return fmt.Errorf("failed to retag image: %w", err)
	}

	xlog.Info("successfully retagged image")
	return nil
}
