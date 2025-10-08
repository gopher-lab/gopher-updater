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

	var cfg config.Config
	err := envconfig.Process(ctx, &cfg)
	if err != nil {
		xlog.Error(err, "failed to process config")
		os.Exit(1)
	}

	if err = xlog.Init(cfg.LogLevel, cfg.LogFormat); err != nil {
		xlog.Error(err, "failed to initialize logger")
		os.Exit(1)
	}

	// setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	upd, err := updater.New(cfg)
	if err != nil {
		xlog.Error(err, "failed to create updater")
		os.Exit(1)
	}

	if err = upd.Start(ctx); err != nil {
		xlog.Error(err, "updater failed")
	}

	xlog.Info("gopher-updater stopped gracefully")
}
