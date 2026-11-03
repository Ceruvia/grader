package machinery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	backendsiface "github.com/RichardKnop/machinery/v2/backends/iface"
	"github.com/RichardKnop/machinery/v2/tasks"
	log "github.com/sirupsen/logrus"
)

// callbackBackend is the Machinery result backend. It stores nothing: the final
// state of each task goes to the Backend in the callback body, and every other
// state is dropped. The embedded backend is Machinery's null backend.
type callbackBackend struct {
	backendsiface.Backend
	cfg *ceruviaConfig.ServerConfig
	// unanswered, when set, receives each signature whose callback got no final
	// answer, so the Pub/Sub broker redelivers its message instead of acking it.
	unanswered *sync.Map
}

// callbackBody is the grade-callback request body, agreed with the Backend.
type callbackBody struct {
	TaskID string `json:"task_id"`
	Kind   string `json:"kind"`
	State  string `json:"state"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

func (b *callbackBackend) SetStateSuccess(signature *tasks.Signature, results []*tasks.TaskResult) error {
	body := callbackBody{State: tasks.StateSuccess}
	if len(results) > 0 {
		body.Result = results[0].Value
	}
	b.post(signature, body)
	return nil
}

func (b *callbackBackend) SetStateFailure(signature *tasks.Signature, taskErr string) error {
	b.post(signature, callbackBody{State: tasks.StateFailure, Error: taskErr})
	return nil
}

func (b *callbackBackend) post(signature *tasks.Signature, body callbackBody) {
	fields := log.Fields{"task": signature.Name, "uuid": signature.UUID, "state": body.State}
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

	body.TaskID = signature.UUID
	body.Kind = "grade"
	if strings.HasPrefix(signature.Name, "compile_check") {
		body.Kind = "compile_check"
	}
	payload, err := json.Marshal(body)
	if err != nil {
		log.WithFields(fields).WithError(err).Error("Failed to encode callback payload")
		return
	}

	callbackUrl := strings.ReplaceAll(b.cfg.BackendCallbackEndpoint, "{submission_id}", submissionId)
	err = postCallback(context.Background(), callbackClient, callbackUrl, b.cfg.BackendAPIToken, payload, callbackBackoff)
	switch {
	case err == nil:
		log.WithFields(fields).Info("Finished task")
	case errors.Is(err, errCallbackRejected):
		log.WithFields(fields).WithError(err).Error("Backend rejected the result")
	case b.unanswered != nil:
		b.unanswered.Store(signature, struct{}{})
		log.WithFields(fields).WithError(err).Error("Callback failed, job released for redelivery")
	default:
		// The backend sweeper marks a stuck submission.
		log.WithFields(fields).WithError(err).Error("Callback failed, submission left pending")
	}
}
