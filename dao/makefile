.PHONY: build test clean

default: build test

build:
	cd elasticsearch && make build
	go build

test:
	cd elasticsearch && make test
	go test

clean:
	cd elasticsearch && make clean
	go clean
