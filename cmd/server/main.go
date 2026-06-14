package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	ossignal "os/signal"
	"syscall"

	"github.com/cloudbridge/cloudbridge/internal/api"
	"github.com/cloudbridge/cloudbridge/internal/relay"
	signalsvc "github.com/cloudbridge/cloudbridge/internal/signal"
	"github.com/cloudbridge/cloudbridge/internal/stun"
)

var version = "0.1.0"

func main() {
	var (
		signalAddr = flag.String("signal-addr", ":10980", "Signal server listen address")
		stunAddr   = flag.String("stun-addr", ":10978", "STUN server listen address")
		relayAddr  = flag.String("relay-addr", ":10988", "Relay server listen address")
		debug      = flag.Bool("debug", false, "Enable debug logging")
		showVer    = flag.Bool("version", false, "Print version and exit")
	)

	flag.Parse()

	if *showVer {
		fmt.Printf("cloudbridge-server v%s\n", version)
		os.Exit(0)
	}

	// Setup structured logging
	logLevel := slog.LevelInfo
	if *debug {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: logLevel,
	})))

	slog.Info("starting CloudBridge server",
		"version", version,
		"signal_addr", *signalAddr,
		"stun_addr", *stunAddr,
		"relay_addr", *relayAddr,
	)

	// Create services
	signalServer := signalsvc.NewServer(signalsvc.ServerConfig{
		Addr: *signalAddr,
		Path: "/signal",
	})

	stunServer := stun.NewServer(stun.ServerConfig{
		Addr: *stunAddr,
	})

	relayServer := relay.NewServer(relay.ServerConfig{
		Addr:        *relayAddr,
		MaxSessions: 1000,
		QuotaBytes:  relay.DefaultQuotaBytes,
		QuotaRate:   relay.DefaultQuotaRate,
	})

	// Start STUN server (UDP, runs in background goroutine)
	if err := stunServer.Start(); err != nil {
		slog.Error("failed to start STUN server", "err", err)
		os.Exit(1)
	}
	slog.Info("STUN server started", "addr", *stunAddr)

	// Start Relay server (TCP, runs in background goroutine)
	if err := relayServer.Start(); err != nil {
		slog.Error("failed to start Relay server", "err", err)
		os.Exit(1)
	}
	slog.Info("Relay server started", "addr", *relayAddr)

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	ossignal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		slog.Info("received signal, shutting down")
		cancel()

		// Gracefully stop services
		stunServer.Close()
		relayServer.Close()
		signalServer.Shutdown(context.Background())
	}()

	// Signal server is blocking (HTTP server)
	slog.Info("Signal server starting", "addr", *signalAddr)

	// Create REST API handler
	deviceAPI := api.NewDeviceAPI(signalServer.Hub(), signalServer.SessionManager())

	if err := signalServer.Start(func(mux *http.ServeMux) {
		deviceAPI.RegisterRoutes(mux)
	}); err != nil {
		slog.Error("Signal server stopped", "err", err)
	}

	<-ctx.Done()
	slog.Info("CloudBridge server stopped")
}