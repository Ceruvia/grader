# ubuntu:22.04 is multi-arch; jrei/systemd-ubuntu only ships amd64.
FROM ubuntu:22.04

ENV container=docker
STOPSIGNAL SIGRTMIN+3

# 1) Install build/runtime deps, Go, isolate, grader…
RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
      systemd systemd-sysv \
      ca-certificates wget git build-essential libcap-dev libsystemd-dev pkg-config zip unzip && \
    rm -f /lib/systemd/system/multi-user.target.wants/* \
      /etc/systemd/system/*.wants/* \
      /lib/systemd/system/local-fs.target.wants/* \
      /lib/systemd/system/sockets.target.wants/*udev* \
      /lib/systemd/system/sockets.target.wants/*initctl* \
      /lib/systemd/system/sysinit.target.wants/systemd-tmpfiles-setup* \
      /lib/systemd/system/systemd-update-utmp* && \
    rm -rf /var/lib/apt/lists/*

# 2) Install Go
ENV GOLANG_VERSION=1.24.2 \
    GOPATH=/go \
    PATH=/usr/local/go/bin:/go/bin:$PATH
RUN wget -O go.tar.gz https://go.dev/dl/go${GOLANG_VERSION}.linux-$(dpkg --print-architecture).tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz

# 3) Build & install isolate (pinned: v2.3+ requires libseccomp)
RUN git clone --depth 1 --branch v2.1 https://github.com/ioi/isolate.git /tmp/isolate && \
    cd /tmp/isolate && \
    make -j"$(nproc)" isolate && \
    make -j"$(nproc)" install \
    && rm -rf /tmp/isolate

# 4) Compile your grader
WORKDIR /grader
COPY . /grader
RUN go mod download && \
    go build -o /usr/local/bin/grader cmd/server/main.go

# 5) Enable your systemd units (make sure you COPY your .service files into /etc/systemd/system/)
COPY deployment/services/grader-docker.service /etc/systemd/system/
COPY deployment/services/dump-env.service /etc/systemd/system/
RUN systemctl enable isolate.service dump-env.service grader-docker.service

VOLUME [ "/sys/fs/cgroup" ]

CMD ["/sbin/init"]
