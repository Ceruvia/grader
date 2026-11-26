# Ceruvia Grader

The grader is the autograding worker of Ceruvia, a programming learning management system. It consumes grading jobs that the Ceruvia backend publishes to RabbitMQ, compiles and runs each submission inside an [isolate](https://github.com/ioi/isolate) sandbox against the question's test cases, stores the grading result in the message broker, and notifies the backend over HTTP that the job has finished.

The grader has no HTTP server and no database of its own. Everything it needs arrives in the job message: download URLs for the submission and test case archives, the test case file names, resource limits, and the language.

For how the grader fits together with the backend, frontend, database, and object storage, see the system architecture overview: [Backend/ARCHITECTURE.md](https://github.com/Ceruvia/Backend/blob/dev/ARCHITECTURE.md).

## Contents

- [Tech stack](#tech-stack)
- [Project structure](#project-structure)
- [How grading works](#how-grading-works)
- [Prerequisites](#prerequisites)
- [Setup and running locally](#setup-and-running-locally)
- [Configuration](#configuration)
- [Testing](#testing)
- [Deployment](#deployment)
- [Team](#team)

## Tech stack

| Concern | Technology |
| --- | --- |
| Language | Go 1.24.2 (`go.mod`) |
| Job queue | RabbitMQ through [Machinery v2](https://github.com/RichardKnop/machinery) (AMQP broker and AMQP result backend) |
| Sandbox | [isolate](https://github.com/ioi/isolate) with control groups (`--cg`); the Docker image pins isolate `v2.1` |
| Logging | logrus (JSON to stdout), optional Loki push via `lokirus` |
| Metrics | Optional InfluxDB v2 push via `influxdb-client-go`, host stats via `gopsutil` |
| Configuration | Environment variables, optionally loaded from `.env` with `godotenv` |
| Packaging | Docker image based on `ubuntu:22.04` running systemd as PID 1; published to GHCR |

## Project structure

```text
grader/
├── cmd/
│   ├── server/            # Entry point: builds the sandbox pool and launches the Machinery worker
│   └── send/              # Development tool: publishes one sample "blackbox" job and prints the result
├── internal/
│   ├── config/            # Reads environment variables into ServerConfig
│   ├── machinery/         # Machinery server/worker setup, task registration, backend callback
│   ├── tasks/             # Task handlers (blackbox, blackbox_with_builder) and file download/unzip
│   ├── pool/              # Fixed-size pool of isolate boxes, one per concurrent job
│   ├── sandboxes/         # isolate wrapper: command building, limits, redirection, meta file parsing
│   ├── orchestrator/
│   │   ├── compilers/     # Compiles inside the sandbox with fixed compile limits
│   │   ├── engines/       # Black-box engine: runs the program once per test case
│   │   └── evaluator/     # Turns sandbox status and output into a verdict
│   ├── languages/         # Compile and run commands per language, plus the Makefile builder
│   ├── models/            # Submission, limit, test case, and verdict types
│   ├── monitoring/
│   │   ├── logging/       # logrus setup and Loki hook
│   │   └── metrics/       # InfluxDB metrics pusher
│   └── helper/            # Small utilities (env parsing, file copy/download, command builder, test asserts)
├── tests/                 # Fixtures for the Go tests (sources, Makefiles, .in/.out files, isolate meta files)
├── deployment/services/   # systemd units for bare-metal and Docker deployments
├── Dockerfile             # Image with systemd, isolate, gcc/g++/make, and the grader binary
├── Makefile               # Bare-metal install of dependencies, isolate, and the grader service
└── .github/workflows/     # CI: build, vet, and publish the image to GHCR
```

## How grading works

### Job transport

The grader is a [Machinery](https://github.com/RichardKnop/machinery) worker. `internal/machinery/setup.go` configures it with:

| Setting | Value |
| --- | --- |
| Broker | `QUEUE_BROKER_URL` (AMQP) |
| Result backend | `QUEUE_RESULT_URL` (AMQP) |
| Exchange | `machinery_exchange`, type `direct` |
| Queue consumed | `QUEUE_NAME` (code default `ceruvia_submissions`) |
| Queue binding key | `machinery_task` |
| Prefetch count | `GRADER_WORKER_COUNT` |
| Worker concurrency | `GRADER_WORKER_COUNT` |
| Result expiry | `QUEUE_RESULT_TTL` seconds |

The backend publishes with the routing key set to its own `QUEUE_NAME`, so the grader's `QUEUE_NAME` must be identical to the backend's. In the backend's `.env.example` and the staging deployment this is `grading_queue`.

Two task names are registered:

| Task name | Handler | Used when |
| --- | --- | --- |
| `blackbox` | `tasks.GradeBlackbox` | The question has no build system; the grader compiles the sources itself. |
| `blackbox_with_builder` | `tasks.GradeBlackboxWithBuilder` | The question uses a Makefile; the grader runs `make`. |

### Message format

Jobs are standard Machinery task signatures. The arguments are positional:

| # | Type | `blackbox` | `blackbox_with_builder` |
| --- | --- | --- | --- |
| 0 | `string` | Submission ID | Submission ID |
| 1 | `string` | Test case archive URL (`grader.zip`) | Test case archive URL |
| 2 | `string` | Submission archive URL (`submission.zip`) | Submission archive URL |
| 3 | `[]string` | Input file names, e.g. `1.in` | Input file names |
| 4 | `[]string` | Expected output file names, e.g. `1.out` | Expected output file names |
| 5 | `int` | Time limit in milliseconds | Time limit in milliseconds |
| 6 | `int` | Memory limit in kilobytes | Memory limit in kilobytes |
| 7 | `string` | Language name (see [Supported languages](#supported-languages)) | Language name |
| 8 | `string` | Main source file name, e.g. `main.c` | Builder name; only `Makefile` exists |
| 9 | `string` | | Compile target passed to `make` |
| 10 | `string` | | Run target (executable name or script) |

Input and output names are paired by index. The backend fills these from the programming question and generates the two URLs as presigned object storage URLs.

### Per-job steps

1. **Acquire a box.** At startup `cmd/server` creates a pool of `GRADER_WORKER_COUNT` isolate boxes with IDs `1..N`. Each job blocks until a box is free and returns it when finished, so at most `N` jobs grade at once on one grader process.
2. **Initialise the box.** The task runs `isolate -b <id> --cg --init` for the acquired box.
3. **Fetch files.** The grader downloads the test case archive to `grader.zip` and the submission archive to `submission.zip` inside the box directory with a plain HTTP GET (any non-200 response fails the job), then runs `unzip -q` inside the sandbox for each. The archives are expected to be flat: test case files and sources sit at the archive root.
4. **Compile.** The compile command runs in the sandbox with fixed limits: 20 s CPU time, 60 s wall time, 1 GB memory. Standard output and standard error are captured to `_compile.out`. A failed compile ends the job with status `Compile Error` and the compiler output as the error message. No test cases are run.
5. **Run each test case.** For every input/output pair the program runs in the sandbox with stdin from the input file and both stdout and stderr written to `<output>.actual`. Limits come from the job (see [Sandbox and limits](#sandbox-and-limits)).
6. **Evaluate.** Each run gets a verdict (see [Verdicts](#verdicts)).
7. **Store the result.** The handler returns the result as a JSON string. Machinery stores it in the AMQP result backend under the task UUID.
8. **Notify the backend.** A post-task hook sends `POST BACKEND_CALLBACK_ENDPOINT` (with `{submission_id}` replaced by argument 0), header `X-API-Key: BACKEND_API_TOKEN`, body `{"task_id": "<task UUID>"}`. The hook runs after Machinery has recorded the final task state, whether the task succeeded or failed. The backend then reads the result from the result backend using the task ID.

If downloading or unzipping fails, the task returns a Go error. Machinery marks it `FAILURE` and still fires the callback. The backend records an internal error for the submission. Tasks are not retried.

### Sequence of one grading job

```mermaid
sequenceDiagram
    autonumber
    participant BE as Backend
    participant MQ as RabbitMQ
    participant W as Grader worker
    participant S3 as Object storage
    participant ISO as isolate box

    BE->>MQ: Publish "blackbox" task to machinery_exchange (routing key = QUEUE_NAME)
    MQ->>W: Deliver task from QUEUE_NAME
    W->>W: Acquire box from pool
    W->>ISO: isolate --cg --init
    W->>S3: GET test case archive URL
    W->>S3: GET submission archive URL
    W->>ISO: unzip grader.zip, unzip submission.zip
    W->>ISO: Compile (20 s CPU, 60 s wall, 1 GB)
    alt Compilation failed
        W->>W: Result: status "Compile Error"
    else Compiled
        loop Each test case
            W->>ISO: Run with stdin=N.in, stdout+stderr=N.out.actual
            ISO-->>W: Meta file (status, time, cg-mem)
            W->>W: Verdict AC / WA / RE / TLE / XX
        end
    end
    W->>MQ: Store JSON result (task state SUCCESS) in result backend
    W->>BE: POST /v1/submissions/{id}/programming/grade-callback {task_id}
    BE->>MQ: Read task state and result by task_id
    W->>W: Release box to pool
```

### Sandbox and limits

Every compile and run goes through `isolate --run` (`internal/sandboxes/sandbox_isolate.go`) with these options:

| Option | Value | Meaning |
| --- | --- | --- |
| `-b <id>` | Box from the pool | Sandbox instance |
| `--dir=/etc` | always | Exposes `/etc` read-only |
| `-e` | always | Inherits the grader's environment |
| `--cg -p50` | always | Control-group accounting, at most 50 processes or threads |
| `-t<sec>` | time limit | CPU time limit |
| `-x0.5` | with `-t` | 0.5 s extra time before the process is killed |
| `-w<sec>` | wall limit | Wall-clock limit |
| `--cg-mem=<KB>` | memory limit | Memory limit for the whole control group |
| `-k<KB>` | memory limit | Stack limit, same value as the memory limit |
| `-f102400` | always | Files written by the program are limited to 102400 KB |
| `-i`, `-o`, `-r`, `-M` | per run | stdin, stdout, stderr, and meta file |

| Phase | CPU time | Wall time | Memory |
| --- | --- | --- | --- |
| Unzip | none | none | none |
| Compile | 20 s | 60 s | 1 GB (1048576 KB) |
| Test case run | job time limit | job time limit + 5 s | job memory limit |

Downloads happen in the grader process on the host or container. Only `unzip`, compilation, and program runs execute inside isolate.

### Supported languages

Language names are matched exactly. The backend's `programming_languages.name` must use these strings.

| Name | Source extensions | Compile | Run |
| --- | --- | --- | --- |
| `C` | `.c` | `gcc -std=gnu99 -o <main> <all .c files> -O2 -lm` | `./<main>` |
| `C++11` | `.cpp`, `.cc` | `g++ -std=c++11 -o <main> <sources> -O2 -lm` | `./<main>` |
| `C++17` | `.cpp`, `.cc` | `g++ -std=c++17 -o <main> <sources> -O2 -lm` | `./<main>` |
| `C++20` | `.cpp`, `.cc` | `g++ -std=c++20 -o <main> <sources> -O2 -lm` | `./<main>` |
| `Java` | `.java` | `javac <all .java files> && jar cfe <Main>.jar <Main> *.class` | `java -jar <Main>.jar` |
| `Python 3` | `.py` | none (`/bin/true`) | `python3 <main file>` |
| `Haskell` | `.hs` | `ghc --make <main>` | `./<main>` |

`<main>` is the main source file name without its extension. For `blackbox` jobs, every file in the box with a matching extension is passed to the compiler, including files that came from the test case archive. The file contents decide whether a file is a library or the entry point, not where it came from.

For `blackbox_with_builder` jobs (builder `Makefile`):

- Compile runs `make <compile target>` in the box. An empty target runs the Makefile's default target.
- Run uses the job's language run command with the run target as the main file name. For example, `C` with run target `prog` runs `./prog`, and `Java` with run target `Main` runs `java -jar Main.jar`.

The toolchains have to be installed where the grader runs. The Docker image installs `build-essential` only (gcc, g++, make), so only `C`, `C++11/17/20`, and Makefile builds of those work in the published image. The bare-metal `Makefile` also installs OpenJDK 17 and Python 3. Nothing in this repository installs GHC.

### Verdicts

`internal/orchestrator/evaluator/simple_evaluator.go` maps the isolate meta file status to a verdict:

| Code | Name | When |
| --- | --- | --- |
| `AC` | Accepted | Exited normally and the output matches the expected output |
| `WA` | Wrong Answer | Exited normally and the output differs |
| `RE` | Runtime Error | Non-zero exit code (isolate `RE`) or killed by a signal (isolate `SG`) |
| `TLE` | Time Limit Exceeded | isolate `TO` (CPU or wall-clock limit) |
| `XX` | Internal Error | isolate internal error, unreadable meta file, or missing output file |
| `CE` | Compilation Error | Defined in the model. A failed compile is reported through the job-level `status` instead (see below). |

Output comparison is byte-exact after converting `\r\n` to `\n`. Trailing whitespace and trailing newlines are significant. Because stderr is written to the same file as stdout, anything the program prints to stderr is part of the compared output. There is no separate memory-limit verdict; a program that exceeds the memory limit is killed and reported as `RE`.

### Result format

The task result is a JSON string:

```json
{
  "is_success": true,
  "status": "Success",
  "error_message": "",
  "testcase_result": [
    {
      "verdict": { "name": "Accepted", "code": "AC" },
      "has_error_message": false,
      "error_message": "",
      "input_filename": "1.in",
      "output_filename": "1.out",
      "time_to_run_ms": 3,
      "memory_used_kb": 1024
    }
  ]
}
```

| `status` | `is_success` | Meaning |
| --- | --- | --- |
| `Success` | `true` | Compiled; `testcase_result` has one entry per test case, in order |
| `Compile Error` | `false` | Compilation failed, or the language name is unknown; `error_message` holds the compiler output or `Language or builder does not exist.` |
| `Internal error` | `false` | The compiler or engine could not be prepared |

`time_to_run_ms` and `memory_used_kb` (isolate `cg-mem`) are filled only for `AC` and `WA` results. For `RE` and `TLE`, `error_message` holds the program's captured output.

### Observability

- Logs are JSON on stdout. The level is `info` when `GRADER_ENV=production` and `debug` otherwise. If `MONITORING_LOKI_URL` is set, info-level and higher entries are also pushed to Loki with a `grader_name` field.
- If `MONITORING_INFLUX_URL` is set, the grader writes a `grader_metrics` point every 500 ms. Its fields are `idle_workers`, `busy_workers`, `system_cpu_percent`, and `system_mem_percent`, and its tags are `grader_name` and `grader_environment`.

## Prerequisites

- **Linux** with cgroup v2. isolate does not run on macOS or Windows. On those systems use the Docker image on a Linux host or VM.
- **Go 1.24.2** for building from source.
- **isolate** installed at `/usr/local/bin/isolate`. The path is hard-coded.
- **Toolchains** for the languages you grade (`gcc`/`g++`/`make`, `openjdk-17-jdk`, `python3`, `ghc`), plus `unzip`, which the grader uses to unpack archives.
- **RabbitMQ** reachable at `QUEUE_BROKER_URL`. The backend repository ships one in `deployments/docker/docker-compose.local.yaml` (user `ceruvia`, password `ceruvia`, port 5672).
- **The backend**, to publish jobs and receive callbacks. A single job can be sent without it using `cmd/send`; see below.
- Root privileges. The provided systemd units run the grader as `root`.

## Setup and running locally

### 1. Start RabbitMQ

From the backend repository:

```bash
docker compose -f deployments/docker/docker-compose.local.yaml up -d
```

### 2. Configure

```bash
cp .env.example .env
```

Then edit `.env`. Set `QUEUE_BROKER_URL` and `QUEUE_RESULT_URL` to match your broker, for example `amqp://ceruvia:ceruvia@localhost:5672/`. Set `QUEUE_NAME` to the backend's value (`grading_queue`). Set `BACKEND_API_TOKEN` to the backend's `APP_API_KEY`. See [ENVIRONMENT.md](ENVIRONMENT.md).

### 3a. Run on a Linux host

Install dependencies and isolate once. These targets use `apt-get` and `sudo`, so they assume Debian or Ubuntu.

```bash
make install-dependency   # build-essential, openjdk-17-jdk, python3, unzip, libcap-dev, libsystemd-dev, ...
make install-isolate      # clones github.com/ioi/isolate (default branch), builds and installs it
```

Then build and run as root:

```bash
make build                # go mod download && go build -o grader cmd/server/main.go
sudo ./grader
```

The grader reads `.env` from the current working directory unless `LOAD_DOTENV=false`. On start it logs `Launching worker...` with the worker count.

### 3b. Run with Docker

The image boots systemd. systemd starts `isolate.service`, captures the container environment, and runs the grader. It needs a privileged container with the host cgroup tree:

```bash
docker build -t ceruvia-grader .
docker run -d --name ceruvia-grader \
  --privileged --cgroupns=host \
  -v /sys/fs/cgroup:/sys/fs/cgroup:rw \
  -e LOAD_DOTENV=false \
  -e QUEUE_NAME=grading_queue \
  -e QUEUE_BROKER_URL=amqp://ceruvia:ceruvia@host.docker.internal:5672/ \
  -e QUEUE_RESULT_URL=amqp://ceruvia:ceruvia@host.docker.internal:5672/ \
  -e BACKEND_API_TOKEN=<backend APP_API_KEY> \
  -e BACKEND_CALLBACK_ENDPOINT='http://host.docker.internal:8080/v1/submissions/{submission_id}/programming/grade-callback' \
  --add-host=host.docker.internal:host-gateway \
  ceruvia-grader
docker exec ceruvia-grader journalctl -u grader-docker.service -f
```

Only variables prefixed `GRADER_`, `QUEUE_`, `BACKEND_`, `MONITORING_`, and `LOAD_DOTENV` reach the grader inside the container. See [DEPLOY.md](DEPLOY.md) for details.

### Sending a test job

`cmd/send` publishes one `blackbox` job (a C program, `ganjilgenap.c`, with 10 test cases from public sample archives). It then waits for the result and prints it. It uses the same configuration as the grader:

```bash
go run ./cmd/send
```

The callback to `BACKEND_CALLBACK_ENDPOINT` is still attempted. Without a running backend it fails and the failure is logged.

### Makefile targets

| Target | What it does |
| --- | --- |
| `all` | `install-dependency`, `install-isolate`, `build`, `install` in order |
| `install-dependency` | `apt-get install` of build tools, `libcap-dev`, `libsystemd-dev`, `openjdk-17-jdk`, `python3`, `zip`, `unzip` |
| `install-isolate` | Clones isolate's default branch to `/tmp/isolate`, builds and installs it, removes the clone |
| `build` | `go mod download` and `go build -o grader cmd/server/main.go` |
| `install` | Copies `grader` to `/usr/local/bin/grader` and `deployment/services/grader.service` to `/etc/systemd/system/` |
| `clean` | Removes the `grader` binary |

`install-isolate` builds isolate's current default branch, while the Dockerfile pins `v2.1` because isolate v2.3 and later need `libseccomp`, which `install-dependency` does not install.

## Configuration

All configuration is read from environment variables. See [ENVIRONMENT.md](ENVIRONMENT.md) for the full reference.

## Testing

The tests are standard Go tests next to the code. Their fixtures live in `tests/`:

| Fixture directory | Used by | Contents |
| --- | --- | --- |
| `tests/c_test/e2e/` | `internal/orchestrator/acceptance_test.go` | C programs with Makefiles and `.in`/`.out` files. The prefix gives the expected outcome: `scs_` all or partly accepted, `ce_` compile error, `re_` runtime error, `tle_` time limit |
| `tests/java_test/`, `tests/makefile_test/` | acceptance tests | Java programs, with and without Makefiles |
| `tests/haskell_test/` | acceptance tests | Haskell programs built with a Makefile |
| `tests/c_test/{adt,uncompileable,runtimeerror,hello,hello_binary}` | compiler, engine, and sandbox tests | Compilable, uncompilable, and crashing C sources |
| `tests/python3_test/` | none currently | Python fixture |
| `tests/sandbox/parse/` | `TestParser` | Sample isolate meta files (`success`, `re`, `sigkill`, `tle`) |
| `tests/copy/`, `tests/not_commited/` | sandbox and engine tests | Scratch directories for file-copy tests (`tests/not_commited` is git-ignored) |

| Test file | Needs isolate | Needs network | Notes |
| --- | --- | --- | --- |
| `internal/languages/{clang,javalang,pylang,builder}/*_test.go` | no | no | Command construction per language |
| `internal/sandboxes/sandbox_test.go` | `TestIsolate` only | no | `TestRedirections` and `TestParser` are file-only |
| `internal/orchestrator/compilers/compiler_test.go` | yes | no | Uses boxes 200–203 |
| `internal/orchestrator/engines/engine_test.go` | partly | no | Uses boxes 300–302 |
| `internal/orchestrator/acceptance_test.go` | yes | no | End-to-end grading of the fixtures above. Uses boxes 900–951, needs `gcc`, `make`, `javac`/`jar`/`java`, and `ghc` |
| `internal/tasks/tasks_test.go` | yes | yes | Downloads sample archives over HTTPS. Uses boxes 400–401 |

`internal/helper/command/_test.go` begins with an underscore, so `go test` ignores it.

Run everything on a Linux machine with isolate and the toolchains installed. The isolate tests create and clean up boxes, so run them as a user allowed to do that (root in the provided setup):

```bash
sudo go test ./...
```

Run only the tests that need neither isolate nor the network:

```bash
go test ./internal/languages/...
go test ./internal/sandboxes -run 'TestRedirections|TestParser'
```

CI does not run the tests. It runs `go build ./...` and `go vet ./...` before building the image.

## Deployment

See [DEPLOY.md](DEPLOY.md) for the Docker image, systemd units, CI image publishing, scaling, and the current deployment status. The combined full-stack VPS guide is [Backend/DEPLOY.md](https://github.com/Ceruvia/Backend/blob/dev/DEPLOY.md).

## Team

Developed and maintained by the Ceruvia team.
