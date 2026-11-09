package machinery

import (
	"context"
	"sync"
	"sync/atomic"
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

// stopState is the shutdown state a broker shares with the callback backend.
type stopState struct {
	// released ends when the stopping grader gives up its running jobs: their
	// results are dropped and their messages go back to the broker.
	released context.Context
	release  context.CancelFunc
	// unanswered holds each signature whose callback got no final answer, so
	// its message is nacked for redelivery instead of acked.
	unanswered sync.Map
	// stopping replaces Machinery's retry flag, which StopConsuming writes unsynchronized.
	stopping atomic.Bool
	// stopped closes when StartConsuming returned for good.
	stopped chan struct{}
}

func newStopState() *stopState {
	released, release := context.WithCancel(context.Background())
	return &stopState{released: released, release: release, stopped: make(chan struct{})}
}

// answered reports whether the callback of signature got a final answer.
func (s *stopState) answered(signature *tasks.Signature) bool {
	_, unanswered := s.unanswered.LoadAndDelete(signature)
	return !unanswered
}

// waitStopped waits shutdownGrace for the consumer to report stopped, that
// is for its running jobs to end. Then it releases the jobs still running:
// their results are dropped, their messages go back to the broker and their
// boxes are killed (sandboxes.BoundRuns).
func (s *stopState) waitStopped() {
	select {
	case <-s.stopped:
	case <-time.After(shutdownGrace):
		s.release()
		<-s.stopped
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

	stop := newStopState()
	sandboxes.BoundRuns(stop.released)
	var broker brokersiface.Broker
	if cfg.MQCfg.Broker == ceruviaConfig.BrokerPubSub {
		var err error
		if broker, err = newPubSubBroker(cnf, cfg.MQCfg, stop); err != nil {
			return nil, err
		}
	} else {
		broker = newAMQPBroker(cnf, stop)
	}
	backend := &callbackBackend{Backend: nullbackend.New(), cfg: cfg, stop: stop}
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
