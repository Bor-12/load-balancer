APP_NAME := cloudbalancer
APP_BINARY := $(APP_NAME).exe
CMD_PATH := ./cmd/cloudbalancer
GO_PACKAGES := ./...

.PHONY: fmt vet test test-race coverage build verify run

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet $(GO_PACKAGES)

test:
	go test $(GO_PACKAGES)

test-race:
	go test -race $(GO_PACKAGES)

coverage:
	go test -coverprofile=coverage.out $(GO_PACKAGES)
	go tool cover -html=coverage.out -o coverage.html

build:
	if not exist bin mkdir bin
	go build -o bin/$(APP_BINARY) $(CMD_PATH)

verify: fmt vet test test-race build

run:
	go run $(CMD_PATH)
