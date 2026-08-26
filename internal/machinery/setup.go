package machinery

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	ceruviaTasks "github.com/Ceruvia/grader/internal/tasks"
	"github.com/RichardKnop/machinery/v2"
	"github.com/RichardKnop/machinery/v2/backends/null"
	gcppubsubbroker "github.com/RichardKnop/machinery/v2/brokers/gcppubsub"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

func startServer(cfg *ceruviaConfig.ServerConfig) (*machinery.Server, error) {
	brokerURL := fmt.Sprintf("gcppubsub://%s/%s", cfg.MQCfg.GCPProjectID, cfg.MQCfg.SubscriptionName)

	cnf := &config.Config{
		Broker:       brokerURL,
		DefaultQueue: cfg.MQCfg.TopicName,
	}

	broker, err := gcppubsubbroker.New(cnf, cfg.MQCfg.GCPProjectID, cfg.MQCfg.SubscriptionName)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Pub/Sub broker: %w", err)
	}
	backend := null.New() // No result backend needed - results sent via callback
	lock := eagerlock.New()
	server := machinery.NewServer(cnf, broker, backend, lock)

	// Initialize callback config for tasks
	ceruviaTasks.InitCallback(cfg.CallbackCfg)

	tasksMap := map[string]interface{}{
		"blackbox":              ceruviaTasks.GradeBlackbox,
		"blackbox_with_builder": ceruviaTasks.GradeBlackboxWithBuilder,
	}

	if err := server.RegisterTasks(tasksMap); err != nil {
		log.WithError(err).Error("Failed to register tasks")
		return nil, err
	}

	return server, nil
}

func LaunchWorker(cfg *ceruviaConfig.ServerConfig) error {
	server, err := startServer(cfg)
	if err != nil {
		log.WithError(err).Error("Failed to start Machinery server")
		return err
	}

	worker := server.NewWorker("ceruvia_worker", cfg.WorkerCount)

	// Optional: log any task-specific errors
	errorHandler := func(err error) {
		log.WithError(err).Error("Error occurred while processing task")
	}

	// Before task execution
	preTaskHandler := func(signature *tasks.Signature) {
		log.WithFields(log.Fields{
			"task":         signature.Name,
			"uuid":         signature.UUID,
			"submissionId": signature.Args[0].Value,
		}).Info("Starting task")
	}

	// After task execution
	postTaskHandler := func(signature *tasks.Signature) {
		log.WithFields(log.Fields{
			"task":         signature.Name,
			"uuid":         signature.UUID,
			"submissionId": signature.Args[0].Value,
		}).Info("Finished task")
	}

	worker.SetPostTaskHandler(postTaskHandler)
	worker.SetErrorHandler(errorHandler)
	worker.SetPreTaskHandler(preTaskHandler)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	// Start worker in goroutine
	errorsChan := make(chan error, 1)
	go func() {
		log.WithFields(log.Fields{
			"worker_count":      cfg.WorkerCount,
			"gcp_project_id":    cfg.MQCfg.GCPProjectID,
			"subscription_name": cfg.MQCfg.SubscriptionName,
		}).Info("Launching worker...")
		errorsChan <- worker.Launch()
	}()

	// Wait for shutdown signal or worker error
	select {
	case <-quit:
		log.Info("Received shutdown signal, gracefully stopping worker...")
		worker.Quit()

		// Wait for tasks to complete with timeout
		done := make(chan struct{})
		go func() {
			time.Sleep(30 * time.Second) // Grace period for tasks to complete
			close(done)
		}()
		<-done
		log.Info("Worker stopped gracefully")
		return nil
	case err := <-errorsChan:
		return err
	}
}
