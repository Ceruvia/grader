FROM jrei/systemd-ubuntu:22.04

RUN apt-get -y update && apt-get -y upgrade
RUN apt-get install -y --reinstall ca-certificates

# Install Go
RUN apt-get install -y wget && \
    wget -O go.tar.gz https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz

# Set Go environment variables
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install isolate dependencies and build isolate
RUN set -xe && \
    apt-get install -y --no-install-recommends git build-essential libcap-dev libsystemd-dev pkg-config && \
    rm -rf /var/lib/apt/lists/* && \
    git clone https://github.com/ioi/isolate.git /tmp/isolate && \
    cd /tmp/isolate && \
    # git checkout v1.8.1 && \
    make -j$(nproc) isolate && \
    make -j$(nproc) install && \
    rm -rf /tmp/*

# Enable isolate service (will be started when container runs)
RUN systemctl enable isolate.service

# Ensure cgroups v2 is properly configured
RUN mkdir -p /etc/systemd/system.conf.d && \
    echo "[Manager]" > /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultCPUAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultMemoryAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf && \
    echo "DefaultIOAccounting=yes" >> /etc/systemd/system.conf.d/cgroup.conf

# Set proper cgroup mount options
RUN echo 'GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"' >> /etc/default/grub || true

# Create working directory for the app
WORKDIR /grader

# Copy your Go application files
COPY . /grader/

# Download Go dependencies and build the application
RUN go mod download && \
    go build -o grader cmd/server/main.go

# Create a systemd service for the grader
RUN echo '[Unit]' > /etc/systemd/system/grader.service && \
    echo 'Description=Code Grader Service' >> /etc/systemd/system/grader.service && \
    echo 'After=isolate.service' >> /etc/systemd/system/grader.service && \
    echo 'Wants=isolate.service' >> /etc/systemd/system/grader.service && \
    echo '' >> /etc/systemd/system/grader.service && \
    echo '[Service]' >> /etc/systemd/system/grader.service && \
    echo 'Type=simple' >> /etc/systemd/system/grader.service && \
    echo 'User=root' >> /etc/systemd/system/grader.service && \
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


# Enable the grader service to start automatically
RUN systemctl enable grader.service

# Keep the existing start-grader.sh script for manual use if needed
RUN echo '#!/bin/bash' > /usr/local/bin/start-grader.sh && \
    echo 'set -e' >> /usr/local/bin/start-grader.sh && \
    echo '' >> /usr/local/bin/start-grader.sh && \
    echo '# Function to check if systemd is running' >> /usr/local/bin/start-grader.sh && \
    echo 'wait_for_systemd() {' >> /usr/local/bin/start-grader.sh && \
    echo '    echo "Waiting for systemd to be ready..."' >> /usr/local/bin/start-grader.sh && \
    echo '    for i in {1..60}; do' >> /usr/local/bin/start-grader.sh && \
    echo '        if systemctl is-system-running >/dev/null 2>&1 || [[ "$(systemctl is-system-running 2>/dev/null)" =~ ^(running|degraded)$ ]]; then' >> /usr/local/bin/start-grader.sh && \
    echo '            echo "Systemd is ready"' >> /usr/local/bin/start-grader.sh && \
    echo '            return 0' >> /usr/local/bin/start-grader.sh && \
    echo '        fi' >> /usr/local/bin/start-grader.sh && \
    echo '        echo "Waiting for systemd... ($i/60)"' >> /usr/local/bin/start-grader.sh && \
    echo '        sleep 2' >> /usr/local/bin/start-grader.sh && \
    echo '    done' >> /usr/local/bin/start-grader.sh && \
    echo '    echo "Timeout waiting for systemd"' >> /usr/local/bin/start-grader.sh && \
    echo '    return 1' >> /usr/local/bin/start-grader.sh && \
    echo '}' >> /usr/local/bin/start-grader.sh && \
    echo '' >> /usr/local/bin/start-grader.sh && \
    echo '# Wait for systemd to be ready' >> /usr/local/bin/start-grader.sh && \
    echo 'wait_for_systemd' >> /usr/local/bin/start-grader.sh && \
    echo '' >> /usr/local/bin/start-grader.sh && \
    echo '# Start isolate service' >> /usr/local/bin/start-grader.sh && \
    echo 'echo "Starting isolate service..."' >> /usr/local/bin/start-grader.sh && \
    echo 'systemctl start isolate.service' >> /usr/local/bin/start-grader.sh && \
    echo 'systemctl status isolate.service --no-pager' >> /usr/local/bin/start-grader.sh && \
    echo '' >> /usr/local/bin/start-grader.sh && \
    echo '# Verify isolate is working' >> /usr/local/bin/start-grader.sh && \
    echo 'echo "Testing isolate..."' >> /usr/local/bin/start-grader.sh && \
    echo 'isolate --cg -b 999 --init' >> /usr/local/bin/start-grader.sh && \
    echo 'isolate --cg -b 999 --cleanup' >> /usr/local/bin/start-grader.sh && \
    echo '' >> /usr/local/bin/start-grader.sh && \
    echo '# Start the grader application' >> /usr/local/bin/start-grader.sh && \
    echo 'echo "Starting grader server..."' >> /usr/local/bin/start-grader.sh && \
    echo 'cd /grader' >> /usr/local/bin/start-grader.sh && \
    echo 'exec ./grader' >> /usr/local/bin/start-grader.sh && \
    chmod +x /usr/local/bin/start-grader.sh

# Use systemd as the main process
CMD ["/sbin/init"]