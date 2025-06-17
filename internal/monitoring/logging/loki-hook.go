package logging

import (
	"github.com/Ceruvia/grader/internal/config"
	"github.com/sirupsen/logrus"
	"github.com/yukitsune/lokirus"
)

func LokiHook(cfg *config.ServerConfig) *lokirus.LokiHook {
	opts := lokirus.NewLokiHookOptions().
		WithLevelMap(lokirus.LevelMap{logrus.PanicLevel: "critical"}).
		WithFormatter(&logrus.JSONFormatter{}).
		WithStaticLabels(lokirus.Labels{
			"grader_app":         cfg.GraderName,
			"grader_environment": cfg.GraderName,
		})

	return lokirus.NewLokiHookWithOpts(
		cfg.MonitoringCfg.LokiURL,
		opts,
		logrus.InfoLevel,
		logrus.WarnLevel,
		logrus.ErrorLevel,
		logrus.FatalLevel)
}
