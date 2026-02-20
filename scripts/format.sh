#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mapfile -t GO_FILES < <(find "${ROOT_DIR}" -name '*.go' -not -path '*/build/*')
if ((${#GO_FILES[@]} > 0)); then
  gofmt -w "${GO_FILES[@]}"
fi

go vet ./...

mapfile -t TEXT_FILES < <(find "${ROOT_DIR}" \
  -type f \
  \( -name '*.go' -o -name '*.h' -o -name '*.hpp' -o -name '*.c' -o -name '*.cpp' -o -name '*.sh' -o -name '*.md' -o -name '*.yml' -o -name 'CMakeLists.txt' \) \
  -not -path '*/build/*')

for file in "${TEXT_FILES[@]}"; do
  if rg -n "[[:blank:]]$" "$file" >/dev/null; then
    echo "Trailing whitespace found in $file"
    exit 1
  fi
  if [[ -s "$file" ]] && [[ "$(tail -c 1 "$file" | wc -l)" -eq 0 ]]; then
    echo "Missing newline at EOF in $file"
    exit 1
  fi
done
