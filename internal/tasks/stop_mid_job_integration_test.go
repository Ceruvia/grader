//go:build linux && integration

// External package: it runs whole graders, and machinery imports tasks. It shares
// the tasks test binary, so its graders never use isolate box 1 alongside another test.
package tasks_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"cloud.google.com/go/pubsub/v2/pstest"
	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/machinery"
	"github.com/Ceruvia/grader/internal/pool"
	"github.com/Ceruvia/grader/internal/sandboxes"
	"github.com/RichardKnop/machinery/v2/tasks"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// graderChildEnv makes the test binary run as a grader (see runGrader).
	graderChildEnv = "GRADER_TEST_GRADER_CHILD"
	// stopWaitLimit bounds every wait for async work.
	stopWaitLimit = 90 * time.Second
	testProject   = "grader-test"
)

// slowProgram doubles its input after 12 s of CPU time: far longer than the 5 s
// the grader waits for running jobs at shutdown (README "Callback").
const slowProgram = `#include <stdio.h>
#include <time.h>
int main(void) {
    int x;
    if (scanf("%d", &x) != 1) return 1;
    clock_t start = clock();
    volatile unsigned long n = 0;
    while ((double)(clock() - start) / CLOCKS_PER_SEC < 12.0) n++;
    printf("%d\n", x * 2);
    return 0;
}
`

const crashingProgram = `#include <stdio.h>
int main(void) { volatile int *p = 0; *p = 1; printf("%d\n", *p); return 0; }
`

// A grader stopped (systemctl stop, as docker stop does) while a test case runs
// must not report that case. It waits 5 s for the job, then kills the run and
// releases the job for redelivery without a result, and the next grader grades
// the program AC. Staging, 9 Nov 2026: the case was stored as RE and the message acked.
func TestStopMidJobReleasesTheJob(t *testing.T) {
	if os.Getenv(graderChildEnv) != "" {
		runGrader(t)
		return
	}
	files := programServer(t)
	for name, start := range map[string]func(*testing.T) testBroker{"pubsub": startPubSub, "amqp": startAMQP} {
		t.Run(name, func(t *testing.T) {
			broker := start(t)
			url, callbacks := callbackServer(t)
			unit := startGraderUnit(t, broker.env, url)
			taskID := broker.publish(t, gradeSignature(t, files, "slow.zip"))
			waitForRunOf(t, "-i1.in")
			stopStart := time.Now()
			stopUnit(t, unit)
			// The 5 s wait, then the kill. A stop that let the run go on would take 10 s or more.
			if took := time.Since(stopStart); took > 8*time.Second {
				t.Fatalf("stop took %s: the released run was not killed", took)
			}

			select {
			case body := <-callbacks:
				t.Fatalf("stopped grader reported a result: %s", body)
			default:
			}
			broker.released(t)

			startGraderUnit(t, broker.env, url)
			if got, want := verdicts(t, receiveCallback(t, callbacks), taskID), "AC"; got != want {
				t.Fatalf("regraded verdicts %s, want %s", got, want)
			}
		})
	}
}

// A program that really crashes stays RE: only the grader's own stop is not a verdict.
func TestCrashingProgramIsRE(t *testing.T) {
	broker := startPubSub(t)
	url, callbacks := callbackServer(t)
	startGraderUnit(t, broker.env, url)
	taskID := broker.publish(t, gradeSignature(t, programServer(t), "crash.zip"))
	if got, want := verdicts(t, receiveCallback(t, callbacks), taskID), "RE"; got != want {
		t.Fatalf("verdicts %s, want %s", got, want)
	}
}

