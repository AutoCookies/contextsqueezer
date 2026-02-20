#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mapfile -t go_files < <(cd "${ROOT_DIR}" && rg --files -g '*.go')
if ((${#go_files[@]} > 0)); then
  gofmt -w "${go_files[@]}"
fi

go vet ./...

while IFS= read -r file; do
  if LC_ALL=C grep -nE '[[:space:]]$' "$file" >/dev/null; then
    echo "trailing whitespace found in $file"
    exit 1
  fi

  if [[ -s "$file" ]]; then
    python - "$file" <<'PY'
from pathlib import Path
import sys
p = Path(sys.argv[1])
data = p.read_bytes()
if data and not data.endswith(b"\n"):
    print(f"missing newline at EOF: {p}")
    raise SystemExit(1)
PY
  fi
done < <(cd "${ROOT_DIR}" && rg --files)
