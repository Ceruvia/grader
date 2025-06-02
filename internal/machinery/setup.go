package machinery

import (
	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/log"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/RichardKnop/machinery/v2"
	amqpbackend "github.com/RichardKnop/machinery/v2/backends/amqp"
	amqpbroker "github.com/RichardKnop/machinery/v2/brokers/amqp"
	"github.com/RichardKnop/machinery/v2/config"
	eagerlock "github.com/RichardKnop/machinery/v2/locks/eager"
)

func startServer(cfg *ceruviaConfig.MessageQueueConfig) (*machinery.Server, error) {
	cnf := &config.Config{
		Broker:          cfg.BrokerURL,
		DefaultQueue:    cfg.QueueName,
		ResultBackend:   cfg.ResultBackendURL,
		ResultsExpireIn: cfg.ResultsExpireIn,
		AMQP: &config.AMQPConfig{
			Exchange:      "machinery_exchange",
			ExchangeType:  "direct",
			BindingKey:    "machinery_task",
			PrefetchCount: 3,
		},
	}

	broker := amqpbroker.New(cnf)
	backend := amqpbackend.New(cnf)
	lock := eagerlock.New()
	server := machinery.NewServer(cnf, broker, backend, lock)

	tasksMap := map[string]interface{}{
		// TODO: add tasks later
	}

	return server, server.RegisterTasks(tasksMap)
}

func LaunchWorker(cfg *ceruviaConfig.MessageQueueConfig) error {
	server, err := startServer(cfg)
	if err != nil {
		return err
	}

	worker := server.NewWorker("ceruvia_worker", cfg.NumOfWorkers)

	errorHandler := func(err error) {
		log.ERROR.Println("I am an error handler:", err)
	}

	preTaskHandler := func(signature *tasks.Signature) {
		log.INFO.Println("I am a start of task handler for:", signature.Name)
	}

	postTaskHandler := func(signature *tasks.Signature) {
		log.INFO.Println("I am an end of task handler for:", signature.Name)
	}

	worker.SetPostTaskHandler(postTaskHandler)
	worker.SetErrorHandler(errorHandler)
	worker.SetPreTaskHandler(preTaskHandler)

	return worker.Launch()
}
