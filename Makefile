VERSION=$(shell git describe --tags --abbrev=0)
DATE=$(shell date -Iseconds)
COMMIT=$(shell git rev-parse --short HEAD)

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildDate=${DATE} -X main.GitCommit=${COMMIT}"

all: clean build

run: build
	./chb3 --debug

build:
	go build $(LDFLAGS) -o chb3 .

test:
	go test ./...

lint:
	golangci-lint run


.PHONY: clean test run all
