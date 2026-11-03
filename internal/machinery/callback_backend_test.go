package machinery

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
)

// The callback body is the contract agreed with the Backend (3 Nov 2026): the
// final state and the task's string result travel in it, for graded
// submissions and compile checks alike.
func TestCallbackCarriesFinalState(t *testing.T) {
	const result = `{"status":"compiled","message":""}`
	succeed := func(string) (string, error) { return result, nil }
	fail := func(string) (string, error) { return "", errors.New("download failed") }

	for _, tc := range []struct {
		task string
		fn   func(string) (string, error)
		want map[string]any
	}{
		{"blackbox", succeed, map[string]any{"task_id": "task_1", "kind": "grade", "state": "SUCCESS", "result": result}},
		{"blackbox_with_builder", fail, map[string]any{"task_id": "task_1", "kind": "grade", "state": "FAILURE", "error": "download failed"}},
		{"compile_check", succeed, map[string]any{"task_id": "task_1", "kind": "compile_check", "state": "SUCCESS", "result": result}},
		{"compile_check_with_builder", fail, map[string]any{"task_id": "task_1", "kind": "compile_check", "state": "FAILURE", "error": "download failed"}},
	} {
		t.Run(tc.task, func(t *testing.T) {
			bodies := make(chan map[string]any, 4)
			backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				if r.URL.Path != "/submissions/sub-1/callback" || r.Header.Get("X-API-Key") != "token" ||
					json.NewDecoder(r.Body).Decode(&body) != nil {
					t.Errorf("unexpected callback %s %s", r.Method, r.URL.Path)
				}
				bodies <- body
			}))
			defer backend.Close()

			cfg := &ceruviaConfig.ServerConfig{
				WorkerCount:             1,
				BackendAPIToken:         "token",
				BackendCallbackEndpoint: backend.URL + "/submissions/{submission_id}/callback",
				MQCfg:                   &ceruviaConfig.MessageQueueConfig{Broker: ceruviaConfig.BrokerAMQP},
			}
			server, err := NewServer(cfg, map[string]interface{}{tc.task: tc.fn})
			if err != nil {
				t.Fatal(err)
			}
			_ = newWorker(server, cfg).Process(&tasks.Signature{
				UUID: "task_1", Name: tc.task, Args: []tasks.Arg{{Type: "string", Value: "sub-1"}},
			})

			if len(bodies) != 1 {
				t.Fatalf("got %d callbacks, want 1", len(bodies))
			}
			if got := <-bodies; !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("callback body %v, want %v", got, tc.want)
			}
		})
	}
}
