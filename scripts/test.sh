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
printf 'Phase1 smoke test with\0binary and repeat. repeat. repeat.' > "${INPUT_FILE}"
"${ROOT_DIR}/build/bin/contextsqueeze" "${INPUT_FILE}" --aggr 0 > "${OUTPUT_FILE}"
cmp -s "${INPUT_FILE}" "${OUTPUT_FILE}"

STATS_OUT="${TMP_DIR}/stats.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" --stats --aggr 6 "${ROOT_DIR}/testdata/sample.txt" > "${STATS_OUT}"

BENCH_OUT="${TMP_DIR}/bench.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" bench "${ROOT_DIR}/testdata/sample.txt" > "${BENCH_OUT}"
rg "Recommended aggressiveness:" "${BENCH_OUT}" >/dev/null

AGGR6_OUT="${TMP_DIR}/aggr6.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" "${ROOT_DIR}/testdata/sample.txt" --aggr 6 > "${AGGR6_OUT}"
IN_BYTES=$(wc -c < "${ROOT_DIR}/testdata/sample.txt")
OUT_BYTES=$(wc -c < "${AGGR6_OUT}")
REDUCTION=$(python - <<PY
in_b=${IN_BYTES}
out_b=${OUT_BYTES}
print((1.0 - (out_b / in_b)) * 100.0)
PY
)
python - <<PY
r=float(${REDUCTION})
assert r >= 30.0, f"aggr6 reduction too low: {r:.2f}%"
PY
