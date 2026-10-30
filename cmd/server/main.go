package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/Ceruvia/grader/internal/monitoring/logging"
	"github.com/Ceruvia/grader/internal/monitoring/metrics"
	"github.com/Ceruvia/grader/internal/pool"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.GetAppConfig()
	if err != nil {
		return err
	}

	logging.InitLogger(cfg)
	logging.RunLogger()

	if err := pool.NewSandboxPool("/usr/local/bin/isolate", cfg.WorkerCount); err != nil {
		return fmt.Errorf("initialize sandbox pool: %w", err)
	}
	defer pool.Pool.Close()

	metrics.InitMetricPusher(cfg)
	if metrics.IsInitialized {
		ctx, stopMetrics := context.WithCancel(context.Background())
		defer stopMetrics()
		go metrics.RunMetricsPusher(ctx, pool.Pool)
	}

	if err := machinery.LaunchWorker(cfg); err != nil {
		return fmt.Errorf("grader worker stopped: %w", err)
	}
	return nil
}
