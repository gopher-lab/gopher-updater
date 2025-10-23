package main

import (
	"context"
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
	"github.com/labstack/echo/v4"
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

	// Setup signal handling
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		xlog.Info("shutdown signal received")
		cancel()
	}()

	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.HTTPMaxIdleConns,
			MaxIdleConnsPerHost: cfg.HTTPMaxIdleConnsPerHost,
			MaxConnsPerHost:     cfg.HTTPMaxConnsPerHost,
		},
	}

	cosmosClient := cosmos.NewClient(cfg.APIURL, httpClient)
	dockerhubClient := dockerhub.NewClient(cfg.DockerHubUser, cfg.DockerHubPassword, httpClient)
	checker := health.NewChecker(cosmosClient, dockerhubClient, cfg.RepoPath)

	// Start HTTP server and set up graceful shutdown
	e := startHTTPServer(cfg, checker, cancel)
	defer func() {
		xlog.Info("shutting down http server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := e.Shutdown(shutdownCtx); err != nil {
			xlog.Error("http server shutdown failed", "err", err)
		}
	}()

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

func startHTTPServer(cfg *config.Config, checker *health.Checker, cancel context.CancelFunc) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// --- Routes ---
	e.GET("/healthz", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.GET("/readyz", func(c echo.Context) error {
		if err := checker.Ready(c.Request().Context()); err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{
				"status": "unready",
				"error":  err.Error(),
			})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ready"})
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// pprof routes
	pprofGroup := e.Group("/debug/pprof")
	pprofGroup.GET("/*", echo.WrapHandler(http.HandlerFunc(pprof.Index)))
	pprofGroup.GET("/cmdline", echo.WrapHandler(http.HandlerFunc(pprof.Cmdline)))
	pprofGroup.GET("/profile", echo.WrapHandler(http.HandlerFunc(pprof.Profile)))
	pprofGroup.GET("/symbol", echo.WrapHandler(http.HandlerFunc(pprof.Symbol)))
	pprofGroup.GET("/trace", echo.WrapHandler(http.HandlerFunc(pprof.Trace)))

	go func() {
		addr := fmt.Sprintf(":%s", cfg.HTTPPort)
		xlog.Info("starting HTTP server for metrics, health, and pprof", "port", cfg.HTTPPort)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			xlog.Error("http server failed", "err", err)
			cancel()
		}
	}()

	return e
}
