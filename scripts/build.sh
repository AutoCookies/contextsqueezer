#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NATIVE_BUILD_DIR="${ROOT_DIR}/build/native/cmake"

cmake -S "${ROOT_DIR}/native" -B "${NATIVE_BUILD_DIR}" -DCMAKE_BUILD_TYPE=Release
cmake --build "${NATIVE_BUILD_DIR}" --config Release

mkdir -p "${ROOT_DIR}/build/bin"
go build -o "${ROOT_DIR}/build/bin/contextsqueeze" "${ROOT_DIR}/cmd/contextsqueeze"
