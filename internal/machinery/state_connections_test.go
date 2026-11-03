//go:build integration

package machinery

import (
	"errors"
	"io"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
)

// countingProxy forwards TCP to target and counts the connections it accepts.
func countingProxy(t *testing.T, target string) (addr string, accepted *atomic.Int64) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	accepted = new(atomic.Int64)
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			accepted.Add(1)
			go func() {
				defer client.Close()
				server, err := net.Dial("tcp", target)
				if err != nil {
					return
				}
				defer server.Close()
				go io.Copy(server, client)
				io.Copy(client, server)
			}()
		}
	}()
	return ln.Addr().String(), accepted
}

// Each AMQP state write opens its own connection. A graded job must cost one
// (its SUCCESS or FAILURE), and that final state must still reach the Backend.
func TestWorkerWritesOnlyTheFinalStatePerJob(t *testing.T) {
	brokerURL := os.Getenv("GRADER_TEST_AMQP_URL")
	if brokerURL == "" {
		t.Skip("GRADER_TEST_AMQP_URL not set")
	}
	broker, err := url.Parse(brokerURL)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, state string
		task        func() (string, error)
	}{
		{"success", tasks.StateSuccess, func() (string, error) { return "graded", nil }},
		{"failure", tasks.StateFailure, func() (string, error) { return "", errors.New("grading failed") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := strconv.FormatInt(time.Now().UnixNano(), 36)
			proxy, accepted := countingProxy(t, broker.Host)
			result := *broker
			result.Host = proxy
			server, err := startServer(&ceruviaConfig.ServerConfig{
				WorkerCount: 1,
				MQCfg: &ceruviaConfig.MessageQueueConfig{
					BrokerURL:        brokerURL,
					ResultBackendURL: result.String(),
					QueueName:        "state-test-" + id,
					ResultsExpireIn:  60,
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := server.RegisterTask("probe", tc.task); err != nil {
				t.Fatal(err)
			}
			signature := &tasks.Signature{UUID: "task_" + id, Name: "probe"}
			_ = server.NewWorker("probe", 1).Process(signature)
			if got := accepted.Load(); got != 1 {
				t.Fatalf("one job opened %d result backend connections, want 1", got)
			}
			state, err := server.GetBackend().GetState(signature.UUID)
			if err != nil {
				t.Fatal(err)
			}
			if state.State != tc.state || (tc.state == tasks.StateFailure && !strings.Contains(state.Error, "grading failed")) {
				t.Fatalf("Backend reads %+v, want %s", state, tc.state)
			}
		})
	}
}
