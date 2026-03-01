.PHONY: all build build-static test clean install

VERSION ?= $(shell cat VERSION 2>/dev/null || echo "1.0.0")
LD_FLAGS = -X contextsqueezer/internal/version.Version=$(VERSION)

all: build

build:
	./scripts/build.sh

build-static:
	./scripts/build_static.sh

test:
	./scripts/test.sh

clean:
	rm -rf build/ build_static/ dist/ bin/
	find . -name "cpu.pprof" -delete
	find . -name "heap.pprof" -delete

install: build
	install -m 0755 build/bin/contextsqueeze /usr/local/bin/

# Helper for cross-compilation targeting edge devices (ARM64)
build-arm64-static:
	GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc \
	CXX=aarch64-linux-gnu-g++ ./scripts/build_static.sh
