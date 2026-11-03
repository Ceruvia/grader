package machinery

import (
	"sync"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	ceruviaTasks "github.com/Ceruvia/grader/internal/tasks"
	"github.com/RichardKnop/machinery/v2"
	nullbackend "github.com/RichardKnop/machinery/v2/backends/null"
	amqpbroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	brokersiface "github.com/RichardKnop/machinery/v2/brokers/iface"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

// GradingTasks are the task names the Backend sends.
var GradingTasks = map[string]interface{}{
	"blackbox":                   ceruviaTasks.GradeBlackbox,
	"blackbox_with_builder":      ceruviaTasks.GradeBlackboxWithBuilder,
	"compile_check":              ceruviaTasks.CompileCheck,
	"compile_check_with_builder": ceruviaTasks.CompileCheckWithBuilder,
}

// NewServer connects to the broker QUEUE_BROKER names and registers taskFuncs.
func NewServer(cfg *ceruviaConfig.ServerConfig, taskFuncs map[string]interface{}) (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:       cfg.MQCfg.BrokerURL,
		DefaultQueue: cfg.MQCfg.QueueName,
		AMQP: &config.AMQPConfig{
			Exchange:      "machinery_exchange",
			ExchangeType:  "direct",
			BindingKey:    "machinery_task",
			PrefetchCount: cfg.WorkerCount,
		},
	}

	var broker brokersiface.Broker
	var unanswered *sync.Map
	if cfg.MQCfg.Broker == ceruviaConfig.BrokerPubSub {
		unanswered = new(sync.Map)
		var err error
		if broker, err = newPubSubBroker(cnf, cfg.MQCfg, unanswered); err != nil {
			return nil, err
		}
	} else {
		broker = amqpbroker.New(cnf)
	}
	backend := &callbackBackend{Backend: nullbackend.New(), cfg: cfg, unanswered: unanswered}
	server := machinery.NewServer(cnf, broker, backend, eagerlock.New())

	if err := server.RegisterTasks(taskFuncs); err != nil {
		log.WithError(err).Error("Failed to register tasks")
		return nil, err
	}
	return server, nil
}

func newWorker(server *machinery.Server, cfg *ceruviaConfig.ServerConfig) *machinery.Worker {
	worker := server.NewWorker("ceruvia_worker", cfg.WorkerCount)
	worker.SetErrorHandler(func(err error) {
		log.WithError(err).Error("Error occurred while processing task")
	})
	worker.SetPreTaskHandler(func(signature *tasks.Signature) {
		fields := log.Fields{"task": signature.Name, "uuid": signature.UUID}
		// A signature without args must not panic the worker before its callback.
		if len(signature.Args) > 0 {
			fields["submissionId"] = signature.Args[0].Value
		}
		log.WithFields(fields).Info("Starting task")
	})
	return worker
}

func LaunchWorker(cfg *ceruviaConfig.ServerConfig) error {
	server, err := NewServer(cfg, GradingTasks)
	if err != nil {
		log.WithError(err).Error("Failed to start Machinery server")
		return err
	}
	log.WithFields(log.Fields{"worker_count": cfg.WorkerCount, "broker": cfg.MQCfg.Broker}).Info("Launching worker...")
	return newWorker(server, cfg).Launch()
}
