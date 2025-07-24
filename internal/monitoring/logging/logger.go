package logging

import (
	"os"

	"github.com/Ceruvia/grader/internal/config"
	machineryLog "github.com/RichardKnop/machinery/v2/log"
	log "github.com/sirupsen/logrus"
)

var (
	EnableLoki bool = false
	srvConfig  *config.ServerConfig
)

func InitLogger(cfg *config.ServerConfig) {
	if cfg.MonitoringCfg.LokiURL != "" {
		EnableLoki = true
	}
	srvConfig = cfg
}

func RunLogger() {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00", // ISO 8601
		PrettyPrint:     false,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	if srvConfig.GraderEnv != "production" {
		log.SetLevel(log.DebugLevel)
	}

	if EnableLoki {
		log.AddHook(&GraderNameHook{GraderName: srvConfig.GraderName})
		log.AddHook(LokiHook(srvConfig))

		log.Info("Logger connected to Loki")
	} else {
		log.Warn("Logger NOT connected to Loki")
	}

	machineryLog.Set(log.StandardLogger())
}
