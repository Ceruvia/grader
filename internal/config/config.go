package config

import (
	"log"

	"github.com/Ceruvia/grader/internal/helper/env"
	"github.com/joho/godotenv"
)

type MessageQueueConfig struct {
	GCPProjectID     string
	SubscriptionName string
	TopicName        string
}

type CallbackConfig struct {
	BaseURL string
	APIKey  string
}

type MonitoringConfig struct {
	MetricsEnabled     bool
	LoggingEnabled     bool
	LokiURL            string
	InfluxURL          string
	InfluxToken        string
	InfluxOrganization string
	InfluxBucket       string
}

type ServerConfig struct {
	GraderName    string
	GraderEnv     string
	WorkerCount   int
	MQCfg         *MessageQueueConfig
	CallbackCfg   *CallbackConfig
	MonitoringCfg *MonitoringConfig
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
		GCPProjectID:     env.GetString("GCP_PROJECT_ID", ""),
		SubscriptionName: env.GetString("PUBSUB_SUBSCRIPTION_NAME", ""),
		TopicName:        env.GetString("PUBSUB_TOPIC_NAME", ""),
	}

	callbackConfig := &CallbackConfig{
		BaseURL: env.GetString("CALLBACK_BASE_URL", ""),
		APIKey:  env.GetString("CALLBACK_API_KEY", ""),
	}

	monitoringConfig := &MonitoringConfig{
		MetricsEnabled:     env.GetBool("METRICS_ENABLED", false),
		LoggingEnabled:     env.GetBool("LOGGING_ENABLED", false),
		LokiURL:            env.GetString("MONITORING_LOKI_URL", ""),
		InfluxURL:          env.GetString("MONITORING_INFLUX_URL", ""),
		InfluxToken:        env.GetString("MONITORING_INFLUX_TOKEN", ""),
		InfluxOrganization: env.GetString("MONITORING_INFLUX_ORGANIZATION", ""),
		InfluxBucket:       env.GetString("MONITORING_INFLUX_BUCKET", ""),
	}

	return &ServerConfig{
		GraderName:    graderName,
		GraderEnv:     graderEnv,
		WorkerCount:   graderWorkerCount,
		MQCfg:         mqConfig,
		CallbackCfg:   callbackConfig,
		MonitoringCfg: monitoringConfig,
	}
}
