FROM jrei/systemd-ubuntu:22.04

# Basic updates
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get install -y --reinstall ca-certificates

# Install Go
RUN apt-get install -y wget && \
    wget -O go.tar.gz https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz

# Set Go environment
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install isolate dependencies and build isolate
RUN set -xe && \
    apt-get install -y --no-install-recommends \
        git build-essential libcap-dev libsystemd-dev pkg-config && \
    rm -rf /var/lib/apt/lists/* && \
    git clone https://github.com/ioi/isolate.git /tmp/isolate && \
    cd /tmp/isolate && \
    make -j$(nproc) isolate && \
    make -j$(nproc) install && \
    rm -rf /tmp/isolate
RUN systemctl enable isolate.service

# Ensure cgroups v2 support
RUN mkdir -p /etc/systemd/system.conf.d && \
    echo "[Manager]" > /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultCPUAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultMemoryAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultIOAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf

# Install necessary packages for compiler, interpreters, helper, etc
RUN apt-get -y update && apt-get -y upgrade
RUN apt-get install -y zip

WORKDIR /grader
COPY . /grader/

# Build Go grader app
RUN go mod download && \
    go build -o grader cmd/server/main.go

# ----- dump-env.service (captures env vars set via `docker run -e`) -----
RUN echo '[Unit]' > /etc/systemd/system/dump-env.service && \
    echo 'Description=Capture Docker Environment Variables' >> /etc/systemd/system/dump-env.service && \
    echo 'DefaultDependencies=no' >> /etc/systemd/system/dump-env.service && \
    echo 'Before=grader.service' >> /etc/systemd/system/dump-env.service && \
    echo '' >> /etc/systemd/system/dump-env.service && \
    echo '[Service]' >> /etc/systemd/system/dump-env.service && \
    echo 'Type=oneshot' >> /etc/systemd/system/dump-env.service && \
    echo 'ExecStart=/bin/bash -c "env | grep -E '\''^(GRADER_|QUEUE_)'\'' > /etc/docker_env"' >> /etc/systemd/system/dump-env.service && \
    echo 'RemainAfterExit=true' >> /etc/systemd/system/dump-env.service && \
    echo '' >> /etc/systemd/system/dump-env.service && \
    echo '[Install]' >> /etc/systemd/system/dump-env.service && \
    echo 'WantedBy=multi-user.target' >> /etc/systemd/system/dump-env.service

RUN systemctl enable dump-env.service

# ----- grader.service -----
RUN echo '[Unit]' > /etc/systemd/system/grader.service && \
    echo 'Description=Code Grader Service' >> /etc/systemd/system/grader.service && \
    echo 'After=isolate.service dump-env.service' >> /etc/systemd/system/grader.service && \
    echo 'Wants=isolate.service' >> /etc/systemd/system/grader.service && \
    echo '' >> /etc/systemd/system/grader.service && \
    echo '[Service]' >> /etc/systemd/system/grader.service && \
    echo 'Type=simple' >> /etc/systemd/system/grader.service && \
    echo 'User=root' >> /etc/systemd/system/grader.service && \
    echo 'EnvironmentFile=/etc/docker_env' >> /etc/systemd/system/grader.service && \
    echo 'WorkingDirectory=/grader' >> /etc/systemd/system/grader.service && \
    echo 'ExecStartPre=/bin/bash -c "until systemctl is-active isolate.service; do sleep 1; done"' >> /etc/systemd/system/grader.service && \
    echo 'ExecStartPre=/usr/local/bin/isolate --cg -b 999 --cleanup' >> /etc/systemd/system/grader.service && \
    echo 'ExecStartPre=/usr/local/bin/isolate --cg -b 999 --init' >> /etc/systemd/system/grader.service && \
    echo 'ExecStart=/grader/grader' >> /etc/systemd/system/grader.service && \
    echo 'Restart=always' >> /etc/systemd/system/grader.service && \
    echo 'RestartSec=5' >> /etc/systemd/system/grader.service && \
    echo '' >> /etc/systemd/system/grader.service && \
    echo '[Install]' >> /etc/systemd/system/grader.service && \
    echo 'WantedBy=multi-user.target' >> /etc/systemd/system/grader.service

RUN systemctl enable grader.service

# Boot with systemd
CMD ["/sbin/init"]