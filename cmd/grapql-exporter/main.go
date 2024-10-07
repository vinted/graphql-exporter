package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/vinted/graphql-exporter/internal/config"
	"github.com/vinted/graphql-exporter/internal/prometheus"
)

func main() {
	var (
		configPath        string
		httpListenAddress string
	)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	flag.StringVar(&configPath, "configPath", "/etc/graphql-exporter/config.json", "Path to config directory.")
	flag.StringVar(&httpListenAddress, "HTTPListenAddress", "0.0.0.0:9353", "Address to bind to.")
	flag.Parse()
	err := config.Init(configPath)
	if err != nil {
		slog.Error(fmt.Sprintf("failed to read configuration: %s", err))
		os.Exit(1)
	}
	prometheus.Start(httpListenAddress)
}
