#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${ROOT_DIR}/build/bin/contextsqueeze"

"${ROOT_DIR}/scripts/build.sh" >/dev/null

if [[ "$(uname -s)" == "Darwin" ]]; then
  export DYLD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${DYLD_LIBRARY_PATH:-}"
else
  export LD_LIBRARY_PATH="${ROOT_DIR}/build/native/lib:${LD_LIBRARY_PATH:-}"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT

cat > "${TMP}/sample.txt" <<'TXT'
# ACCEPTANCE
Keep this anchor sentence.
Noise sentence. Noise sentence. Noise sentence.
Visit https://example.com/docs
TXT

cat > "${TMP}/sample.html" <<'HTML'
<html><body><h1>Acceptance H1</h1><p>Visible text.</p><script>ignore()</script></body></html>
HTML

cat > "${TMP}/sample.pdf" <<'PDF'
%PDF-1.1
1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj
2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj
3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 300 300] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj
4 0 obj << /Length 70 >> stream
BT
/F1 12 Tf
40 250 Td
(Acceptance PDF text line.) Tj
ET
endstream endobj
5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj
xref
0 6
0000000000 65535 f
0000000010 00000 n
0000000063 00000 n
0000000122 00000 n
0000000270 00000 n
0000000414 00000 n
trailer << /Root 1 0 R /Size 6 >>
startxref
492
%%EOF
PDF

python - <<PY
from zipfile import ZipFile, ZIP_DEFLATED
from pathlib import Path
p=Path("${TMP}/sample.docx")
with ZipFile(p, "w", compression=ZIP_DEFLATED) as z:
    z.writestr("word/document.xml", """<?xml version='1.0' encoding='UTF-8'?><w:document xmlns:w='http://schemas.openxmlformats.org/wordprocessingml/2006/main'><w:body><w:p><w:r><w:t>Acceptance DOCX heading</w:t></w:r></w:p><w:p><w:r><w:t>Acceptance docx paragraph.</w:t></w:r></w:p></w:body></w:document>""")
PY

for f in sample.txt sample.html sample.pdf sample.docx; do
  "${BIN}" stats --source auto "${TMP}/${f}" >/dev/null 2>"${TMP}/${f}.stats"
done

# determinism
"${BIN}" --aggr 6 --max-tokens 50 "${TMP}/sample.txt" > "${TMP}/o1.txt"
"${BIN}" --aggr 6 --max-tokens 50 "${TMP}/sample.txt" > "${TMP}/o2.txt"
sha1sum "${TMP}/o1.txt" "${TMP}/o2.txt" | awk '{print $1}' | uniq | wc -l | grep -q '^1$'

# max tokens honored
json="$(${BIN} --json --max-tokens 40 "${TMP}/sample.txt")"
CSQ_JSON="$json" python - <<'PY'
import json, os
obj=json.loads(os.environ["CSQ_JSON"])
assert obj["tokens_out_approx"] <= 40
PY

# anchor retention
"${BIN}" --aggr 6 "${TMP}/sample.txt" > "${TMP}/anchor_out.txt"
grep -q "# ACCEPTANCE" "${TMP}/anchor_out.txt"
grep -q "https://example.com/docs" "${TMP}/anchor_out.txt"

# usage exit code
if "${BIN}" --badflag >/dev/null 2>/dev/null; then
  echo "expected usage failure"
  exit 1
else
  code=$?
  [[ "$code" -eq 2 ]]
fi

# runtime loose threshold
start=$(date +%s)
"${BIN}" bench --file "${TMP}/sample.txt" --runs 2 --warmup 0 --aggr 0..9 >/dev/null 2>/dev/null
end=$(date +%s)
[[ $((end-start)) -lt 20 ]]

echo "acceptance: OK"
