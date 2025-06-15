FROM jrei/systemd-ubuntu:22.04

ENV container=docker
STOPSIGNAL SIGRTMIN+3

# 1) Install build/runtime deps, Go, isolate, grader…
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      ca-certificates wget git build-essential libcap-dev libsystemd-dev pkg-config zip && \
    rm -rf /var/lib/apt/lists/*

# 2) Install Go
ENV GOLANG_VERSION=1.21.5 \
    GOPATH=/go \
    PATH=/usr/local/go/bin:/go/bin:$PATH
RUN wget -O go.tar.gz https://go.dev/dl/go${GOLANG_VERSION}.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz

# 3) Build & install isolate
RUN git clone https://github.com/ioi/isolate.git /tmp/isolate && \
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
COPY deployment/services/grader.service /etc/systemd/system/
COPY deployment/services/dump-env.service /etc/systemd/system/
RUN systemctl enable isolate.service dump-env.service grader.service

VOLUME [ "/sys/fs/cgroup" ]

CMD ["/sbin/init"]
