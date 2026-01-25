.PHONY: all build test clean

all: build test

build:
	go build -o fsrecov fsrecov.go fat32.go

test:
	go test -v

clean:
	rm -f fsrecov
	go clean -testcache
