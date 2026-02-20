#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${ROOT_DIR}/scripts/build.sh"

"${ROOT_DIR}/build/native/bin/contextsqueeze_tests"

go test ./...

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TMP_DIR}"' EXIT
INPUT_FILE="${TMP_DIR}/input.txt"
OUTPUT_FILE="${TMP_DIR}/output.txt"
printf 'Phase0 smoke test with\0binary' > "${INPUT_FILE}"
"${ROOT_DIR}/build/bin/contextsqueeze" "${INPUT_FILE}" > "${OUTPUT_FILE}"
cmp -s "${INPUT_FILE}" "${OUTPUT_FILE}"
