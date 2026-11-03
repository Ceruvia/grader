//go:build integration

package machinery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	ceruviaConfig "github.com/Ceruvia/grader/internal/config"
	"github.com/RichardKnop/machinery/v2/tasks"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	testProject = "grader-test"
	// childEnv names the subscription a child grader process consumes.
	childEnv = "GRADER_TEST_PUBSUB_CHILD"
	// waitLimit bounds every wait for async work.
	waitLimit = 60 * time.Second
)

// TestMain starts one Pub/Sub emulator for the package when GRADER_TEST_PUBSUB=1.
// Each test makes its own topic and subscription. CI does not set the variable,
// because its tests run inside the grader container without Docker.
func TestMain(m *testing.M) {
	shutdownGrace = 200 * time.Millisecond
	lease = 3 * time.Second
	settleRetry = 50 * time.Millisecond
	rpcTimeout = 2 * time.Second
	callbackBackoff = []time.Duration{10 * time.Millisecond, 10 * time.Millisecond, 10 * time.Millisecond}
	os.Exit(runWithEmulator(m))
}

func runWithEmulator(m *testing.M) int {
	if os.Getenv("GRADER_TEST_PUBSUB") != "1" || os.Getenv(childEnv) != "" {
		return m.Run()
	}
	ctx := context.Background()
	ctr, err := testcontainers.Run(ctx, "gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators",
		testcontainers.WithExposedPorts("8085/tcp"),
		testcontainers.WithCmd("gcloud", "beta", "emulators", "pubsub", "start",
			"--host-port=0.0.0.0:8085", "--project="+testProject),
		testcontainers.WithWaitStrategy(wait.ForLog("Server started").WithStartupTimeout(2*time.Minute)),
	)
	if err != nil {
		log.Printf("start Pub/Sub emulator: %v", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(ctr) }()
	endpoint, err := ctr.PortEndpoint(ctx, "8085/tcp", "")
	if err != nil {
		log.Printf("emulator endpoint: %v", err)
		return 1
	}
	os.Setenv("PUBSUB_EMULATOR_HOST", endpoint)
	return m.Run()
}

func requireEmulator(t *testing.T) *pubsub.Client {
	t.Helper()
	if os.Getenv("PUBSUB_EMULATOR_HOST") == "" {
		t.Skip("GRADER_TEST_PUBSUB=1 not set")
	}
	client, err := pubsub.NewClient(context.Background(), testProject)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// newSubscription makes a fresh topic with one subscription and returns both IDs.
func newSubscription(t *testing.T, client *pubsub.Client, ackDeadlineSeconds int32) (topic, sub string) {
	t.Helper()
	ctx := context.Background()
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	topic, sub = "jobs-"+id, "graders-"+id
	topicName := fmt.Sprintf("projects/%s/topics/%s", testProject, topic)
	subName := fmt.Sprintf("projects/%s/subscriptions/%s", testProject, sub)
	if _, err := client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: topicName}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{
		Name: subName, Topic: topicName, AckDeadlineSeconds: ackDeadlineSeconds,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.SubscriptionAdminClient.DeleteSubscription(ctx, &pubsubpb.DeleteSubscriptionRequest{Subscription: subName})
		_ = client.TopicAdminClient.DeleteTopic(ctx, &pubsubpb.DeleteTopicRequest{Topic: topicName})
	})
	return topic, sub
}

// startGrader launches a grader with one task and returns its stop function.
func startGrader(t *testing.T, sub string, workers int, task string, fn any, callbackURL string) func() {
	t.Helper()
	cfg := &ceruviaConfig.ServerConfig{
		WorkerCount:             workers,
		BackendAPIToken:         "token",
		BackendCallbackEndpoint: callbackURL + "/submissions/{submission_id}",
		MQCfg: &ceruviaConfig.MessageQueueConfig{
			Broker: ceruviaConfig.BrokerPubSub, ProjectID: testProject, SubscriptionName: sub,
		},
	}
	server, err := NewServer(cfg, map[string]any{task: fn})
	if err != nil {
		t.Fatal(err)
	}
	server.GetConfig().NoUnixSignals = true
	worker := newWorker(server, cfg)
	worker.LaunchAsync(make(chan error, 2))
	var once sync.Once
	stop := func() { once.Do(worker.Quit) }
	t.Cleanup(stop)
	return stop
}

