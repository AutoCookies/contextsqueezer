#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_NATIVE_DIR="$ROOT_DIR/build/native"
BUILD_GO_DIR="$ROOT_DIR/build/bin"

cmake -S "$ROOT_DIR/native" -B "$BUILD_NATIVE_DIR" -DCMAKE_BUILD_TYPE=Release
cmake --build "$BUILD_NATIVE_DIR" --config Release

mkdir -p "$BUILD_GO_DIR"
go build -o "$BUILD_GO_DIR/contextsqueeze" "$ROOT_DIR/cmd/contextsqueeze"
