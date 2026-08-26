package main

import (
	"os/exec"
	"strconv"

	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/Ceruvia/grader/internal/monitoring/logging"
	"github.com/Ceruvia/grader/internal/monitoring/metrics"
	"github.com/Ceruvia/grader/internal/pool"
	log "github.com/sirupsen/logrus"
)

func init() {
	cfg := config.GetAppConfig()
	logging.InitLogger(cfg)
}

// cleanupStaleSandboxes cleans up any leftover isolate boxes from previous crashes
func cleanupStaleSandboxes(isolatePath string, maxBoxId int) {
	log.Info("Cleaning up stale sandboxes from previous runs...")
	for i := 1; i <= maxBoxId; i++ {
		cmd := exec.Command(isolatePath, "-b", strconv.Itoa(i), "--cg", "--cleanup")
		// Ignore errors - box might not exist
		_ = cmd.Run()
	}
	log.Info("Stale sandbox cleanup completed")
}

func main() {
	cfg := config.GetAppConfig()

	isolatePath := "/usr/local/bin/isolate"
	
	// Clean up any stale sandboxes before starting
	cleanupStaleSandboxes(isolatePath, cfg.WorkerCount)

	// setup sandbox pool
	if err := pool.NewSandboxPool(isolatePath, cfg.WorkerCount); err != nil {
		panic(err)
	}
	defer pool.Pool.Close()

	go metrics.RunMetricsPusher(cfg)

	// setup machinery job queue
	if err := machinery.LaunchWorker(cfg); err != nil {
		panic(err)
	}
}
