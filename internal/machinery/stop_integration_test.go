//go:build integration

package machinery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
)

// A stop always ends within shutdownGrace plus releaseGrace, even when a
// released job never ends and, on Pub/Sub, its nack hangs. Docker kills the
// grader 10 s after the stop signal.
func TestStopIsBounded(t *testing.T) {
	savedShutdown, savedRelease := shutdownGrace, releaseGrace
	shutdownGrace, releaseGrace = time.Second, 200*time.Millisecond
	t.Cleanup(func() { shutdownGrace, releaseGrace = savedShutdown, savedRelease })
	limit := shutdownGrace + releaseGrace*5/4 + time.Second // waitStopped bound plus 1 s to return

	run := func(t *testing.T, start func(fn any, callbackURL string) func(), publish func()) {
		callbackURL, _ := backendStub(t, http.StatusOK)
		started := make(chan struct{}, 1)
		gate := make(chan struct{})
		t.Cleanup(func() { close(gate) })
		stop := start(func(string) (string, error) {
			select {
			case started <- struct{}{}:
			default:
			}
			<-gate // a run that never ends
			return "{}", nil
		}, callbackURL)
		publish()
		receive(t, started, "job start")
		begin := time.Now()
		stop()
		if took := time.Since(begin); took > limit {
			t.Fatalf("stop took %s, want under %s", took, limit)
		}
	}

	t.Run("amqp", func(t *testing.T) {
		url := requireAMQP(t)
		ch, queue := newQueue(t, url)
		run(t, func(fn any, callbackURL string) func() {
			return launchGrader(t, &ceruviaConfig.MessageQueueConfig{
				Broker: ceruviaConfig.BrokerAMQP, BrokerURL: url, QueueName: queue,
			}, 1, "blackbox", fn, callbackURL)
		}, func() { amqpPublish(t, ch, queue, "sub-1") })
	})
	t.Run("pubsub", func(t *testing.T) {
		client := requireEmulator(t)
		topic, sub := newSubscription(t, client, 600)
		faults := &rpcFaults{hangNacks: true}
		run(t, func(fn any, callbackURL string) func() {
			return startFaultyGrader(t, faults, sub, fn, callbackURL)
		}, func() { publish(t, client, topic, "blackbox") })
	})
}

// On AMQP a job whose callback always fails is acked after maxCallbackFailures
// failures and its submission left pending. Requeued forever, it would starve
// the jobs behind it: RabbitMQ requeues a nacked message at its old place.
func TestPoisonJobIsDropped(t *testing.T) {
	url := requireAMQP(t)
	ch, queue := newQueue(t, url)
	callbackURL, poisonPosts, good := poisonStub(t, "poison")
	stop := launchGrader(t, &ceruviaConfig.MessageQueueConfig{
		Broker: ceruviaConfig.BrokerAMQP, BrokerURL: url, QueueName: queue,
	}, 1, "blackbox", func(string) (string, error) { return "{}", nil }, callbackURL)
	amqpPublish(t, ch, queue, "poison")
	amqpPublish(t, ch, queue, "good")
	// One worker with prefetch 1, and a requeued poison job goes back ahead of
	// the good one. So once the good job is answered, the poison job was settled
	// for good: no later redelivery can add a post.
	receive(t, good, "callback of the job behind the poison job")
	want := maxCallbackFailures * (len(callbackBackoff) + 1)
	if got := poisonPosts(); got != want {
		t.Fatalf("poison job posted %d callbacks, want %d (%d failures)", got, want, maxCallbackFailures)
	}
	stop()
	q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if q.Messages != 0 {
		t.Fatalf("queue holds %d messages after the poison job was dropped, want 0", q.Messages)
	}
}

// On Pub/Sub a job whose callback keeps failing is released again after
// maxCallbackFailures failures, not dropped: a nack does not block the other
// messages, and the job must survive a Backend deploy.
func TestPubSubFailingCallbackIsNotDropped(t *testing.T) {
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 10)
	callbackURL, poisonPosts, _ := poisonStub(t, "poison")
	stop := startGrader(t, sub, 1, "blackbox", func(string) (string, error) { return "{}", nil }, callbackURL)
	publishFor(t, client, topic, "blackbox", "poison")
	// A post after the cap is a redelivery of a job the cap would have dropped.
	beyondCap := maxCallbackFailures*(len(callbackBackoff)+1) + 1
	for deadline := time.Now().Add(waitLimit); poisonPosts() < beyondCap; time.Sleep(10 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("poison job posted %d callbacks within %s, want it redelivered past %d failures",
				poisonPosts(), waitLimit, maxCallbackFailures)
		}
	}
	stop()
}

// poisonStub answers 500 to every callback for poison and 200 to the others,
// whose task ids it sends on good. poisonPosts counts the callbacks for poison.
func poisonStub(t *testing.T, poison string) (url string, poisonPosts func() int, good <-chan string) {
	t.Helper()
	var mu sync.Mutex
	count := 0
	ids := make(chan string, 100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/"+poison) {
			mu.Lock()
			count++
			mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ids <- r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	return server.URL, func() int { mu.Lock(); defer mu.Unlock(); return count }, ids
}
