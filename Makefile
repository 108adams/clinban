GOCACHE ?= /tmp/clinban-gocache
export GOCACHE

.PHONY: build install test vet fmt check clean

build:
	go build -o clinban ./cmd/clinban

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