func publish(t *testing.T, client *pubsub.Client, topic, task string) string {
	t.Helper()
	sig, err := tasks.NewSignature(task, []tasks.Arg{{Type: "string", Value: "sub-1"}})
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(sig)
	publisher := client.Publisher(topic)
	defer publisher.Stop()
	if _, err := publisher.Publish(context.Background(), &pubsub.Message{Data: data}).Get(context.Background()); err != nil {
		t.Fatal(err)
	}
	return sig.UUID
}

// backendStub answers every callback with status and sends each task_id on the channel.
func backendStub(t *testing.T, status int) (string, <-chan string) {
	t.Helper()
	ids := make(chan string, 100)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			TaskID string `json:"task_id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		select {
		case ids <- body.TaskID:
		default: // a redelivery loop must not block the callback
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server.URL, ids
}

func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(waitLimit):
		t.Fatalf("no %s within %s", what, waitLimit)
		panic("unreachable")
	}
}

// pending pulls what the subscription would deliver right now, without leasing it long.
func pending(t *testing.T, client *pubsub.Client, sub string) []string {
	t.Helper()
	res, err := client.SubscriptionAdminClient.Pull(context.Background(), &pubsubpb.PullRequest{
		Subscription:      fmt.Sprintf("projects/%s/subscriptions/%s", testProject, sub),
		MaxMessages:       10,
		ReturnImmediately: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range res.ReceivedMessages {
		var sig tasks.Signature
		_ = json.Unmarshal(m.Message.Data, &sig)
		ids = append(ids, sig.UUID)
	}
	return ids
}

// pollPending pulls until a message arrives or until deadline, and returns what arrived.
func pollPending(t *testing.T, client *pubsub.Client, sub string, deadline time.Time) []string {
	t.Helper()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for time.Now().Before(deadline) {
		if got := pending(t, client, sub); len(got) > 0 {
			return got
		}
		<-tick.C
	}
	return nil
}

// Each grader holds at most GRADER_WORKER_COUNT messages, so two one-slot graders
// run two jobs side by side instead of one grader taking both.
func TestPubSubWorkerSlotsCapDeliveries(t *testing.T) {
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 600)
	callbackURL, callbacks := backendStub(t, http.StatusOK)

	// Both jobs wait in the subscription before any grader pulls, so a grader
	// that takes more than its slots would get both. The long ack deadline keeps
	// a hoarded job from coming back to the other grader within waitLimit.
	publish(t, client, topic, "blackbox")
	publish(t, client, topic, "blackbox")
	gate := make(chan struct{})
	started := make(chan string, 4)
	var overCap atomic.Bool
	for _, name := range []string{"a", "b"} {
		var running atomic.Int32
		startGrader(t, sub, 1, "blackbox", func(string) (string, error) {
			if running.Add(1) > 1 {
				overCap.Store(true)
			}
			defer running.Add(-1)
			started <- name
			<-gate
			return "{}", nil
		}, callbackURL)
	}

	first, second := receive(t, started, "first job start"), receive(t, started, "second job start")
	close(gate)
	receive(t, callbacks, "first callback")
	receive(t, callbacks, "second callback")

	if overCap.Load() {
		t.Fatal("a one-slot grader ran two jobs at once")
	}
	if first == second {
		t.Fatalf("grader %s ran both jobs, the other grader got none", first)
	}
}

// A grader killed mid-job never acks, so the job reaches another grader.
func TestPubSubRedeliversWhenGraderDies(t *testing.T) {
	if sub := os.Getenv(childEnv); sub != "" {
		runDyingGrader(sub)
		return
	}
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 10)

	childStarted := make(chan struct{}, 1)
	childURL := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		childStarted <- struct{}{}
	}))
	t.Cleanup(childURL.Close)
	child := exec.Command(os.Args[0], "-test.run=^TestPubSubRedeliversWhenGraderDies$")
	child.Env = append(os.Environ(), childEnv+"="+sub, "GRADER_TEST_CHILD_URL="+childURL.URL)
	child.Stdout, child.Stderr = os.Stderr, os.Stderr
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = child.Process.Kill(); _ = child.Wait() })

	taskID := publish(t, client, topic, "blackbox")
	receive(t, childStarted, "job start in the child grader")
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()

	callbackURL, callbacks := backendStub(t, http.StatusOK)
	startGrader(t, sub, 1, "blackbox", func(string) (string, error) { return "{}", nil }, callbackURL)
	if got := receive(t, callbacks, "callback after redelivery"); got != taskID {
		t.Fatalf("callback for %s, want %s", got, taskID)
	}
}

// runDyingGrader is the child of TestPubSubRedeliversWhenGraderDies: its job
// reports its start and then blocks until the parent kills the process.
func runDyingGrader(sub string) {
	cfg := &ceruviaConfig.ServerConfig{
		WorkerCount:             1,
		BackendCallbackEndpoint: "http://127.0.0.1:1/{submission_id}",
		MQCfg: &ceruviaConfig.MessageQueueConfig{
			Broker: ceruviaConfig.BrokerPubSub, ProjectID: testProject, SubscriptionName: sub,
		},
	}
	server, err := NewServer(cfg, map[string]any{"blackbox": func(string) (string, error) {
		if res, err := http.Get(os.Getenv("GRADER_TEST_CHILD_URL")); err == nil {
			res.Body.Close()
		}
		select {}
	}})
	if err != nil {
		log.Fatal(err)
	}
	_ = newWorker(server, cfg).Launch()
}

// A message is acked only after the Backend gave a final answer. A 5xx after
// every retry, or a job still running at shutdown, releases it for redelivery.
func TestPubSubAcksOnlyAfterFinalAnswer(t *testing.T) {
	for _, tc := range []struct {
		name         string
		status       int
		blocks       bool
		wantReleased bool
	}{
		{"2xx acks", http.StatusOK, false, false},
		{"4xx acks", http.StatusNotFound, false, false},
		{"5xx releases", http.StatusServiceUnavailable, false, true},
		{"running job at shutdown releases", http.StatusOK, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := requireEmulator(t)
			topic, sub := newSubscription(t, client, 10)
			callbackURL, callbacks := backendStub(t, tc.status)
			started := make(chan struct{}, 1)
			gate := make(chan struct{})
			t.Cleanup(func() { close(gate) })
			stop := startGrader(t, sub, 1, "blackbox", func(string) (string, error) {
				select {
				case started <- struct{}{}:
				default: // redeliveries after the first start
				}
				if tc.blocks {
					<-gate
				}
				return "{}", nil
			}, callbackURL)

			taskID := publish(t, client, topic, "blackbox")
			receive(t, started, "job start")
			if !tc.blocks {
				receive(t, callbacks, "callback")
			}
			stop()

			if tc.wantReleased {
				if got := pollPending(t, client, sub, time.Now().Add(waitLimit)); len(got) != 1 || got[0] != taskID {
					t.Fatalf("subscription holds %v after shutdown, want %s released", got, taskID)
				}
				return
			}
			// An unacked job comes back once its lease lapses. The last extension
			// landed before stop, so two leases after stop it would be back.
			if got := pollPending(t, client, sub, time.Now().Add(2*lease)); len(got) != 0 {
				t.Fatalf("subscription redelivered %v after its lease lapsed, want the job acked", got)
			}
		})
	}
}

// rpcFaults injects broker faults into the grader's client and records, in
// order, every message pull, ack, nack and lease lapse it sees.
type rpcFaults struct {
	mu     sync.Mutex
	events []string
	// failAcks fails that many Acknowledge calls, or all of them when negative.
	failAcks int
	// loseExtension applies the extension with this 1-based number and then
	// reports a failure, as when its response is lost. Zero loses none.
	loseExtension int
	extensions    int
	// lostExtension is closed when the lost extension was applied.
	lostExtension chan struct{}
}

func (f *rpcFaults) record(e string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, e)
}

func (f *rpcFaults) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func (f *rpcFaults) intercept(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	injected := status.Error(codes.FailedPrecondition, "injected fault")
	f.mu.Lock()
	failAck := strings.HasSuffix(method, "/Acknowledge") && f.failAcks != 0
	if failAck && f.failAcks > 0 {
		f.failAcks--
	}
	lose := false
	if m, ok := req.(*pubsubpb.ModifyAckDeadlineRequest); ok && m.AckDeadlineSeconds > 0 {
		f.extensions++
		lose = f.extensions == f.loseExtension
	}
	f.mu.Unlock()
	if failAck {
		f.record("ack failed")
		return injected
	}

	sent := time.Now()
	err := invoker(ctx, method, req, reply, cc, opts...)
	switch m := req.(type) {
	case *pubsubpb.AcknowledgeRequest:
		if err == nil {
			f.record("ack")
		}
	case *pubsubpb.PullRequest:
		if err == nil && len(reply.(*pubsubpb.PullResponse).ReceivedMessages) > 0 {
			f.record("pull")
		}
	case *pubsubpb.ModifyAckDeadlineRequest:
		if err == nil && m.AckDeadlineSeconds == 0 {
			f.record("nack")
		}
		if err == nil && lose {
			// The broker applied it no earlier than sent, so the message is
			// leased at least until sent plus the new deadline.
			time.AfterFunc(time.Until(sent.Add(time.Duration(m.AckDeadlineSeconds)*time.Second)), func() {
				f.record("lease lapsed")
			})
			close(f.lostExtension)
			return injected
		}
	}
	return err
}

// startFaultyGrader starts a one-slot grader whose broker calls go through f.
func startFaultyGrader(t *testing.T, f *rpcFaults, sub string, fn any, callbackURL string) func() {
	t.Helper()
	pubsubClientOptions = []option.ClientOption{option.WithGRPCDialOption(grpc.WithUnaryInterceptor(f.intercept))}
	defer func() { pubsubClientOptions = nil }()
	return startGrader(t, sub, 1, "blackbox", fn, callbackURL)
}

// waitEvents polls until f has recorded at least n events and returns them.
func waitEvents(t *testing.T, f *rpcFaults, n int) []string {
	t.Helper()
	deadline := time.Now().Add(waitLimit)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for got := f.snapshot(); ; got = f.snapshot() {
		if len(got) >= n {
			return got
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("got only %v within %s", got, waitLimit)
		}
		<-tick.C
	}
}

// A slot whose ack fails keeps the message and retries, so it never pulls a
// second message while the first is still leased to it.
func TestPubSubSlotHoldsMessageUntilAckLands(t *testing.T) {
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 600)
	callbackURL, _ := backendStub(t, http.StatusOK)
	publish(t, client, topic, "blackbox")
	publish(t, client, topic, "blackbox")

	faults := &rpcFaults{failAcks: 3}
	startFaultyGrader(t, faults, sub, func(string) (string, error) { return "{}", nil }, callbackURL)

	want := []string{"pull", "ack failed", "ack failed", "ack failed", "ack", "pull", "ack"}
	if got := waitEvents(t, faults, len(want)); !reflect.DeepEqual(got[:len(want)], want) {
		t.Fatalf("pulls and acks %v, want %v", got, want)
	}
}

// An extension whose response is lost may still have been applied, so a slot
// whose ack keeps failing waits out that extension before it pulls again.
func TestPubSubSlotCountsLostExtensionAsLeased(t *testing.T) {
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 600)
	callbackURL, _ := backendStub(t, http.StatusOK)
	publish(t, client, topic, "blackbox")
	publish(t, client, topic, "blackbox")

	// Extension 1 is the lease set on receipt, extension 2 the first renewal.
	faults := &rpcFaults{failAcks: -1, loseExtension: 2, lostExtension: make(chan struct{})}
	startFaultyGrader(t, faults, sub, func(string) (string, error) {
		<-faults.lostExtension
		return "{}", nil
	}, callbackURL)

	got := waitEvents(t, faults, 1)
	for !slices.Contains(got[1:], "pull") {
		got = waitEvents(t, faults, len(got)+1)
	}
	second := slices.Index(got[1:], "pull") + 1
	if lapsed := slices.Index(got, "lease lapsed"); lapsed < 0 || lapsed > second {
		t.Fatalf("events %v: the slot pulled again while its first message was still leased", got)
	}
}

// A stopping grader does not wait on a failing ack: it nacks once and stops
// within the shutdown grace.
func TestPubSubStopAbandonsFailingAck(t *testing.T) {
	client := requireEmulator(t)
	topic, sub := newSubscription(t, client, 600)
	callbackURL, _ := backendStub(t, http.StatusOK)
	publish(t, client, topic, "blackbox")

	faults := &rpcFaults{failAcks: -1}
	stop := startFaultyGrader(t, faults, sub, func(string) (string, error) { return "{}", nil }, callbackURL)
	waitEvents(t, faults, 2) // pull, ack failed

	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(shutdownGrace + time.Second):
		t.Fatalf("stop still blocked after the %s grace, events %v", shutdownGrace, faults.snapshot())
	}
	if got := faults.snapshot(); !slices.Contains(got, "nack") {
		t.Fatalf("events %v, want the stopping grader to nack its unacked job", got)
	}
}
