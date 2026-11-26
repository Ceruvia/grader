# Grader environment variables

The grader reads all configuration from environment variables in `internal/config/config.go`. Unless `LOAD_DOTENV` is `false`, it first loads a `.env` file from the current working directory with `godotenv`. Variables already set in the process environment take precedence over `.env`. If the file is missing while `LOAD_DOTENV` is not `false`, the grader exits with `Error loading .env file`.

Integer and boolean values that fail to parse fall back silently to their defaults.

Start from the template:

```bash
cp .env.example .env
```

## Reference

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `LOAD_DOTENV` | No | `true` | Load `.env` from the working directory before reading variables. Set to `false` in containers and anywhere the environment is supplied directly. Parsed with Go's `strconv.ParseBool`. |
| `GRADER_NAME` | No | Host name (`grader` if the host name cannot be read) | Identifier for this grader instance. Added as `grader_name` to Loki log entries and InfluxDB metrics. |
| `GRADER_ENV` | No | `development` | `production` sets the log level to `info`. Any other value sets it to `debug`. `testing` also prints each pushed metrics sample to stdout. Also sent as the `grader_environment` InfluxDB tag. |
| `GRADER_WORKER_COUNT` | No | `20` | Number of jobs graded concurrently by this process. It sets the isolate box pool size (box IDs `1..N`), the Machinery worker concurrency, and the AMQP prefetch count. Must be greater than 0. |
| `QUEUE_BROKER_URL` | Yes in practice | `amqp://guest:guest@localhost:5672/` | AMQP URL of the RabbitMQ broker that jobs are consumed from. |
| `QUEUE_RESULT_URL` | Yes in practice | `amqp://guest:guest@localhost:5672/` | AMQP URL for Machinery's result backend, where task states and results are stored. The backend must use the same broker. |
| `QUEUE_NAME` | Yes in practice | `ceruvia_submissions` | Queue to consume from. Must equal the backend's `QUEUE_NAME`, because the backend publishes with that name as the routing key. The backend and the staging deployment use `grading_queue`. |
| `QUEUE_RESULT_TTL` | No | `36000` | How long stored task results are kept, in seconds (Machinery `ResultsExpireIn`). |
| `BACKEND_API_TOKEN` | Yes in practice | `token` | Sent as the `X-API-Key` header on the completion callback. Must equal the backend's `APP_API_KEY`. |
| `BACKEND_CALLBACK_ENDPOINT` | Yes in practice | `http://localhost:8080/v1/submissions/{submission_id}/programming/grade-callback` | URL the grader POSTs `{"task_id": "<uuid>"}` to after each job. The literal `{submission_id}` is replaced with the job's submission ID. |
| `MONITORING_LOKI_URL` | No | empty (disabled) | Loki push URL. When set, logs at `info` level and above are also shipped to Loki. |
| `MONITORING_INFLUX_URL` | No | empty (disabled) | InfluxDB v2 URL. When set, the grader writes a `grader_metrics` point every 500 ms. |
| `MONITORING_INFLUX_TOKEN` | When InfluxDB is enabled | empty | InfluxDB API token. |
| `MONITORING_INFLUX_ORGANIZATION` | When InfluxDB is enabled | empty | InfluxDB organization. |
| `MONITORING_INFLUX_BUCKET` | When InfluxDB is enabled | empty | InfluxDB bucket. |

"Yes in practice" means the code has a default, but that default only works for a single-host development setup with the backend on `localhost:8080`, RabbitMQ's `guest` account, and a placeholder API token. Set these values in every real deployment.

## Notes

- **`.env.example` vs. backend.** `.env.example` sets `QUEUE_NAME=submissions`, the code default is `ceruvia_submissions`, and the backend uses `grading_queue`. Whichever value you choose, the grader and the backend must agree. Otherwise jobs sit in a queue with no consumer.
- **`.env.example` is incomplete.** It omits `LOAD_DOTENV`, `BACKEND_API_TOKEN`, `BACKEND_CALLBACK_ENDPOINT`, and the `MONITORING_*` variables. Add them to your `.env` as needed.
- **Docker image.** Inside the container the grader is started by systemd, which does not pass on the container environment. The `dump-env.service` unit copies only variables matching `GRADER_*`, `QUEUE_*`, `BACKEND_*`, `MONITORING_*`, and `LOAD_DOTENV` from PID 1 into `/etc/docker_env`, which `grader-docker.service` loads. Variables with other names are not visible to the grader. Set `LOAD_DOTENV=false` in the container. Otherwise the grader looks for `/grader/.env`, and that file exists only if a `.env` was present in the build context.
- **Bare-metal systemd.** `deployment/services/grader.service` has no `EnvironmentFile`. It runs from `/opt/grader`, so configuration comes from `/opt/grader/.env` with the default `LOAD_DOTENV=true`.
- **Sandbox environment.** isolate runs programs with `-e`, so submitted code can read the grader's environment, including `BACKEND_API_TOKEN` and the broker URLs.
