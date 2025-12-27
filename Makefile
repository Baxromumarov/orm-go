.PHONY: all test bench

all: test

test:
	go test -v ./...

bench:
	go test -bench . -benchmem ./...
