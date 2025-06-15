package logging

import (
	"os"

	"github.com/Ceruvia/grader/internal/config"
	log "github.com/sirupsen/logrus"
	machineryLog "github.com/RichardKnop/machinery/v2/log"

)

func InitLogger(cfg *config.ServerConfig) {
	log.SetFormatter(&log.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05Z07:00", // ISO 8601
		PrettyPrint:     false,
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	if cfg.GraderEnv == "development" {
		log.SetLevel(log.DebugLevel)
	}

	// Optional: add hooks here (e.g. for Sentry)
	machineryLog.Set(log.StandardLogger())
}
