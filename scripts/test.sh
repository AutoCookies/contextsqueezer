#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export LD_LIBRARY_PATH="$ROOT_DIR/build/native/lib:${LD_LIBRARY_PATH:-}"
export DYLD_LIBRARY_PATH="$ROOT_DIR/build/native/lib:${DYLD_LIBRARY_PATH:-}"

"$ROOT_DIR/scripts/build.sh"
"$ROOT_DIR/build/native/bin/contextsqueeze_tests"
go test ./...

TMP_IN="$(mktemp)"
TMP_OUT="$(mktemp)"
trap 'rm -f "$TMP_IN" "$TMP_OUT"' EXIT
printf 'phase0 smoke\nline2\0x' > "$TMP_IN"
"$ROOT_DIR/build/bin/contextsqueeze" "$TMP_IN" > "$TMP_OUT"
if ! cmp -s "$TMP_IN" "$TMP_OUT"; then
  echo "smoke test mismatch"
  exit 1
fi
