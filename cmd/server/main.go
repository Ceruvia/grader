package main

import (
	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/Ceruvia/grader/internal/monitoring/logging"
	"github.com/Ceruvia/grader/internal/monitoring/metrics"
	"github.com/Ceruvia/grader/internal/pool"
)

func init() {
	cfg := config.GetAppConfig()

	logging.InitLogger(cfg)
	logging.RunLogger()

	metrics.InitMetricPusher(cfg)
	if metrics.IsInitialized {
		go metrics.RunMetricsPusher()
	}
}

func main() {
	cfg := config.GetAppConfig()

	if err := pool.NewSandboxPool("/usr/local/bin/isolate", cfg.WorkerCount); err != nil {
		panic(err)
	}

	if err := machinery.LaunchWorker(cfg); err != nil {
		panic(err)
	}
}
