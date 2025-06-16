GOLANG_VERSION := 1.24.2
GOPATH         := /go
PATH           := /usr/local/go/bin:$(GOPATH)/bin:$(PATH)

.PHONY: all install-dependency install-go install-isolate build install-services clean

all: install-dependency install-isolate build install

install-dependency:
	sudo apt-get update && \
	sudo apt-get install -y --no-install-recommends \
	  ca-certificates wget git build-essential libcap-dev libsystemd-dev pkg-config zip openjdk-8-jdk && \
	sudo rm -rf /var/lib/apt/lists/*

install-go:
	wget -O go.tar.gz https://go.dev/dl/go$(GOLANG_VERSION).linux-amd64.tar.gz && \
	sudo tar -C /usr/local -xzf go.tar.gz && \
	rm go.tar.gz

install-isolate:
	git clone https://github.com/ioi/isolate.git /tmp/isolate && \
	cd /tmp/isolate && \
	make -j"$(shell nproc)" isolate && \
	sudo make -j"$(shell nproc)" install && \
	rm -rf /tmp/isolate

build:
	@echo "→ Building grader…"
	mkdir -p /grader
	cp -R . /grader
	cd /grader && \
	export GOPATH=$(GOPATH) PATH=$(PATH) && \
	go mod download && \
	go build -o /usr/local/bin/grader cmd/server/main.go

install:
	sudo cp deployment/services/grader.service.service

clean:
	sudo rm -rf grader