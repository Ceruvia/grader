package machinery

import (
	"sync"
	"time"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/sandboxes"
	ceruviaTasks "github.com/Ceruvia/grader/internal/tasks"
	"github.com/RichardKnop/machinery/v2"
	nullbackend "github.com/RichardKnop/machinery/v2/backends/null"
	brokersiface "github.com/RichardKnop/machinery/v2/brokers/iface"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

// shutdownGrace is how long a stopping grader waits for running jobs before
// it releases their messages for redelivery. It stays below Docker's
// default 10 s stop timeout so the release happens before the kill.
var shutdownGrace = 5 * time.Second

// stopWithGrace waits shutdownGrace for the consumer to report stopped, that
// is for its running jobs to end. Then it releases the jobs still running:
// closing release drops their results and returns their messages to the
// broker, and StopRuns kills their boxes. release closes first, so a run the
// kill cut short can never be reported.
func stopWithGrace(stopped <-chan struct{}, release chan struct{}) {
	select {
	case <-stopped:
	case <-time.After(shutdownGrace):
		close(release)
		sandboxes.StopRuns()
		<-stopped
	}
}

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

	// release closes when a stopping grader gives up its running jobs (see stopWithGrace).
	release := make(chan struct{})
	var broker brokersiface.Broker
	var unanswered *sync.Map
	if cfg.MQCfg.Broker == ceruviaConfig.BrokerPubSub {
		unanswered = new(sync.Map)
		var err error
		if broker, err = newPubSubBroker(cnf, cfg.MQCfg, unanswered, release); err != nil {
			return nil, err
		}
	} else {
		broker = newAMQPBroker(cnf, release)
	}
	backend := &callbackBackend{Backend: nullbackend.New(), cfg: cfg, unanswered: unanswered, release: release}
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
