package main

import (
	"flag"
	"log"

	"github.com/vinted/graphql-exporter/internal/config"
	"github.com/vinted/graphql-exporter/internal/prometheus"
)

func main() {
	var (
		configPath        string
		httpListenAddress string
	)
	flag.StringVar(&configPath, "configPath", "/etc/graphql-exporter/config.json", "Path to config directory.")
	flag.StringVar(&httpListenAddress, "HTTPListenAddress", "0.0.0.0:9353", "Address to bind to.")
	flag.Parse()
	err := config.Init(configPath)
	if err != nil {
		log.Fatalf("Failed to read configuration. %v", err)
	}
	prometheus.Start(httpListenAddress)
}
