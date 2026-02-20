#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

gofmt -w $(find "$ROOT_DIR" -name '*.go' -not -path '*/build/*')
go vet ./...

if rg -n "[[:blank:]]+$" "$ROOT_DIR" --glob '!build/**' --glob '!*go.sum' --glob '!*LICENSE'; then
  echo "trailing whitespace detected"
  exit 1
fi

ROOT_DIR="$ROOT_DIR" python3 - <<'PY'
import os
from pathlib import Path
root = Path(os.environ['ROOT_DIR'])
for p in root.rglob('*'):
    if not p.is_file() or 'build' in p.parts or '.git' in p.parts:
        continue
    data = p.read_bytes()
    if data and data[-1] != 0x0A:
        print(f"missing newline at EOF: {p}")
        raise SystemExit(1)
PY