// A crashed grader (kill -9) must not leave its student program running until
// the next start. With the main PID dead, KillMode=mixed SIGKILLs the rest of
// the unit at once, keepers included. A keeper is init of the box PID namespace,
// so the kernel kills the program too, although its cgroup is under
// isolate.service. The unit runs without ExecStopPost: KillMode alone must do it.
func TestCrashKillsTheRunningProgram(t *testing.T) {
	t.Cleanup(func() { _ = sandboxes.CleanBox("/usr/local/bin/isolate", 1) }) // the crash leaves the box
	broker := startPubSub(t)
	url, _ := callbackServer(t)
	unit := startUnitWith(t, broker.env, url, "KillMode=")
	broker.publish(t, gradeSignature(t, programServer(t), "slow.zip"))
	waitForRunOf(t, "-i1.in")
	root, err := os.ReadFile("/run/isolate/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	procs := filepath.Join(strings.TrimSpace(string(root)), "box-1", "cgroup.procs")
	// isolate moves the program into the box cgroup a moment after it starts.
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
		pids, err := os.ReadFile(procs)
		if err == nil && len(bytes.TrimSpace(pids)) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no program running in box 1 before the crash: %q, %v", pids, err)
		}
	}
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit).Output()
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("kill", "-9", strings.TrimSpace(string(out))).CombinedOutput(); err != nil {
		t.Fatalf("kill -9: %v: %s", err, out)
	}
	crash := time.Now()
	for {
		pids, err := os.ReadFile(procs)
		if os.IsNotExist(err) || (err == nil && len(bytes.TrimSpace(pids)) == 0) {
			return
		}
		if time.Since(crash) > 2*time.Second {
			t.Fatalf("box 1 still runs %q 2 s after the grader crashed (err %v)", pids, err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// The unit's ExecStopPost is a backstop: it kills the pool's box cgroups after
// any exit, for a box whose keeper the unit's kill did not reach. Here the keeper
// runs outside the unit, so only ExecStopPost can reach the program.
func TestExecStopPostKillsBoxesOutsideTheUnit(t *testing.T) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("needs systemd (run inside the grader test container)")
	}
	const isolate = "/usr/local/bin/isolate"
	if err := sandboxes.CleanBox(isolate, 1); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command(isolate, "--cg", "-b", "1", "--init").CombinedOutput(); err != nil {
		t.Fatalf("init: %v: %s", err, out)
	}
	keeper := exec.Command(isolate, "--cg", "-b", "1", "--run", "--", "/bin/sleep", "3002")
	if err := keeper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = keeper.Process.Kill()
		_ = keeper.Wait()
		_ = sandboxes.CleanBox(isolate, 1)
	})
	root, err := os.ReadFile("/run/isolate/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	procs := filepath.Join(strings.TrimSpace(string(root)), "box-1", "cgroup.procs")
	boxEmpty := func() bool {
		pids, err := os.ReadFile(procs)
		return os.IsNotExist(err) || (err == nil && len(bytes.TrimSpace(pids)) == 0)
	}
	for deadline := time.Now().Add(5 * time.Second); boxEmpty(); time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("the program never started in box 1")
		}
	}

	unitFile, err := os.ReadFile("../../deployment/services/grader-docker.service")
	if err != nil {
		t.Fatal(err)
	}
	unit := "grader-stoppost-" + strconv.FormatInt(time.Now().UnixNano(), 36) + ".service"
	args := []string{"--unit=" + unit, "--collect", "--setenv=GRADER_WORKER_COUNT=1"}
	for _, line := range strings.Split(string(unitFile), "\n") {
		if strings.HasPrefix(line, "KillMode=") || strings.HasPrefix(line, "ExecStopPost=") {
			args = append(args, "-p", line)
		}
	}
	if out, err := exec.Command("systemd-run", append(args, "/bin/sleep", "600")...).CombinedOutput(); err != nil {
		t.Fatalf("systemd-run: %v: %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command("systemctl", "stop", unit).Run() })
	pid, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", unit).Output()
	if err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("kill", "-9", strings.TrimSpace(string(pid))).CombinedOutput(); err != nil {
		t.Fatalf("kill -9: %v: %s", err, out)
	}
	for crash := time.Now(); !boxEmpty(); time.Sleep(20 * time.Millisecond) {
		if time.Since(crash) > 2*time.Second {
			t.Fatal("box 1 still runs its program 2 s after the unit's main process was killed")
		}
	}
}

// runGrader is the child: what cmd/server does, with config from the environment.
func runGrader(t *testing.T) {
	cfg, err := config.GetAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if err := pool.NewSandboxPool("/usr/local/bin/isolate", cfg.WorkerCount); err != nil {
		t.Fatal(err)
	}
	defer pool.Pool.Close()
	t.Log(machinery.LaunchWorker(cfg))
}

// startGraderUnit runs a grader child as a transient systemd service with the
// KillMode and ExecStopPost of the deployed unit, so `systemctl stop` signals it as in production.
func startGraderUnit(t *testing.T, env map[string]string, callbackURL string) string {
	t.Helper()
	return startUnitWith(t, env, callbackURL, "KillMode=", "ExecStopPost=")
}

// startUnitWith is startGraderUnit with only the deployed unit lines that have
// one of the given prefixes.
func startUnitWith(t *testing.T, env map[string]string, callbackURL string, prefixes ...string) string {
	t.Helper()
	if _, err := exec.LookPath("systemd-run"); err != nil {
		t.Skip("needs systemd (run inside the grader test container)")
	}
	unitFile, err := os.ReadFile("../../deployment/services/grader-docker.service")
	if err != nil {
		t.Fatal(err)
	}
	unit := "grader-test-" + strconv.FormatInt(time.Now().UnixNano(), 36) + ".service"
	args := []string{"--unit=" + unit, "--collect"}
	for _, line := range strings.Split(string(unitFile), "\n") {
		if slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(line, p) }) {
			args = append(args, "-p", line)
		}
	}
	for k, v := range env {
		args = append(args, "--setenv="+k+"="+v)
	}
	for k, v := range map[string]string{
		graderChildEnv: "1", "GRADER_ENV": "development", "GRADER_WORKER_COUNT": "1",
		"BACKEND_API_TOKEN": "token", "BACKEND_CALLBACK_ENDPOINT": callbackURL + "/submissions/{submission_id}",
	} {
		args = append(args, "--setenv="+k+"="+v)
	}
	args = append(args, os.Args[0], "-test.run=^TestStopMidJobReleasesTheJob$", "-test.v")
	if out, err := exec.Command("systemd-run", args...).CombinedOutput(); err != nil {
		t.Fatalf("systemd-run: %v: %s", err, out)
	}
	t.Cleanup(func() {
		if t.Failed() {
			out, _ := exec.Command("journalctl", "-u", unit, "--no-pager", "-o", "cat").CombinedOutput()
			t.Logf("journal of %s:\n%s", unit, out)
		}
		_ = exec.Command("systemctl", "stop", unit).Run()
	})
	return unit
}

