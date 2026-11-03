package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/Ceruvia/grader/internal/helper/env"
	"github.com/joho/godotenv"
)

// Brokers selectable with QUEUE_BROKER.
const (
	BrokerAMQP   = "amqp"
	BrokerPubSub = "gcppubsub"
)

type MessageQueueConfig struct {
	Broker string

	// AMQP (RabbitMQ)
	BrokerURL string
	QueueName string

	// GCP Pub/Sub. TopicName is used only by cmd/send. The client reads
	// GOOGLE_APPLICATION_CREDENTIALS and PUBSUB_EMULATOR_HOST itself.
	ProjectID        string
	TopicName        string
	SubscriptionName string
}

type MonitoringConfig struct {
	LokiURL            string
	InfluxURL          string
	InfluxToken        string
	InfluxOrganization string
	InfluxBucket       string
}

type ServerConfig struct {
	GraderName  string
	GraderEnv   string
	WorkerCount int

	BackendAPIToken         string
	BackendCallbackEndpoint string

	MQCfg         *MessageQueueConfig
	MonitoringCfg *MonitoringConfig
}

func GetAppConfig() (*ServerConfig, error) {
	loadDotEnv := false
	if value, exists := os.LookupEnv("LOAD_DOTENV"); exists {
		var err error
		loadDotEnv, err = strconv.ParseBool(value)
		if err != nil {
			return nil, errors.New("LOAD_DOTENV must be a boolean")
		}
	}
	if loadDotEnv {
		if err := godotenv.Load(); err != nil {
			return nil, errors.New("cannot read .env with LOAD_DOTENV enabled")
		}
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "grader"
	}

	graderName := env.GetString("GRADER_NAME", hostname)
	graderEnv := env.GetString("GRADER_ENV", "production")
	if graderEnv != "production" && graderEnv != "development" {
		return nil, errors.New("GRADER_ENV must be production or development")
	}
	graderWorkerCount := 3
	if value, exists := os.LookupEnv("GRADER_WORKER_COUNT"); exists {
		var err error
		graderWorkerCount, err = strconv.Atoi(value)
		if err != nil || graderWorkerCount <= 0 {
			return nil, errors.New("GRADER_WORKER_COUNT must be a positive integer")
		}
	}

	backendAPIToken := env.GetString("BACKEND_API_TOKEN", "")
	backendCallbackEndpoint := env.GetString("BACKEND_CALLBACK_ENDPOINT", "")

	mqConfig := &MessageQueueConfig{
		Broker:           env.GetString("QUEUE_BROKER", BrokerAMQP),
		BrokerURL:        env.GetString("QUEUE_BROKER_URL", ""),
		QueueName:        env.GetString("QUEUE_NAME", ""),
		ProjectID:        env.GetString("GCP_PROJECT_ID", ""),
		TopicName:        env.GetString("PUBSUB_TOPIC_NAME", ""),
		SubscriptionName: env.GetString("PUBSUB_SUBSCRIPTION_NAME", ""),
	}
	type setting struct{ name, value string }
	required := []setting{
		{"BACKEND_API_TOKEN", backendAPIToken},
		{"BACKEND_CALLBACK_ENDPOINT", backendCallbackEndpoint},
	}
	switch mqConfig.Broker {
	case BrokerAMQP:
		required = append(required, setting{"QUEUE_BROKER_URL", mqConfig.BrokerURL}, setting{"QUEUE_NAME", mqConfig.QueueName})
	case BrokerPubSub:
		required = append(required, setting{"GCP_PROJECT_ID", mqConfig.ProjectID},
			setting{"PUBSUB_SUBSCRIPTION_NAME", mqConfig.SubscriptionName})
	default:
		return nil, errors.New("QUEUE_BROKER must be amqp or gcppubsub")
	}

	if graderEnv == "production" {
		for _, s := range required {
			if strings.TrimSpace(s.value) == "" {
				return nil, fmt.Errorf("%s is required in production", s.name)
			}
		}
	}
	if backendCallbackEndpoint != "" {
		callback, err := url.Parse(backendCallbackEndpoint)
		if err != nil || callback.Hostname() == "" ||
			(callback.Scheme != "http" && callback.Scheme != "https") ||
			callback.User != nil || callback.Fragment != "" || strings.HasSuffix(callback.Host, ":") {
			return nil, errors.New("BACKEND_CALLBACK_ENDPOINT must be an absolute http or https URL")
		}
		if port := callback.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return nil, errors.New("BACKEND_CALLBACK_ENDPOINT has an invalid port")
			}
		}
	}

	monitoringConfig := &MonitoringConfig{
		LokiURL:            env.GetString("MONITORING_LOKI_URL", ""),
		InfluxURL:          env.GetString("MONITORING_INFLUX_URL", ""),
		InfluxToken:        env.GetString("MONITORING_INFLUX_TOKEN", ""),
		InfluxOrganization: env.GetString("MONITORING_INFLUX_ORGANIZATION", ""),
		InfluxBucket:       env.GetString("MONITORING_INFLUX_BUCKET", ""),
	}

	return &ServerConfig{
		GraderName:              graderName,
		GraderEnv:               graderEnv,
		WorkerCount:             graderWorkerCount,
		BackendAPIToken:         backendAPIToken,
		BackendCallbackEndpoint: backendCallbackEndpoint,
		MQCfg:                   mqConfig,
		MonitoringCfg:           monitoringConfig,
	}, nil
}
