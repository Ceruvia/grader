package machinery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
)

func TestPostTaskHandlerSkipsCompileCheck(t *testing.T) {
	for _, tc := range []struct {
		name string
		want int64
	}{
		{"compile_check", 0},
		{"compile_check_with_builder", 0},
		{"blackbox", 1},
		{"blackbox_with_builder", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/submissions/compile-only" {
					t.Errorf("unexpected callback %s %s", r.Method, r.URL.Path)
				}
				var payload struct {
					TaskID string `json:"task_id"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil || payload.TaskID != "task-id" {
					t.Errorf("invalid callback body: %+v, %v", payload, err)
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			handler := newPostTaskHandler(&ceruviaConfig.ServerConfig{
				BackendCallbackEndpoint: server.URL + "/submissions/{submission_id}",
			})
			handler(&tasks.Signature{Name: tc.name, UUID: "task-id", Args: []tasks.Arg{{Type: "string", Value: "compile-only"}}})
			if got := requests.Load(); got != tc.want {
				t.Fatalf("%s made %d HTTP callbacks, want %d", tc.name, got, tc.want)
			}
		})
	}
}
