#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build_static"
INSTALL_DIR="${BUILD_DIR}/install"

mkdir -p "${BUILD_DIR}"
cd "${BUILD_DIR}"

# Build static C++ core
cmake "${ROOT_DIR}/native" \
  -DCMAKE_INSTALL_PREFIX="${INSTALL_DIR}" \
  -DBUILD_SHARED_LIBS=OFF \
  -DCSQ_WITH_SIMD=ON \
  -DCMAKE_BUILD_TYPE=Release

make -j$(nproc)
make install

# Build Go CLI with static linking
cd "${ROOT_DIR}"
export CGO_LDFLAGS="-L${INSTALL_DIR}/lib -lcontextsqueeze -lstdc++ -lm"
export CGO_CFLAGS="-I${INSTALL_DIR}/include"

mkdir -p "${ROOT_DIR}/build/bin"
go build -o "${ROOT_DIR}/build/bin/contextsqueeze_static" ./cmd/contextsqueeze

echo "Static build complete: ${ROOT_DIR}/build/bin/contextsqueeze_static"
ldd "${ROOT_DIR}/build/bin/contextsqueeze_static" || true
