package main

import (
	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/Ceruvia/grader/internal/pool"
)

func main() {
	cfg := config.GetAppConfig()

	// setup sandbox pool
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", cfg.WorkerCount); err != nil {
		panic(err)
	}

	// setup machinery job queue
	if err := machinery.LaunchWorker(cfg); err != nil {
		panic(err)
	}
}
