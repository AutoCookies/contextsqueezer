#!/usr/bin/env bash
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

for pat in '*.pdf' '*.docx' '*.so' '*.dylib' '*.exe' '*.zip'; do
  hits="$(git ls-files "$pat" | while read -r f; do [[ -f "$f" ]] && echo "$f"; done || true)"
  if [[ -n "$hits" ]]; then
    echo "forbidden tracked binary extension detected: $pat"
    echo "$hits"
    exit 1
  fi
done

dist_hits="$(git ls-files 'dist/*' | while read -r f; do [[ -f "$f" ]] && echo "$f"; done || true)"
if [[ -n "$dist_hits" ]]; then
  echo "forbidden tracked dist artifacts"
  echo "$dist_hits"
  exit 1
fi

large="$(git ls-files -z | xargs -0 -I{} sh -c 'if [ -f "{}" ]; then s=$(wc -c <"{}" 2>/dev/null || echo 0); if [ "$s" -gt 2097152 ]; then echo "{} $s"; fi; fi')"
if [[ -n "$large" ]]; then
  echo "tracked files >2MB detected"
  echo "$large"
  exit 1
fi

for range in "" "--cached"; do
  if git diff $range --numstat | awk '$1=="-" || $2=="-" {print}' | grep -q .; then
    echo "binary additions detected in git diff $range"
    git diff $range --numstat | awk '$1=="-" || $2=="-" {print}'
    exit 1
  fi
done

echo "verify_no_binaries: OK"
