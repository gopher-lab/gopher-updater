package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gopher-lab/gopher-updater/config"
	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/dockerhub"
	"github.com/gopher-lab/gopher-updater/pkg/xlog"
	"github.com/gopher-lab/gopher-updater/updater"
)

func main() {
	xlog.Info("starting gopher-updater")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.New(ctx)
	if err != nil {
		xlog.Error("failed to process config", "err", err)
		os.Exit(1)
	}

	// setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HTTPMaxIdleConns,
			MaxIdleConnsPerHost: cfg.HTTPMaxIdleConnsPerHost,
			MaxConnsPerHost:     cfg.HTTPMaxConnsPerHost,
		},
	}

	cosmosClient := cosmos.NewClient(cfg.RPCURL, httpClient)
	dockerhubClient := dockerhub.NewClient(cfg.DockerHubUser, cfg.DockerHubPassword, httpClient)

	upd := updater.New(cosmosClient, dockerhubClient, cfg)

	if err = upd.Run(ctx); err != nil && err != context.Canceled {
		xlog.Error("updater failed", "err", err)
	}

	xlog.Info("gopher-updater stopped gracefully")
}
