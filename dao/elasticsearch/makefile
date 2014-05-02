.PHONY: build test clean

default: build

build:
	go build

test: build
	go test $(GOTEST_FLAGS)

clean:
	go clean
