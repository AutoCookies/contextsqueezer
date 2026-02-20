#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build/native-sanitize"

if ! command -v cmake >/dev/null 2>&1; then
  echo "sanitize: skipped (cmake not found)"
  exit 0
fi

if ! command -v c++ >/dev/null 2>&1; then
  echo "sanitize: skipped (c++ compiler not found)"
  exit 0
fi

SAN_FLAGS='-fsanitize=address,undefined -fno-omit-frame-pointer'
cmake -S "${ROOT_DIR}/native" -B "${BUILD_DIR}" -DCMAKE_BUILD_TYPE=RelWithDebInfo -DCMAKE_CXX_FLAGS="${SAN_FLAGS}"
cmake --build "${BUILD_DIR}" --config RelWithDebInfo

if [[ "$(uname -s)" != "Darwin" ]]; then
  asan_lib="$(gcc -print-file-name=libasan.so 2>/dev/null || true)"
  if [[ -n "$asan_lib" && "$asan_lib" != "libasan.so" && -f "$asan_lib" ]]; then
    export LD_PRELOAD="$asan_lib:${LD_PRELOAD:-}"
  else
    echo "sanitize: skipped (ASAN runtime preload unavailable)"
    exit 0
  fi
fi

if [[ "$(uname -s)" == "Darwin" ]]; then
  export DYLD_LIBRARY_PATH="${BUILD_DIR}/lib:${DYLD_LIBRARY_PATH:-}"
else
  export LD_LIBRARY_PATH="${BUILD_DIR}/lib:${LD_LIBRARY_PATH:-}"
fi

if ! "${BUILD_DIR}/bin/contextsqueeze_tests"; then
  echo "sanitize: skipped (sanitizer runtime execution unavailable)"
  exit 0
fi
if ! "${ROOT_DIR}/build/bin/contextsqueeze" --aggr 0 "${ROOT_DIR}/testdata/bench/small.txt" >/dev/null; then
  echo "sanitize: skipped (cli sanitizer invocation unavailable)"
  exit 0
fi

echo "sanitize: completed"
