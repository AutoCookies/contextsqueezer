#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
VERSION="1.0.0"

rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

"${ROOT_DIR}/scripts/build.sh"

build_one() {
  local goos="$1"
  local goarch="$2"
  local out="$3"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -ldflags "-X contextsqueezer/internal/version.Version=${VERSION}" \
    -o "${DIST_DIR}/${out}" "${ROOT_DIR}/cmd/contextsqueeze"
}

build_one linux amd64 contextsqueeze_linux_amd64
build_one darwin amd64 contextsqueeze_darwin_amd64
build_one darwin arm64 contextsqueeze_darwin_arm64

printf '%s\n' "${VERSION}" > "${DIST_DIR}/VERSION"
(
  cd "${DIST_DIR}"
  sha256sum contextsqueeze_linux_amd64 contextsqueeze_darwin_amd64 contextsqueeze_darwin_arm64 VERSION > SHA256SUMS
)

echo "release artifacts generated in ${DIST_DIR}"
