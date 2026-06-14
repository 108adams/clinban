GOCACHE ?= /tmp/clinban-gocache
export GOCACHE

.PHONY: help build install test vet fmt check clean

help:
	@echo "Available targets:"
	@echo "  build    build ./clinban binary"
	@echo "  install  install to \$$GOPATH/bin"
	@echo "  test     go test ./..."
	@echo "  vet      go vet ./..."
	@echo "  fmt      gofmt all Go files in place"
	@echo "  check    vet + test"
	@echo "  clean    remove local ./clinban binary"

build:
	go build -ldflags "-X main.version=$$(git describe --tags --always --dirty)" -o clinban ./cmd/clinban

install:
	go install ./cmd/clinban

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(gofmt -l .)

check: vet test

clean:
	rm -f clinban
