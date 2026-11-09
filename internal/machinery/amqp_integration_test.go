//go:build integration

package machinery

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
)

// On AMQP, as on Pub/Sub, a message is acked only after the Backend gave a final
// answer. A 5xx after every retry, or a job still running at shutdown, goes back
// to the queue. The stop waits for a released job to end, and cancels a
// callback still retrying.
func TestAMQPAcksOnlyAfterFinalAnswer(t *testing.T) {
	url := os.Getenv("GRADER_TEST_AMQP_URL")
	if url == "" {
		t.Skip("GRADER_TEST_AMQP_URL not set")
	}
	for _, tc := range []struct {
		name         string
		status       int
		blocks       bool
		wantReleased bool
		runs         time.Duration // a job that ends on its own after the release
		retryWait    time.Duration // the callback's wait before each retry
	}{
		{"2xx acks", http.StatusOK, false, false, 0, 0},
		{"4xx acks", http.StatusNotFound, false, false, 0, 0},
		{"5xx releases", http.StatusServiceUnavailable, false, true, 0, 0},
		{"running job at shutdown releases", http.StatusOK, true, true, 0, 0},
		{"stop waits for the released job", http.StatusOK, false, true, 3 * shutdownGrace / 2, 0},
		{"stop cancels a retrying callback", http.StatusServiceUnavailable, false, true, 0, time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ch, queue := newQueue(t, url)
			callbackURL, callbacks := backendStub(t, tc.status)
			started := make(chan struct{}, 1)
			gate := make(chan struct{})
			t.Cleanup(func() { close(gate) })
			if tc.retryWait > 0 {
				saved := callbackBackoff
				callbackBackoff = []time.Duration{tc.retryWait, tc.retryWait, tc.retryWait}
				t.Cleanup(func() { callbackBackoff = saved })
			}
			var ended atomic.Bool
			stop := launchGrader(t, &ceruviaConfig.MessageQueueConfig{
				Broker: ceruviaConfig.BrokerAMQP, BrokerURL: url, QueueName: queue,
			}, 1, "blackbox", func(string) (string, error) {
				select {
				case started <- struct{}{}:
				default: // redeliveries after the first start
				}
				if tc.blocks {
					<-gate
				}
				time.Sleep(tc.runs)
				ended.Store(true)
				return "{}", nil
			}, callbackURL)

			sig, err := tasks.NewSignature("blackbox", []tasks.Arg{{Type: "string", Value: "sub-1"}})
			if err != nil {
				t.Fatal(err)
			}
			body, _ := json.Marshal(sig)
			if err := ch.PublishWithContext(context.Background(), "", queue, false, false,
				amqp.Publishing{ContentType: "application/json", Body: body}); err != nil {
				t.Fatal(err)
			}
			receive(t, started, "job start")
			if !tc.blocks && tc.runs == 0 {
				receive(t, callbacks, "callback")
			}
			stop()
			if tc.runs > 0 && !ended.Load() {
				t.Fatal("stop returned while the released job still ran")
			}
			if tc.retryWait > 0 {
				select {
				case <-callbacks:
					t.Fatal("callback retried after the release")
				case <-time.After(2 * tc.retryWait):
				}
			}

			q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			if want := map[bool]int{true: 1, false: 0}[tc.wantReleased]; q.Messages != want {
				t.Fatalf("queue holds %d ready messages after shutdown, want %d", q.Messages, want)
			}
		})
	}
}

// newQueue declares a fresh durable queue, as Machinery declares it.
func newQueue(t *testing.T, url string) (*amqp.Channel, string) {
	t.Helper()
	conn, err := amqp.Dial(url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	ch, err := conn.Channel()
	if err != nil {
		t.Fatal(err)
	}
	queue := "grader-test-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = ch.QueueDelete(queue, false, false, false) })
	return ch, queue
}
