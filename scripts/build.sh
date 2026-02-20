#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build/native"
BIN_DIR="${ROOT_DIR}/build/bin"

cmake -S "${ROOT_DIR}/native" -B "${BUILD_DIR}" -DCMAKE_BUILD_TYPE=Release
cmake --build "${BUILD_DIR}" --config Release

mkdir -p "${BIN_DIR}"
go build -o "${BIN_DIR}/contextsqueeze" "${ROOT_DIR}/cmd/contextsqueeze"
