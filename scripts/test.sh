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

# identity smoke
TMPDIR_CSQ="$(mktemp -d)"
trap 'rm -rf "${TMPDIR_CSQ}"' EXIT
printf 'phase2 identity smoke\nline2 raw\n' > "${TMPDIR_CSQ}/input.bin"
"${ROOT_DIR}/build/bin/contextsqueeze" --aggr 0 "${TMPDIR_CSQ}/input.bin" > "${TMPDIR_CSQ}/out.bin"
cmp -s "${TMPDIR_CSQ}/input.bin" "${TMPDIR_CSQ}/out.bin"

# ingest smoke for all fixture types
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.txt" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.html" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats "${ROOT_DIR}/internal/ingest/fixtures/sample.pdf" >/dev/null
"${ROOT_DIR}/build/bin/contextsqueeze" stats --source docx "${ROOT_DIR}/internal/ingest/fixtures/sample.docx" >/dev/null

# budget + json schema smoke
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
