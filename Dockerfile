# ubuntu:22.04 is multi-arch; jrei/systemd-ubuntu only ships amd64.
# Stages: base (systemd + isolate + language toolchains), build (grader binary),
# test (base + Go, source in /src), runtime (default, no Go toolchain).
FROM ubuntu:22.04 AS base

ENV container=docker
STOPSIGNAL SIGRTMIN+3

# 1) Install build/runtime deps
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      systemd systemd-sysv \
      ca-certificates wget git build-essential libcap-dev libsystemd-dev pkg-config zip unzip \
      python3 ghc && \
    rm -f /lib/systemd/system/multi-user.target.wants/* \
      /etc/systemd/system/*.wants/* \
      /lib/systemd/system/local-fs.target.wants/* \
      /lib/systemd/system/sockets.target.wants/*udev* \
      /lib/systemd/system/sockets.target.wants/*initctl* \
      /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
      /lib/systemd/system/systemd-update-utmp* && \
    rm -rf /var/lib/apt/lists/*

# 1b) Debian symlinks ghc's package db to /var/lib/ghc, which isolate boxes do not bind (only /usr etc.).
RUN rm /usr/lib/ghc/package.conf.d && cp -a /var/lib/ghc/package.conf.d /usr/lib/ghc/package.conf.d

# 2) Build & install isolate (pinned: v2.3+ requires libseccomp)
RUN git clone --depth 1 --branch v2.1 https://github.com/ioi/isolate.git /tmp/isolate && \
    cd /tmp/isolate && \
    make -j"$(nproc)" isolate && \
    make -j"$(nproc)" install \
    && rm -rf /tmp/isolate

# 3) Enable your systemd units
COPY deployment/services/grader-docker.service /etc/systemd/system/
COPY deployment/services/dump-env.service /etc/systemd/system/
RUN systemctl enable isolate.service dump-env.service grader-docker.service

VOLUME [ "/sys/fs/cgroup" ]

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /out/grader ./cmd/server

# Test image: `docker build --target test`. Source lives in /src.
FROM base AS test
COPY --from=golang:1.26 /usr/local/go /usr/local/go
ENV PATH=/usr/local/go/bin:$PATH
WORKDIR /src
COPY . /src
CMD ["/sbin/init"]

FROM base AS runtime
WORKDIR /grader
COPY --from=build /out/grader /usr/local/bin/grader
CMD ["/sbin/init"]
