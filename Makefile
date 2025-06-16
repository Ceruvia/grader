GOLANG_VERSION := 1.24.2
GOPATH         := /go
PATH           := /usr/local/go/bin:$(GOPATH)/bin:$(PATH)

.PHONY: all install-dependency install-isolate build clean

all: install-dependency install-isolate build install

install-dependency:
	sudo apt-get update && \
	sudo apt-get install -y --no-install-recommends \
	  software-properties-common ca-certificates wget git libcap-dev libsystemd-dev pkg-config \
	  build-essential zip unzip openjdk-17-jdk python3 && \
	sudo rm -rf /var/lib/apt/lists/*

install-isolate:
	git clone https://github.com/ioi/isolate.git /tmp/isolate && \
	cd /tmp/isolate && \
	make -j"$(shell nproc)" isolate && \
	sudo make -j"$(shell nproc)" install && \
	rm -rf /tmp/isolate

build:
	@echo "→ Building grader…"
	go mod download && \
	go build -o grader cmd/server/main.go

install:
	sudo cp grader /usr/local/bin/grader
	sudo cp deployment/services/grader.service /etc/systemd/system/

clean:
	sudo rm -rf grader