func stopUnit(t *testing.T, unit string) {
	t.Helper()
	if out, err := exec.Command("systemctl", "stop", unit).CombinedOutput(); err != nil {
		t.Fatalf("systemctl stop: %v: %s", err, out)
	}
}

// waitForRunOf polls until an isolate run with arg is running.
func waitForRunOf(t *testing.T, arg string) {
	t.Helper()
	for deadline := time.Now().Add(stopWaitLimit); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		cmdlines, _ := filepath.Glob("/proc/[0-9]*/cmdline")
		for _, path := range cmdlines {
			cmdline, _ := os.ReadFile(path)
			args := strings.Split(string(cmdline), "\x00")
			if strings.HasSuffix(args[0], "isolate") && slices.Contains(args, "--run") && slices.Contains(args, arg) {
				return
			}
		}
	}
	t.Fatalf("no isolate run with %s within %s", arg, stopWaitLimit)
}

type testBroker struct {
	env      map[string]string
	publish  func(*testing.T, *tasks.Signature) string
	released func(*testing.T) // asserts the stopped grader handed its job back
}

// startPubSub runs pstest, the Pub/Sub fake of the client library, on a local port.
func startPubSub(t *testing.T) testBroker {
	t.Helper()
	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })
	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	client, err := pubsub.NewClient(context.Background(), testProject, option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	topic := fmt.Sprintf("projects/%s/topics/jobs", testProject)
	if _, err := client.TopicAdminClient.CreateTopic(context.Background(), &pubsubpb.Topic{Name: topic}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.SubscriptionAdminClient.CreateSubscription(context.Background(), &pubsubpb.Subscription{
		Name: fmt.Sprintf("projects/%s/subscriptions/graders", testProject), Topic: topic, AckDeadlineSeconds: 600,
	}); err != nil {
		t.Fatal(err)
	}
	var messageID string
	return testBroker{
		env: map[string]string{
			"QUEUE_BROKER": config.BrokerPubSub, "GCP_PROJECT_ID": testProject,
			"PUBSUB_SUBSCRIPTION_NAME": "graders", "PUBSUB_EMULATOR_HOST": srv.Addr,
		},
		publish: func(t *testing.T, sig *tasks.Signature) string {
			publisher := client.Publisher(topic)
			defer publisher.Stop()
			id, err := publisher.Publish(context.Background(), &pubsub.Message{Data: mustJSON(t, sig)}).Get(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			messageID = id
			return sig.UUID
		},
		released: func(t *testing.T) {
			// 600 s ack deadline: only a nack (deadline 0) brings the job back within the test.
			m := srv.Message(messageID)
			if m.Acks != 0 || len(m.Modacks) == 0 || m.Modacks[len(m.Modacks)-1].AckDeadline != 0 {
				t.Fatalf("stopped grader left acks %d, modacks %+v, want a nack and no ack", m.Acks, m.Modacks)
			}
		},
	}
}

// startAMQP uses the RabbitMQ at GRADER_TEST_AMQP_URL, with a fresh queue.
func startAMQP(t *testing.T) testBroker {
	t.Helper()
	url := os.Getenv("GRADER_TEST_AMQP_URL")
	if url == "" {
		t.Skip("GRADER_TEST_AMQP_URL not set")
	}
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
	// Declared as Machinery declares it, so the grader's declaration matches.
	if _, err := ch.QueueDeclare(queue, true, false, false, false, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = ch.QueueDelete(queue, false, false, false) })
	return testBroker{
		env: map[string]string{"QUEUE_BROKER": config.BrokerAMQP, "QUEUE_BROKER_URL": url, "QUEUE_NAME": queue},
		publish: func(t *testing.T, sig *tasks.Signature) string {
			if err := ch.PublishWithContext(context.Background(), "", queue, false, false,
				amqp.Publishing{ContentType: "application/json", Body: mustJSON(t, sig)}); err != nil {
				t.Fatal(err)
			}
			return sig.UUID
		},
		released: func(t *testing.T) {
			q, err := ch.QueueDeclarePassive(queue, true, false, false, false, nil)
			if err != nil {
				t.Fatal(err)
			}
			if q.Messages != 1 {
				t.Fatalf("queue holds %d ready messages after the stop, want the job back", q.Messages)
			}
		},
	}
}

// programServer serves the teacher archive (21 -> 42) and one archive per program.
func programServer(t *testing.T) string {
	t.Helper()
	archives := map[string][]byte{
		"/grader.zip": zipOf(t, map[string]string{"1.in": "21\n", "1.out": "42\n"}),
		"/slow.zip":   zipOf(t, map[string]string{"main.c": slowProgram}),
		"/crash.zip":  zipOf(t, map[string]string{"main.c": crashingProgram}),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func gradeSignature(t *testing.T, files, program string) *tasks.Signature {
	t.Helper()
	sig, err := tasks.NewSignature("blackbox", []tasks.Arg{
		{Type: "string", Value: "submission-1"},
		{Type: "string", Value: files + "/grader.zip"},
		{Type: "string", Value: files + "/" + program},
		{Type: "[]string", Value: []string{"1.in"}},
		{Type: "[]string", Value: []string{"1.out"}},
		{Type: "int", Value: 30000},
		{Type: "int", Value: 262144},
		{Type: "string", Value: "C"},
		{Type: "string", Value: "main.c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return sig
}

// callbackServer answers every callback 200 and sends its body on the channel.
func callbackServer(t *testing.T) (string, <-chan []byte) {
	t.Helper()
	bodies := make(chan []byte, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies <- body
	}))
	t.Cleanup(srv.Close)
	return srv.URL, bodies
}

func receiveCallback(t *testing.T, bodies <-chan []byte) []byte {
	t.Helper()
	select {
	case body := <-bodies:
		return body
	case <-time.After(stopWaitLimit):
		t.Fatalf("no callback within %s", stopWaitLimit)
		return nil
	}
}

// verdicts checks the callback is the SUCCESS of taskID and returns its case verdicts, comma-separated.
func verdicts(t *testing.T, body []byte, taskID string) string {
	t.Helper()
	var callback struct {
		TaskID string `json:"task_id"`
		State  string `json:"state"`
		Result string `json:"result"`
	}
	var result struct {
		Cases []struct {
			Verdict struct{ Code string } `json:"verdict"`
		} `json:"testcase_result"`
	}
	if err := json.Unmarshal(body, &callback); err != nil || json.Unmarshal([]byte(callback.Result), &result) != nil ||
		callback.TaskID != taskID || callback.State != tasks.StateSuccess {
		t.Fatalf("callback %s, want the SUCCESS of %s", body, taskID)
	}
	var codes []string
	for _, c := range result.Cases {
		codes = append(codes, c.Verdict.Code)
	}
	return strings.Join(codes, ",")
}

func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
