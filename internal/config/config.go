package config

import (
	"log"

	"github.com/Ceruvia/grader/internal/helper/env"
	"github.com/joho/godotenv"
)

type MessageQueueConfig struct {
	BrokerURL        string
	ResultBackendURL string
	QueueName        string
	ResultsExpireIn  int
}

type ServerConfig struct {
	GraderName  string
	GraderEnv   string
	WorkerCount int
	MQCfg       *MessageQueueConfig
}

func loadEnvFile() {
	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file")
	}
}

func GetAppConfig() *ServerConfig {
	loadDotEnv := env.GetBool("LOAD_DOTENV", true)

	if loadDotEnv {
		loadEnvFile()
	}

	graderName := env.GetString("GRADER_NAME", "grader")
	graderEnv := env.GetString("GRADER_ENV", "development")
	graderWorkerCount := env.GetInt("GRADER_WORKER_COUNT", 20)

	mqConfig := &MessageQueueConfig{
		BrokerURL:        env.GetString("QUEUE_BROKER_URL", "amqp://guest:guest@localhost:5672/"),
		ResultBackendURL: env.GetString("QUEUE_RESULT_URL", "amqp://guest:guest@localhost:5672/"),
		QueueName:        env.GetString("QUEUE_NAME", "ceruvia_submissions"),
		ResultsExpireIn:  env.GetInt("QUEUE_RESULT_TTL", 36000),
	}

	return &ServerConfig{
		GraderName:  graderName,
		GraderEnv:   graderEnv,
		WorkerCount: graderWorkerCount,
		MQCfg:       mqConfig,
	}
}
