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

TMPDIR_CSQ="$(mktemp -d)"
trap 'rm -rf "${TMPDIR_CSQ}"' EXIT

printf 'phase4 identity smoke\nline2 raw\n' > "${TMPDIR_CSQ}/input.txt"
"${ROOT_DIR}/build/bin/contextsqueeze" --aggr 0 "${TMPDIR_CSQ}/input.txt" > "${TMPDIR_CSQ}/out.txt"
cmp -s "${TMPDIR_CSQ}/input.txt" "${TMPDIR_CSQ}/out.txt"

"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.txt" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.html" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.pdf" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats --source docx "${ROOT_DIR}/internal/ingest/fixtures/sample.docx" >/dev/null

json_out="$(${ROOT_DIR}/build/bin/contextsqueeze --max-tokens 30 --json ${ROOT_DIR}/internal/ingest/fixtures/sample.txt)"
printf '%s\n' "$json_out" > "${TMPDIR_CSQ}/out.json"
CSQ_JSON_PATH="${TMPDIR_CSQ}/out.json" python - <<'PY'
import json, os
from pathlib import Path
obj=json.loads(Path(os.environ["CSQ_JSON_PATH"]).read_text())
required=["bytes_in","bytes_out","tokens_in_approx","tokens_out_approx","reduction_pct","aggressiveness","profile","budget_applied","truncated","source_type","warnings"]
for k in required:
    assert k in obj, f"missing {k}"
assert obj["tokens_out_approx"] <= 30, "max tokens not honored"
PY

# Phase 4 bench suite and determinism check
"${ROOT_DIR}/build/bin/contextsqueeze" bench --suite default --runs 3 --warmup 1 --aggr 0..9 > "${TMPDIR_CSQ}/bench.txt"

# Conservative performance gate using counters from medium fixture
stats_out="$(${ROOT_DIR}/build/bin/contextsqueeze stats --aggr 6 ${ROOT_DIR}/testdata/bench/medium.txt)"
printf '%s\n' "$stats_out" > "${TMPDIR_CSQ}/stats.txt"
CSQ_STATS_PATH="${TMPDIR_CSQ}/stats.txt" python - <<'PY'
import os, re
from pathlib import Path
text=Path(os.environ["CSQ_STATS_PATH"]).read_text()
m=re.search(r"counters tokens/sentences/candidates/pairs: (\d+)/(\d+)/(\d+)/(\d+)", text)
assert m, "missing counters line"
tokens,sents,cands,pairs=map(int,m.groups())
assert sents > 0, "invalid sentence count"
# Gate: per 10k sentences similarity pairs must remain conservative
pairs_per_10k = pairs * 10000 / sents
assert pairs_per_10k <= 500000, f"similarity pairs too high: {pairs_per_10k}"
# Gate: candidate checks should be bounded too
cand_per_10k = cands * 10000 / sents
assert cand_per_10k <= 700000, f"candidate checks too high: {cand_per_10k}"
PY
