#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

"${ROOT_DIR}/scripts/verify_no_binaries.sh"
"${ROOT_DIR}/scripts/build.sh"

if [[ "$(uname -s)" == "Darwin" ]]; then
  export DYLD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${DYLD_LIBRARY_PATH:-}"
else
  export LD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${LD_LIBRARY_PATH:-}"
fi

"${ROOT_DIR}/build/native/bin/contextsqueeze_tests"
go test ./...

go test ./internal/ingest -run '^$' -fuzz FuzzDetectType -fuzztime=1s
go test ./internal/ingest -run '^$' -fuzz FuzzParseHTML -fuzztime=1s
go test ./pkg/api -run '^$' -fuzz FuzzSqueezeBytes -fuzztime=1s

TMPDIR_CSQ="$(mktemp -d)"
trap 'rm -rf "${TMPDIR_CSQ}"' EXIT

printf 'phase5 identity smoke\nline2 raw\n' > "${TMPDIR_CSQ}/input.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" --aggr 0 "${TMPDIR_CSQ}/input.txt" > "${TMPDIR_CSQ}/out.txt"
cmp -s "${TMPDIR_CSQ}/input.txt" "${TMPDIR_CSQ}/out.txt"

"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.txt" >/dev/null 2>"${TMPDIR_CSQ}/stats.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.html" >/dev/null 2>"${TMPDIR_CSQ}/stats_html.txt"

json_out="$(${ROOT_DIR}/build/bin/contextsqueeze --max-tokens 30 --json ${ROOT_DIR}/internal/ingest/fixtures/sample.txt)"
printf '%s\n' "$json_out" > "${TMPDIR_CSQ}/out.json"
CSQ_JSON_PATH="${TMPDIR_CSQ}/out.json" python - <<'PY'
import json, os
from pathlib import Path
obj=json.loads(Path(os.environ["CSQ_JSON_PATH"]).read_text())
required=["schema_version","engine_version","build","bytes_in","bytes_out","tokens_in_approx","tokens_out_approx","reduction_pct","aggressiveness","profile","budget_applied","truncated","source_type","warnings"]
for k in required:
    assert k in obj, f"missing {k}"
assert obj["schema_version"] == 1
assert obj["tokens_out_approx"] <= 30
PY

"${ROOT_DIR}/build/bin/contextsqueeze" bench --suite default --runs 3 --warmup 1 --aggr 0..9 >/dev/null 2>"${TMPDIR_CSQ}/bench.txt"

stats_out="$(${ROOT_DIR}/build/bin/contextsqueeze stats --aggr 6 ${ROOT_DIR}/testdata/bench/medium.txt 2>&1 >/dev/null)"
printf '%s\n' "$stats_out" > "${TMPDIR_CSQ}/stats_gate.txt"
CSQ_STATS_PATH="${TMPDIR_CSQ}/stats_gate.txt" python - <<'PY'
import os, re
from pathlib import Path
text=Path(os.environ["CSQ_STATS_PATH"]).read_text()
m=re.search(r"counters tokens/sentences/candidates/pairs: (\d+)/(\d+)/(\d+)/(\d+)", text)
assert m, "missing counters line"
_,sents,cands,pairs=map(int,m.groups())
assert sents > 0
assert pairs * 10000 / sents <= 500000
assert cands * 10000 / sents <= 700000
PY

"${ROOT_DIR}/scripts/acceptance.sh"
