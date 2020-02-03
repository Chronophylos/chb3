VERSION=$(shell git describe --tags --abbrev=0)
DATE=$(shell date -Iseconds)
COMMIT=$(shell git rev-parse --short HEAD)

LDFLAGS=-ldflags "-X github.com/chronophylos/chb3/buildinfo.version=${VERSION} -X github.com/chronophylos/chb3/buildinfo.buildDate=${DATE} -X github.com/chronophylos/chb3/buildinfo.commit=${COMMIT}"

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
