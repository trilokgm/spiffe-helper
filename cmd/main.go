package main

import (
	"context"
	"flag"
	"github.com/apex/log"
	"spiffe-helper/cmd/app"
	"spiffe-helper/cmd/config"
)

func main() {
	// 0. Load configuration
	// 1. Create Sidecar
	// 2. Run Sidecar's Daemon

	configFile := flag.String("config", "helper.conf", "<configFile> Configuration file path")
	flag.Parse()

	cfg, err := config.ParseConfig(*configFile)
	if err != nil {
		log.Fatalf("error parsing configuration file: %v\n%v", *configFile, err)
	}

	log.Infof("Sidecar is up! Will use agent at %s\n\n", cfg.AgentAddress)
	if cfg.Cmd == "" {
		log.Warn("Warning: no cmd defined to execute.\n")
	}

	log.Infof("Using configuration file: %v\n", *configFile)

	sidecar, err := app.NewSidecar(cfg)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	err = sidecar.RunDaemon(ctx)
	if err != nil {
		panic(err)
	}
}
