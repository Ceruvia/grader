# Deploying the grader

This document covers the grader on its own: the Docker image, the systemd units in `deployment/`, the CI workflow that publishes the image, and how to scale. The combined full-stack VPS guide is [Backend/DEPLOY.md](https://github.com/Ceruvia/Backend/blob/dev/DEPLOY.md). It covers the registry login, the backend, frontend, and RabbitMQ setup, rollbacks, and staging, and this document does not repeat those steps.

## Current deployment status

| Environment | Grader deployed? | Source |
| --- | --- | --- |
| Production (VPS) | **No** | `Backend/deployments/docker/docker-compose.prod.yaml` defines `backend`, `frontend`, and `cloudflared` only. There is no RabbitMQ service and no grader. Backend/DEPLOY.md, section 5: "Grading is not deployed … Submissions will be accepted and never graded." |
| Staging (same VPS) | **Yes** | `Backend/deployments/docker/docker-compose.staging.yaml` runs `broker` (`rabbitmq:3.13-management`) and `grader` (`ghcr.io/ceruvia/grader:${GRADER_TAG:-staging}`) beside the staging backend and frontend. Configuration is in Backend/DEPLOY.md, section 6.1. |

To grade in production, add a RabbitMQ service and a grader service to the production compose file, configured like staging, and point the production backend's `QUEUE_*` variables at that broker. Until then, production submissions stay ungraded.

## Docker image

The `Dockerfile` builds a single image that runs **systemd as PID 1**. systemd starts isolate's control-group service and then the grader:

1. Base `ubuntu:22.04`. The base image is multi-arch; the comment explains it replaced an amd64-only systemd image. The image sets `container=docker` and `STOPSIGNAL SIGRTMIN+3`, the signal systemd expects for shutdown. Most default systemd unit links (udev, tmpfiles setup, utmp, and the default `*.wants` targets) are removed.
2. Installs `systemd`, `build-essential` (gcc, g++, make), `git`, `wget`, `libcap-dev`, `libsystemd-dev`, `pkg-config`, `zip`, and `unzip`.
3. Installs Go 1.24.2 for the build architecture.
4. Builds and installs isolate from tag `v2.1`. It is pinned because isolate v2.3 and later need `libseccomp`.
5. Copies the whole repository to `/grader` and builds `/usr/local/bin/grader` from `cmd/server/main.go`.
6. Installs and enables `isolate.service`, `dump-env.service`, and `grader-docker.service`.
7. Declares `VOLUME /sys/fs/cgroup` and runs `/sbin/init`.

### Runtime requirements

isolate creates Linux namespaces and control groups, so the container needs elevated access. The staging compose service uses:

```yaml
privileged: true
cgroup: host
volumes:
  - /sys/fs/cgroup:/sys/fs/cgroup:rw
```

The host must use cgroup v2. With `docker run`, the equivalent flags are `--privileged --cgroupns=host -v /sys/fs/cgroup:/sys/fs/cgroup:rw`. A privileged container has broad access to the host, so give the grader its own host or VM if untrusted submissions are a concern.

### Language toolchains in the image

The image installs only `build-essential`. Submissions in `C`, `C++11`, `C++17`, and `C++20`, and Makefile builds that use those compilers, work. `Java`, `Python 3`, and `Haskell` submissions fail in this image because `javac`/`java`, `python3`, and `ghc` are not installed. Supporting them means adding those packages to the Dockerfile.

### Build context

The repository has no `.dockerignore`, and `COPY . /grader` copies the whole build context. Do not build the image from a working copy that contains a real `.env`, because the file would be baked into the image. CI builds from a clean checkout, where `.env` does not exist because it is git-ignored.

## systemd units (`deployment/services/`)

| Unit | Used by | What it does |
| --- | --- | --- |
| `dump-env.service` | Docker image | Oneshot, runs before the grader. Copies variables matching `GRADER_*`, `QUEUE_*`, `BACKEND_*`, `MONITORING_*`, and `LOAD_DOTENV` from PID 1's environment (`/proc/1/environ`) into `/etc/docker_env`. This is how `docker run -e` and compose `env_file` values reach a service started by systemd. |
| `grader-docker.service` | Docker image | Runs `/usr/local/bin/grader` as root from `/grader` with `EnvironmentFile=/etc/docker_env`. Before starting, it waits until `isolate.service` is active and runs `isolate --cg -b 999 --cleanup` and `--init` to check that isolate works. Restarts always, 5 s delay. |
| `grader.service` | Bare-metal install (`make install`) | Same command and isolate check without the wait loop or `EnvironmentFile`. Runs from `/opt/grader`, so configuration comes from `/opt/grader/.env`. Restarts always, 5 s delay. |

`isolate.service` is not in this repository. It is installed by isolate's own `make install` and sets up the control-group hierarchy isolate uses.

In the container, view the grader's logs with:

```bash
docker exec <container> journalctl -u grader-docker.service -f
```

## Bare-metal install

On a Debian or Ubuntu host with cgroup v2:

```bash
make all                       # install-dependency, install-isolate, build, install
sudo mkdir -p /opt/grader
sudo cp .env /opt/grader/.env  # edit first; see ENVIRONMENT.md
sudo systemctl daemon-reload
sudo systemctl enable --now isolate.service grader.service
```

`make install-isolate` builds isolate's default branch, not `v2.1` as the Dockerfile does. If that branch needs `libseccomp` (v2.3 and later), install `libseccomp-dev` first, or check out `v2.1` by hand. The bare-metal dependency list also installs `openjdk-17-jdk` and `python3`. GHC is not installed by either path.

## CI: `.github/workflows/deploy.yml`

The workflow runs on every push to `main` or `staging` and on manual dispatch (`workflow_dispatch`). It has a single job:

1. Check out the code. Set up Go from `go.mod`, with module caching.
2. `go build ./...`
3. `go vet ./...`
4. Log in to `ghcr.io` with the workflow's `GITHUB_TOKEN` (`packages: write`).
5. Build the `Dockerfile` with Buildx, using the GitHub Actions cache, and push `ghcr.io/ceruvia/grader` with these tags:

| Branch | Tags |
| --- | --- |
| `main` | `latest`, `main`, `sha-<full commit SHA>` |
| `staging` | `staging`, `sha-<full commit SHA>` |

No `platforms` are set, so the image is built for the runner's architecture (linux/amd64). Tests are not run in CI. The workflow only publishes the image. It does not deploy anything. The VPS pulls the new image when someone runs `docker compose pull && docker compose up -d` there, as described in Backend/DEPLOY.md.

To deploy a specific build on staging, pin the tag:

```bash
GRADER_TAG=sha-<commit> docker compose -p ceruvia-staging -f docker-compose.staging.yaml up -d
```

## Scaling

Grading capacity is set manually. Nothing scales automatically.

### Workers per process: `GRADER_WORKER_COUNT`

One grader process grades up to `GRADER_WORKER_COUNT` jobs at once. The value sets the size of the isolate box pool, the Machinery worker concurrency, and the AMQP prefetch count. Further jobs wait in RabbitMQ. Size it against memory: each concurrent job can use up to 1 GB while compiling, plus the question's memory limit while running. Staging uses `GRADER_WORKER_COUNT=3` on a VM with about 4 GB shared with production. Change the value in the env file and recreate the container:

```bash
docker compose -p ceruvia-staging -f docker-compose.staging.yaml up -d grader
```

### Number of grader replicas

Every grader consumes from the same `QUEUE_NAME`, so several grader processes can share one broker, and RabbitMQ distributes jobs among them. Results and callbacks are keyed by task UUID, so it does not matter which replica handles a job. To add a replica:

- **On another host:** run the same image (or a bare-metal install) with the same `QUEUE_*` and `BACKEND_*` values and a distinct `GRADER_NAME`. The broker and the callback URL must be reachable from that host.
- **On the same host:** the staging compose service sets a fixed `container_name`, so `docker compose up --scale grader=N` cannot create more than one. Define an additional service with its own `container_name` and `GRADER_NAME` instead. Each grader container numbers its isolate boxes from 1 and shares the host cgroup tree (`cgroup: host`). Check that the replicas' isolate control groups do not collide before you rely on this. Raising `GRADER_WORKER_COUNT` in a single container is the simpler way to add capacity on one host.

After scaling, check that the queue has the expected number of consumers:

```bash
docker compose -p ceruvia-staging -f docker-compose.staging.yaml exec broker \
  rabbitmqctl list_queues name messages consumers
```

## Configuration

See [ENVIRONMENT.md](ENVIRONMENT.md). The staging values are in Backend/DEPLOY.md, section 6.1 (`grader.staging.env`).
