package machinery

import (
	"context"
	"encoding/json"
	"strings"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

func newPostTaskHandler(cfg *ceruviaConfig.ServerConfig) func(*tasks.Signature) {
	return func(signature *tasks.Signature) {
		fields := log.Fields{"task": signature.Name, "uuid": signature.UUID}
		defer func() {
			if r := recover(); r != nil {
				log.WithFields(fields).Errorf("Recovered panic in post task handler: %v", r)
			}
		}()

		// Compile-only checks return through the result backend. They have no
		// submission row or grade, so they must never call the grading callback.
		if signature.Name == "compile_check" || signature.Name == "compile_check_with_builder" {
			return
		}
		if len(signature.Args) == 0 {
			log.WithFields(fields).Error("Task has no submission id, callback skipped")
			return
		}
		submissionId, ok := signature.Args[0].Value.(string)
		if !ok {
			log.WithFields(fields).Errorf("Task submission id has type %T, callback skipped", signature.Args[0].Value)
			return
		}
		fields["submission_id"] = submissionId

		payload, err := json.Marshal(struct {
			TaskId string `json:"task_id"`
		}{TaskId: signature.UUID})
		if err != nil {
			log.WithFields(fields).WithError(err).Error("Failed to encode callback payload")
			return
		}

		callbackUrl := strings.ReplaceAll(cfg.BackendCallbackEndpoint, "{submission_id}", submissionId)
		if err := postCallback(context.Background(), callbackClient, callbackUrl, cfg.BackendAPIToken, payload, callbackBackoff); err != nil {
			// The backend sweeper marks a stuck submission until a late result arrives.
			log.WithFields(fields).WithError(err).Error("Callback failed, submission left pending")
			return
		}
		log.WithFields(fields).Info("Finished task")
	}
}
