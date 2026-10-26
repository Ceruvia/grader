# syntax=docker/dockerfile:1
# ubuntu:22.04 is multi-arch; jrei/systemd-ubuntu only ships amd64.
# Stages: isolate (builds isolate, nothing of it but the installed files leaves),
# base (systemd + isolate + language toolchains), build (grader binary),
# test (base + Go, source in /src), runtime (default, no Go toolchain).

# ---------- isolate ----------
# Build tools (git, libcap-dev, libsystemd-dev, pkg-config) live only in this stage.
FROM ubuntu:22.04 AS isolate

# Ubuntu images delete downloaded packages after every apt run. Turn that off so the cache mounts work.
RUN rm -f /etc/apt/apt.conf.d/docker-clean \
    && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      ca-certificates git build-essential libcap-dev libsystemd-dev pkg-config

# Pinned: v2.3+ requires libseccomp. DESTDIR collects the installed files (binaries,
# default.cf, systemd units, box dir) so the next stage copies only those.
RUN git clone --depth 1 --branch v2.1 https://github.com/ioi/isolate.git /tmp/isolate && \
    cd /tmp/isolate && \
    make -j"$(nproc)" isolate && \
    make -j"$(nproc)" install DESTDIR=/out

# ---------- base ----------
FROM ubuntu:22.04 AS base

ENV container=docker
STOPSIGNAL SIGRTMIN+3

RUN rm -f /etc/apt/apt.conf.d/docker-clean \
    && echo 'Binary::apt::APT::Keep-Downloaded-Packages "true";' > /etc/apt/apt.conf.d/keep-cache

# 1) Runtime deps. Student code needs build-essential, python3, ghc, zip and unzip.
# libcap2 and libsystemd0 (isolate runtime libs) come in with systemd. The apt lists
# stay in the cache mount, so they are not in the image.
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      systemd systemd-sysv \
      ca-certificates build-essential zip unzip \
      python3 ghc && \
    rm -f /lib/systemd/system/multi-user.target.wants/* \
      /etc/systemd/system/*.wants/* \
      /lib/systemd/system/local-fs.target.wants/* \
      /lib/systemd/system/sockets.target.wants/*udev* \
      /lib/systemd/system/sockets.target.wants/*initctl* \
      /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
      /lib/systemd/system/systemd-update-utmp*

# 1b) Debian symlinks ghc's package db to /var/lib/ghc, which isolate boxes do not bind (only /usr etc.).
RUN rm /usr/lib/ghc/package.conf.d && cp -a /var/lib/ghc/package.conf.d /usr/lib/ghc/package.conf.d

# 2) isolate v2.1 from the isolate stage (setuid bit and units come along)
COPY --from=isolate /out/ /

# 3) Enable your systemd units
COPY deployment/services/grader-docker.service /etc/systemd/system/
COPY deployment/services/dump-env.service /etc/systemd/system/
RUN systemctl enable isolate.service dump-env.service grader-docker.service

VOLUME [ "/sys/fs/cgroup" ]

# ---------- build ----------
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
# Only the Go sources, so a change under tests/ or deployment/ does not rebuild the binary.
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/grader ./cmd/server

# ---------- test ----------
# Test image: `docker build --target test`. Source lives in /src.
FROM base AS test
COPY --from=golang:1.26 /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH
WORKDIR /src
COPY . /src
CMD ["/sbin/init"]

# ---------- runtime ----------
FROM base AS runtime
WORKDIR /grader
COPY --from=build /out/grader /usr/local/bin/grader
CMD ["/sbin/init"]
