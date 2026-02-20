#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${ROOT_DIR}/scripts/build.sh"

if [[ "$(uname -s)" == "Darwin" ]]; then
  export DYLD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${DYLD_LIBRARY_PATH:-}"
else
  export LD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${LD_LIBRARY_PATH:-}"
fi

"${ROOT_DIR}/build/native/bin/contextsqueeze_tests"
go test ./...

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

input_file="${tmpdir}/input.txt"
printf 'phase0 identity smoke\nline2\0raw\n' > "${input_file}"

output_file="${tmpdir}/out.bin"
"${ROOT_DIR}/build/bin/contextsqueeze" "${input_file}" > "${output_file}"

cmp -s "${input_file}" "${output_file}"
