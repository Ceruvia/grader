package machinery

import (
	"context"
	"encoding/json"
	"strings"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	ceruviaTasks "github.com/Ceruvia/grader/internal/tasks"
	"github.com/RichardKnop/machinery/v2"
	amqpbackend "github.com/RichardKnop/machinery/v2/backends/amqp"
	amqpbroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

func startServer(cfg *ceruviaConfig.ServerConfig) (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:          cfg.MQCfg.BrokerURL,
		DefaultQueue:    cfg.MQCfg.QueueName,
		ResultBackend:   cfg.MQCfg.ResultBackendURL,
		ResultsExpireIn: cfg.MQCfg.ResultsExpireIn,
		AMQP: &config.AMQPConfig{
			Exchange:      "machinery_exchange",
			ExchangeType:  "direct",
			BindingKey:    "machinery_task",
			PrefetchCount: cfg.WorkerCount,
		},
	}

	broker := amqpbroker.New(cnf)
	backend := amqpbackend.New(cnf)
	lock := eagerlock.New()
	server := machinery.NewServer(cnf, broker, backend, lock)

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
		fields := log.Fields{"task": signature.Name, "uuid": signature.UUID}
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(fields).Errorf("Recovered panic in post task handler: %v", r)
			}
		}()

		if len(signature.Args) == 0 {
			log.WithFields(fields).Error("Task has no submission id, callback skipped")
			return
		}
		submissionId, ok := signature.Args[0].Value.(string)
		if !ok {
			log.WithFields(fields).Errorf("Task submission id has type %T, callback skipped", signature.Args[0].Value)
			return
		}
		fields["submission_id"] = submissionId

		payload, err := json.Marshal(struct {
			TaskId string `json:"task_id"`
		}{TaskId: signature.UUID})
		if err != nil {
			log.WithFields(fields).WithError(err).Error("Failed to encode callback payload")
			return
		}

		callbackUrl := strings.ReplaceAll(cfg.BackendCallbackEndpoint, "{submission_id}", submissionId)
		if err := postCallback(context.Background(), callbackClient, callbackUrl, cfg.BackendAPIToken, payload, callbackBackoff); err != nil {
			// The backend sweeper fails the submission and refunds the try.
			log.WithFields(fields).WithError(err).Error("Callback failed, submission left pending")
			return
		}

		log.WithFields(fields).Info("Finished task")
	}

	worker.SetPostTaskHandler(postTaskHandler)
	worker.SetErrorHandler(errorHandler)
	worker.SetPreTaskHandler(preTaskHandler)

	log.WithField("worker_count", cfg.WorkerCount).Info("Launching worker...")
	return worker.Launch()
}
