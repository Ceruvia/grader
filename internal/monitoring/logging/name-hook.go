package logging

import (
	log "github.com/sirupsen/logrus"
)

type GraderNameHook struct {
	GraderName string
}

func (h *GraderNameHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *GraderNameHook) Fire(entry *log.Entry) error {
	entry.Data["grader_name"] = h.GraderName
	return nil
}
