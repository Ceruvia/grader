# Ceruvia grader

The grader runs student code for Ceruvia programming questions and reports a verdict per test case.
It is a Go worker. It has no HTTP API and no database of its own.

## Release status

The work of the `refactor/production-ready` branch was merged to `dev` on 3 Nov 2026 (grader #10).
`dev`, `staging` and `main` carry the same content.

- Production grades through GCP Pub/Sub with one grader on the VM `grader1`, since the cutover of
  3 Nov 2026. The grader is not on the `main` VM.
- Staging runs the same code on the `main` VM, with RabbitMQ, only inside a test window. The one
  grader of `grader1` serves staging during that window.

Deployment of the whole stack (backend, frontend, broker, grader) is described in `Backend/DEPLOY.md`
in the Backend repository. This repository holds only the grader code, its image and its CI.

## What it does

1. The backend publishes a [Machinery](https://github.com/RichardKnop/machinery) task. `QUEUE_BROKER`
   picks the broker. With `amqp` (default) the grader consumes RabbitMQ queue `QUEUE_NAME` (exchange
   `machinery_exchange`, type `direct`, binding key `machinery_task`). With `gcppubsub` it pulls the
   Pub/Sub subscription `PUBSUB_SUBSCRIPTION_NAME`, where each message body is the task signature JSON.
2. It takes a free [isolate](https://github.com/ioi/isolate) box from its pool. The pool has
   `GRADER_WORKER_COUNT` boxes, so that many tasks run in parallel.
3. It downloads the question archive (`grader.zip`: test cases, teacher files, optional makefile) and
   unzips it on the host, outside the box. Teacher files go into the box. Hidden `.in` and `.out` files
   stay outside and each input is copied in just before its own test runs.
4. It downloads the submission archive (`submission.zip`) and unzips it inside the box. Student files
   never overwrite teacher files. When the teacher sent a makefile, student makefiles are removed.
   After extraction, student build products (`.o`, `.hi`, `.dyn_o`, `.dyn_hi`, `.p_o`, `.p_hi`,
   `.gch`, `.pyc`, and `__pycache__` directories) are removed by their effective filenames so
   cached objects cannot bypass source compilation. Teacher-provided build products are preserved.
5. It compiles and runs the code in isolate v2.1 with cgroup limits, then compares the output with the
   expected `.out` file.
6. It POSTs the final state to the backend callback `BACKEND_CALLBACK_ENDPOINT` with the header
   `X-API-Key: <BACKEND_API_TOKEN>`. Nothing is stored in the broker. Body:
   `{"task_id": "<task uuid>", "kind": "grade" or "compile_check", "state": "SUCCESS" or "FAILURE", "result": "<task result JSON as a string>", "error": "<FAILURE only>"}`.
   `result` is absent on `FAILURE`.

Verdicts: `AC` (Accepted), `WA` (Wrong Answer), `CE` (Compilation Error), `RE` (Runtime Error),
`TLE` (Time Limit Exceeded), `XX` (Internal Error).

### Tasks

The worker registers four task names (`internal/machinery/setup.go`). Arguments are positional.

| Task | Arguments | Callback |
|---|---|---|
| `blackbox` | submission id, grader zip URL, submission zip URL, input files, output files, time limit (ms), memory limit (KB), language, main source file | yes |
| `blackbox_with_builder` | submission id, grader zip URL, submission zip URL, input files, output files, time limit (ms), memory limit (KB), language, builder, compile script, run script | yes |
| `compile_check` | submission id, grader zip URL, submission zip URL, language, main source file, deadline (Unix ms) | yes, `kind` `compile_check` |
| `compile_check_with_builder` | submission id, grader zip URL, submission zip URL, language, builder, compile script, deadline (Unix ms) | yes, `kind` `compile_check` |

Input and output files come in pairs with the same base name (`1.in` and `1.out`).
Compile checks return `compiled`, `compile_error` or `internal_error` in the callback `result`.
A compile check whose deadline has passed is skipped and returns `internal_error`.

When the grader itself fails (download, unzip, sandbox), the task returns an error. Machinery marks it
`FAILURE` and the backend shows `XX`, so the student is not blamed with `CE`.

### Callback

`internal/machinery/callback.go`: each attempt has a 10 s timeout. There are up to 4 attempts, with
waits of 2, 4 and 8 s. It retries on network errors, 5xx and 429, and stops on any other non-2xx
status, which counts as the backend's final answer.

With `amqp` the message is acked after the callback either way. If every attempt fails, the submission
stays pending and the backend sweeper marks it stuck.

With `gcppubsub` delivery is at least once:

- The message is acked only after a 2xx or a final 4xx. If every attempt fails, it is nacked, so
  the job is graded again (by any grader). The backend ignores a second result for a final row.
- A grader holds at most `GRADER_WORKER_COUNT` unacked messages, so graders share the load. Each
  worker slot pulls one message only when it is free. A failed ack or nack is retried every second,
  and the slot pulls nothing new until it lands or the message's lease has lapsed.
- The grader keeps a 30 s lease on a running job and extends it every 10 s, for up to 1 hour.
- On SIGTERM the grader stops pulling and waits 5 s for running jobs. Jobs still running then are
  nacked so another grader takes them at once. A killed grader's jobs come back within 30 s.

### Limits

| Step | Limit | Where |
|---|---|---|
| Download of each archive | 30 s, 64 MiB | `internal/helper/files/downloader.go` |
| Teacher archive unzip (host) | 30 s | `internal/tasks/prepare.go` |
| Submission unzip (in box) | 10 s CPU, 20 s wall, 256 MiB memory | `internal/tasks/prepare.go` |
| Compile | 20 s CPU, 60 s wall, 1 GiB memory | `internal/orchestrator/compilers/compiler_source_file.go` |
| Run, per test case | question time limit (CPU), wall = time limit + 5 s, question memory limit | `internal/orchestrator/engines/engine_blackbox.go` |
| Every box | 50 processes, 100 MiB per file, 256 MiB and 10000 inodes of scratch | `internal/sandboxes/` |
| Diagnostics in one result | 64 KiB in total | `internal/helper/diagnostics/message.go` |

The question memory limit applies the same way to every language. See known issue B10 for Haskell.

## Supported languages

The registry is `internal/languages/factory.go`. The backend sends the language name as a string, and
it must match exactly. The `programming_languages` rows in the backend database must use these names
(`Backend/DEPLOY.md` section 1.10).

| Name | Extensions | Compile | Run |
|---|---|---|---|
| `C` | `.c` | `/usr/bin/gcc -std=gnu99 -o <main> <sources> -O2 -lm` | `./<main>` |
| `C++11` | `.cpp`, `.cc` | `/usr/bin/g++ -std=c++11 -o <main> <sources> -O2 -lm` | `./<main>` |
| `C++17` | `.cpp`, `.cc` | `/usr/bin/g++ -std=c++17 -o <main> <sources> -O2 -lm` | `./<main>` |
| `C++20` | `.cpp`, `.cc` | `/usr/bin/g++ -std=c++20 -o <main> <sources> -O2 -lm` | `./<main>` |
| `Haskell` | `.hs` | `/usr/bin/ghc --make <main>` | `./<main>` |
| `Java` | `.java` | `/usr/bin/javac <sources>`, then `/usr/bin/jar cfe <main>.jar <main> *.class` | `/usr/bin/java -jar <main>.jar` |
| `Python 3` | `.py` | none (`/bin/true`) | `/usr/bin/python3 <main>.py` |
| `Makefile` (builder) | any | `/usr/bin/make -B -f <teacher makefile> <compile script>` | `./<run script>` |

`<main>` is the main source file name without its extension. The image has no JDK, so Java does not
work in the image today (see known issues).

## Configuration

`internal/config/config.go` reads these variables. `.env.example` lists them with comments.

| Name | Required | Default | Notes |
|---|---|---|---|
| `LOAD_DOTENV` | no | `false` | `true` loads `.env` from the working directory and fails startup if it cannot be read. Must be a boolean. |
| `GRADER_NAME` | no | the hostname | Label for logs and metrics. |
| `GRADER_ENV` | no | `production` | `production` or `development`. Any other value fails startup. `development` turns on debug logs and skips the required checks below. |
| `GRADER_WORKER_COUNT` | no | `3` | Positive integer. Boxes graded in parallel, the AMQP prefetch count and the Pub/Sub cap on unacked messages. Keep it at or below the CPU cores the container gets. |
| `QUEUE_BROKER` | no | `amqp` | `amqp` (RabbitMQ) or `gcppubsub` (GCP Pub/Sub). Any other value fails startup. |
| `QUEUE_BROKER_URL` | with `amqp`, in production | empty | AMQP URL of RabbitMQ. |
| `QUEUE_NAME` | with `amqp`, in production | empty | Must equal `QUEUE_NAME` in the backend, or tasks are never picked up. |
| `GCP_PROJECT_ID` | with `gcppubsub`, in production | empty | GCP project of the topic and subscription. |
| `PUBSUB_SUBSCRIPTION_NAME` | with `gcppubsub`, in production | empty | Pull subscription on the backend's topic. Every grader uses the same one. |
| `PUBSUB_TOPIC_NAME` | no | empty | Read only by `cmd/send`. |
| `GOOGLE_APPLICATION_CREDENTIALS` | no | empty | Read by the Google client. Path of a mounted service account key file. Without it the client uses the VM's default credentials. |
| `PUBSUB_EMULATOR_HOST` | no | empty | Read by the Google client. Points it at a local emulator. Tests and local stacks only. |
| `BACKEND_API_TOKEN` | yes in production | empty | Secret. Must equal `APP_API_KEY` in the backend. |
| `BACKEND_CALLBACK_ENDPOINT` | yes in production | empty | Absolute `http` or `https` URL with a `{submission_id}` placeholder. Checked whenever it is set. Example: `http://ingress/v1/submissions/{submission_id}/programming/grade-callback` |
| `MONITORING_LOKI_URL` | no | empty | When set, logs are also sent to Loki. |
| `MONITORING_INFLUX_URL` | no | empty | When set, pool metrics are pushed to InfluxDB. |
| `MONITORING_INFLUX_TOKEN` | no | empty | Secret. Used with `MONITORING_INFLUX_URL`. |
| `MONITORING_INFLUX_ORGANIZATION` | no | empty | Used with `MONITORING_INFLUX_URL`. |
| `MONITORING_INFLUX_BUCKET` | no | empty | Used with `MONITORING_INFLUX_URL`. |

`CERUVIA_POOL_RESTART_HELPER` is read only by `internal/tasks/pool_lifecycle_linux_test.go`. Never set it.

In the container, `dump-env.service` copies the container environment to `/run/grader/env` for the
grader unit. Only `LOAD_DOTENV`, `GCP_PROJECT_ID`, `GOOGLE_APPLICATION_CREDENTIALS` and names that
start with `GRADER_`, `QUEUE_`, `BACKEND_`, `MONITORING_` or `PUBSUB_` pass through. A variable with
any other name never reaches the grader.

The deployed values live in env files on the VM: `grader.env` for production
and `grader.staging.env` for staging (`Backend/DEPLOY.md` sections 1.5 and 7.1).

## Runtime requirements

The grader only runs in its Docker image on a Linux host. The image is `ghcr.io/ceruvia/grader`.

- **cgroup v2 on the host.** isolate runs every box with `--cg`. Check it on the host with
  `stat -fc %T /sys/fs/cgroup`, which must print `cgroup2fs` (`Backend/DEPLOY.md` section 1.1).
- **Privileged container with the host cgroup namespace.** The compose files in the Backend repo use
  `privileged: true`, `cgroup: host` and the volume `/sys/fs/cgroup:/sys/fs/cgroup:rw`. With
  `docker run` that is `--privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw`.
- **systemd as PID 1.** The image starts `/sbin/init`. systemd starts three units:
  - `isolate.service` (installed by isolate v2.1) sets up the isolate cgroup.
  - `dump-env.service` writes the filtered environment to `/run/grader/env`.
  - `grader-docker.service` waits for `isolate.service`, runs `isolate --cg -b 999 --cleanup` and
    `isolate --cg -b 999 --init` as a start check, then runs `/usr/local/bin/grader`. It restarts the
    grader 5 s after any exit.
- **isolate v2.1**, pinned in the `Dockerfile`. v2.3 and later need libseccomp.
- **Health check.** The compose files mark the container healthy when
  `systemctl is-active --quiet isolate.service && systemctl is-active --quiet grader-docker.service`
  succeeds. Docker does not restart an unhealthy container, the check only drives `depends_on`,
  `up --wait` and operators.

Staging gives the grader 768 MiB of memory and 2 CPUs (`Backend/deployments/docker/docker-compose.staging.yaml`).
The isolate boxes live inside the container's cgroup, so compilers and student programs share that budget
with the grader (`Backend/DEPLOY.md` section 7.5). Size `GRADER_WORKER_COUNT` with it.

The grader runs as a systemd unit inside the container, so its logs (tasks, isolate errors, callback
errors) go to the systemd journal in the container. `grader-log-forward.service` copies the lines of
`grader-docker.service` from the journal to the container's stdout, so `docker logs` and
`docker compose logs grader` show them too (images from 2 Nov 2026 on). The journal stays the full
copy. On the VM, from `Backend/deployments/docker`, use the compose file of the environment
(`docker-compose.prod.yaml` or `docker-compose.staging.yaml`):

```bash
docker compose -f docker-compose.staging.yaml exec grader journalctl -u grader-docker.service -n 100 --no-pager
```

Add `-f` to follow new lines, or `--since` (for example `--since "10 min ago"`) for a time window.

## Build the image

The `Dockerfile` has five stages:

| Stage | Contents |
|---|---|
| `isolate` | Builds isolate v2.1. Only the installed files leave this stage. |
| `base` | Ubuntu 22.04, systemd, isolate, `build-essential`, `python3`, `python-is-python3`, `ghc`, `zip`, `unzip`, and the systemd units. |
| `build` | `golang:1.26`, builds `/out/grader` from `./cmd/server`. |
| `test` | `base` plus the Go toolchain, with the whole source in `/src`. |
| `runtime` | The default target. `base` plus the grader binary, no Go toolchain. |

CI builds `runtime` and pushes it to GHCR (see CI below). For a local full stack, the Backend e2e
compose builds this directory as `ceruvia/grader:local`. From `Backend/deployments/docker`:

```bash
docker compose -f docker-compose.e2e.yaml up -d --build --wait
```

## Tests

There are two kinds of tests. Isolate tests carry the `linux && integration` build tag, so a plain
`go test` skips them.

**Unit tests** need no isolate. They run on any machine with Go 1.26 or later:

```bash
go build ./... && go vet ./...
go test ./...
```

**Isolate tests** run real compilers in real boxes. Every file that calls isolate has the build tag
`linux && integration`, and they use the fixtures under `tests/`. They need the `test` image, started
the way CI starts it (`.github/workflows/deploy.yml`, job `test`):

```bash
docker build --target test -t ceruvia/grader:test .
docker run -d --name grader-test --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw ceruvia/grader:test
```

Wait until this command succeeds. CI tries it once a second for up to 60 s.

```bash
docker exec grader-test systemctl is-active --quiet isolate.service
```

Then run the whole suite and remove the container:

```bash
docker exec -e GRADER_TEST_NO_JDK=1 -w /src grader-test sh -c 'go vet -tags integration ./... && go test -tags integration ./...'
docker rm -f grader-test
```

`GRADER_TEST_NO_JDK=1` skips the Java acceptance tests on purpose because the image has no JDK (see
Known issues). Without `-tags integration` the isolate tests do not compile in, so a green plain
`go test ./...` says nothing about grading.

The Pub/Sub tests (`internal/machinery/pubsub_integration_test.go`) start the emulator image
`gcr.io/google.com/cloudsdktool/google-cloud-cli:emulators` with testcontainers, so they need Docker
on the host and skip unless `GRADER_TEST_PUBSUB=1`. CI does not set it. Run them on the host:

```bash
GRADER_TEST_PUBSUB=1 go test -tags integration -run TestPubSub ./internal/machinery/
```

## Bare-metal Makefile

The `Makefile` is an older bare-metal install path for Ubuntu. No deploy uses it. The Docker image is
the supported path. Targets: `install-dependency`, `install-isolate`, `build`, `install`, `clean`, and
`all` for all of them. `make build` builds `./grader` with the local Go toolchain and prints the Go
version from `go.mod`.

Known gaps if you use it:

- `install-isolate` builds the latest isolate, not v2.1.
- `install-dependency` installs `openjdk-17-jdk` but not `ghc`.
- `install` copies `deployment/services/grader.service`, which uses `/opt/grader` as working
  directory and reads no env file. You must give the grader its variables yourself.

## Project layout

| Path | Contents |
|---|---|
| `cmd/server` | The grader worker (`main`). |
| `cmd/send` | A manual task sender for debugging. It reads the same env vars as the worker and sends one hard-coded `blackbox` task with fixed public archive URLs. |
| `internal/config` | Env var parsing and validation. |
| `internal/machinery` | Worker setup, task registration, the backend callback. |
| `internal/tasks` | The four task handlers: download, unzip, prepare the box, call the orchestrator. |
| `internal/orchestrator` | Compile, run each test case, evaluate (`compilers`, `engines`, `evaluator`). |
| `internal/languages` | The language registry and compile and run commands. |
| `internal/sandboxes` | The isolate wrapper, box files, meta file parsing, scratch storage. |
| `internal/pool` | The pool of isolate boxes. |
| `internal/models` | Submission, test case and verdict types. |
| `internal/monitoring` | Logging (logrus, optional Loki) and metrics (optional InfluxDB). |
| `internal/helper` | Command builder, downloads, env helpers, diagnostics limits, test helpers. |
| `tests/` | Fixtures for the isolate tests (C, C++, Java, Haskell, Python, makefile questions). |
| `deployment/services` | systemd units: `grader-docker.service` and `dump-env.service` for the image, `grader.service` for bare metal. |

## CI

`.github/workflows/deploy.yml` runs on pushes to `main` and `staging`, on pull requests to `main`,
`staging` and `dev`, and by hand (`workflow_dispatch`, staging only).

| Job | When | What |
|---|---|---|
| `test` | always | Builds the `test` image, starts it privileged, waits for `isolate.service`, runs `go vet -tags integration ./... && go test -tags integration ./...` inside it. |
| `build` | not on pull requests | Builds the `runtime` image and pushes it to GHCR by digest, with no tag. |
| `publish` | not on pull requests, after `test` and `build` | Tags the tested digest `sha-<commit>`, the branch name, and `latest` on `main`. |
| `deploy-staging` | push to `staging` or a manual run on `staging` | Connects over SSH through a Cloudflare tunnel and runs the staging deploy for `grader sha-<commit>@<digest>` (`Backend/DEPLOY.md` section 7.4). |

There is no production deploy job. Production is deployed by hand with `deploy.sh` (`Backend/DEPLOY.md`
section 2.1).

`deploy-staging` uses these repository secrets in the `staging` environment: `STAGING_SSH_HOST`,
`STAGING_SSH_PORT`, `STAGING_SSH_USER`, `STAGING_SSH_KEY`, `STAGING_SSH_KNOWN_HOSTS`,
`STAGING_CF_ACCESS_CLIENT_ID`, `STAGING_CF_ACCESS_CLIENT_SECRET`.

Dependabot (`.github/dependabot.yml`) updates the pinned GitHub Actions weekly.

## Known issues

- **B10, Haskell memory.** A Haskell hello-world used about 277 MB in isolate. Olympia Haskell
  questions allow 64 MB. The grader applies the question memory limit to every language and has no
  Haskell rule (`internal/orchestrator/engines/engine_blackbox.go`). Measure where the memory goes
  (GHC runtime, compile or run) and decide the limit rule before any Haskell exam.
- **Java has no JDK in the image.** `Java` is in the language registry, but the `Dockerfile` installs
  no JDK. A Java submission gets `Compile Error` with `/usr/bin/javac: No such file or directory`.
  The Java acceptance tests fail without `/usr/bin/javac` and skip only when `GRADER_TEST_NO_JDK=1`
  is set. Do not offer Java questions until the image has a JDK.
- **Bare-metal Makefile is stale.** See the Makefile section above.
