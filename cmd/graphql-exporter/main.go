package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/vinted/graphql-exporter/internal/config"
	"github.com/vinted/graphql-exporter/internal/prometheus"
)

func main() {
	var (
		configPath        string
		httpListenAddress string
		mode              string
		pushEndpoint      string
	)
	logOpts := &slog.HandlerOptions{
		Level: getLogLevelFromEnv(),
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, logOpts))
	slog.SetDefault(logger)
	flag.StringVar(&configPath, "config_path", "/etc/graphql-exporter/config.json", "Path to config directory.")
	flag.StringVar(&httpListenAddress, "http_listen_address", "0.0.0.0:9353", "Address to bind to.")
	flag.StringVar(&mode, "mode", "pull", "prometheus mode (push or pull); default pull")
	flag.StringVar(&pushEndpoint, "push_endpoint", "localhost:1234", "http endpoint to push mode")
	flag.Parse()
	err := config.Init(configPath)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to read configuration: %s", err))
		os.Exit(1)
	}
	if mode == "push" {
		prometheus.Push(pushEndpoint, "graphql-exporter")
	} else {
		prometheus.Start(httpListenAddress)
	}
}

func getLogLevelFromEnv() slog.Level {
	levelStr := os.Getenv("LOG_LEVEL")
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.Level(100) // Custom level higher than any standard level, so silent by default

	}
}
