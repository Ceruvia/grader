package machinery

import (
	"bytes"
	"encoding/json"
	"net/http"
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
		submissionId := signature.Args[0].Value.(string)

		// Send taskId to BE's callback endpoint
		callbackUrl := strings.Replace(cfg.BackendCallbackEndpoint, "{submission_id}", submissionId, -1)
		callbackToken := cfg.BackendAPIToken

		payload, err := json.Marshal(struct {
			TaskId string `json:"task_id"`
		}{
			TaskId: signature.UUID,
		})

		r, err := http.NewRequest("POST", callbackUrl, bytes.NewBuffer(payload))
		if err != nil {
			log.WithFields(log.Fields{
				"task":         signature.Name,
				"uuid":         signature.UUID,
				"submissionId": signature.Args[0].Value,
			}).Error("Error while creating callback request: ", err)
		}
		r.Header.Add("Content-Type", "application/json")
		r.Header.Add("X-API-Key", callbackToken)

		client := &http.Client{}
		res, err := client.Do(r)
		if err != nil {
			log.WithFields(log.Fields{
				"task":         signature.Name,
				"uuid":         signature.UUID,
				"submissionId": signature.Args[0].Value,
			}).Error("Error while sending callback request: ", err)
		}

		defer res.Body.Close()

		log.WithFields(log.Fields{
			"task":         signature.Name,
			"uuid":         signature.UUID,
			"submissionId": signature.Args[0].Value,
		}).Info("Finished task")
	}

	worker.SetPostTaskHandler(postTaskHandler)
	worker.SetErrorHandler(errorHandler)
	worker.SetPreTaskHandler(preTaskHandler)

	log.WithField("worker_count", cfg.WorkerCount).Info("Launching worker...")
	return worker.Launch()
}
