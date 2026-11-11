package machinery

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"

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
	// stop marks each job whose callback got no final answer unanswered, so its
	// broker redelivers it. Once the stopping grader released its running jobs,
	// their results are dropped: a run the stop cut short must not become a verdict.
	stop *stopState
	// capFailures drops a job after maxCallbackFailures failed callbacks. AMQP
	// only: a requeued message keeps its place, so a job whose callback always
	// fails would starve every job behind it. A Pub/Sub nack does not block the
	// other messages, so a job is redelivered until the Backend answers, also
	// through a Backend deploy of several minutes.
	capFailures bool
	// failures counts the failed callbacks of each job by signature UUID.
	// shortcut: a count that never reaches the cap stays in memory until restart, fine for one exam day.
	failures sync.Map
}

// maxCallbackFailures is how many failed callbacks a job gets before it is
// acked anyway and its submission left pending, which the Backend marks stuck.
const maxCallbackFailures = 5

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
	if b.stop.released.Err() != nil {
		b.stop.unanswered.Store(signature, struct{}{})
		log.WithFields(fields).Warn("Grader stopping, result dropped, job released for redelivery")
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
	// A release cancels a callback in flight, so a stop never waits on its retries.
	err = postCallback(b.stop.released, callbackClient, callbackUrl, b.cfg.BackendAPIToken, payload, callbackBackoff)
	switch {
	case err == nil:
		b.failures.Delete(signature.UUID)
		log.WithFields(fields).Info("Finished task")
	case errors.Is(err, errCallbackRejected):
		b.failures.Delete(signature.UUID)
		log.WithFields(fields).WithError(err).Error("Backend rejected the result")
	case b.stop.released.Err() != nil:
		b.stop.unanswered.Store(signature, struct{}{})
		log.WithFields(fields).WithError(err).Warn("Grader stopping, callback cancelled, job released for redelivery")
	case !b.capFailures:
		b.stop.unanswered.Store(signature, struct{}{})
		log.WithFields(fields).WithError(err).Error("Callback failed, job released for redelivery")
	default:
		count, _ := b.failures.LoadOrStore(signature.UUID, new(atomic.Int32))
		if n := count.(*atomic.Int32).Add(1); n >= maxCallbackFailures {
			b.failures.Delete(signature.UUID)
			log.WithFields(fields).WithError(err).Errorf("Callback failed %d times, job dropped, submission left pending", n)
			return
		}
		b.stop.unanswered.Store(signature, struct{}{})
		log.WithFields(fields).WithError(err).Error("Callback failed, job released for redelivery")
	}
}
