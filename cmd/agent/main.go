package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudbridge/cloudbridge/internal/agent"
)

func main() {
	cfg := agent.DefaultConfig()

	var (
		serverURL  = flag.String("server", cfg.ServerURL, "WebSocket URL of the signal server")
		deviceName = flag.String("name", cfg.DeviceName, "Human-readable device name")
		version    = flag.Bool("version", false, "Print version and exit")
		debug      = flag.Bool("debug", false, "Enable debug logging")
	)

	flag.Parse()

	if *version {
		fmt.Println("cloudbridge-agent v0.1.0")
		os.Exit(0)
	}

	// Setup logging
	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	cfg.ServerURL = *serverURL
	cfg.DeviceName = *deviceName

	slog.Info("starting CloudBridge agent",
		"server", cfg.ServerURL,
		"name", cfg.DeviceName,
	)

	a, err := agent.New(cfg)
	if err != nil {
		slog.Error("failed to create agent", "err", err)
		os.Exit(1)
	}

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		slog.Info("received signal, shutting down", "signal", sig)
		cancel()
	}()

	if err := a.Run(ctx); err != nil {
		slog.Error("agent stopped", "err", err)
		os.Exit(1)
	}
}