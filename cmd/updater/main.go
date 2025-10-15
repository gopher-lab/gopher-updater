package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gopher-lab/gopher-updater/config"
	"github.com/gopher-lab/gopher-updater/cosmos"
	"github.com/gopher-lab/gopher-updater/dockerhub"
	"github.com/gopher-lab/gopher-updater/health"
	"github.com/gopher-lab/gopher-updater/pkg/xlog"
	"github.com/gopher-lab/gopher-updater/updater"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
		xlog.Info("shutdown signal received")
		cancel()
	}()

	// Start HTTP server for health, metrics, and pprof
	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
			xlog.Error("failed to write health check response", "err", err)
		}
	})
	httpMux.Handle("/metrics", promhttp.Handler())
	httpMux.HandleFunc("/debug/pprof/", pprof.Index)
	httpMux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	httpMux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	httpMux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	httpMux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: httpMux,
	}
	go func() {
		xlog.Info("starting HTTP server for metrics, health, and pprof", "port", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			xlog.Error("http server failed", "err", err)
		}
	}()

	// Graceful shutdown for http server
	defer func() {
		xlog.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			xlog.Error("http server shutdown failed", "err", err)
		}
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

	// Set up readiness checker
	checker := health.NewChecker(cosmosClient, dockerhubClient, cfg.RepoPath)
	httpMux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := checker.Ready(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{"status": "unready", "error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
	})

	upd := updater.New(cosmosClient, dockerhubClient, cfg)

	// Run the main updater loop
	go func() {
		if err := upd.Run(ctx); err != nil && err != context.Canceled {
			xlog.Error("updater failed", "err", err)
		}
		cancel() // if the updater stops for any reason, cancel the context
	}()

	<-ctx.Done() // Wait for shutdown signal or updater to finish

	xlog.Info("gopher-updater stopped gracefully")
}
