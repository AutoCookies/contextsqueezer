# Context-Squeezer

Context-Squeezer is a deterministic Go CLI + C++ core library for compressing document text.

## Version

Current release: **1.0.0**.

## Requirements

- Go 1.22+
- CMake 3.16+
- Linux: `gcc`, `g++`
- macOS: `clang`, `clang++`

No third-party cgo dependencies are used beyond the native C++ core.

## Build

```bash
./scripts/build.sh
```

Artifacts:
- `build/bin/contextsqueeze`
- `build/native/lib/libcontextsqueeze.so` (Linux) or `libcontextsqueeze.dylib` (macOS)
- `build/native/bin/contextsqueeze_tests`

## Test

```bash
./scripts/test.sh
```

Also available:

```bash
./scripts/format.sh
```

## Supported Input Formats

- PDF
- DOCX
- HTML
- TXT / MD

Source detection uses extension + magic bytes. Override detection with `--source`.

## CLI Usage

Squeeze to stdout:

```bash
./build/bin/contextsqueeze doc.pdf --max-tokens 8000 --profile api
```

Emit JSON package:

```bash
./build/bin/contextsqueeze page.html --json > out.json
```

Write to file:

```bash
./build/bin/contextsqueeze --input report.docx --out squeezed.txt --max-tokens 2000
```

Stats view:

```bash
./build/bin/contextsqueeze stats ./internal/ingest/fixtures/sample.txt --max-tokens 80
```

Bench table:

```bash
./build/bin/contextsqueeze bench ./internal/ingest/fixtures/sample.html --max-tokens 120
```

## JSON Schema

`--json` emits stable UTF-8 JSON metadata fields:

- `bytes_in`, `bytes_out`
- `tokens_in_approx`, `tokens_out_approx`
- `reduction_pct`
- `aggressiveness`, `profile`
- `budget_applied`, `truncated`
- `source_type`, `warnings`
- `text` (if valid UTF-8) OR `text_b64` (otherwise)

## Token Approximation

Budgeting uses a deterministic approximation:

`approx_tokens = ceil(bytes/4) + whitespace_word_count`

This is not a model tokenizer/BPE.

## Guardrails

- Default input max size: **50MB** (override with `CSQ_MAX_BYTES`)
- Timeout cap in CLI for ingest + squeeze: **5s**
- Unknown binary-like files are rejected (`unsupported binary file`)

## Troubleshooting

### PDF extraction quality

- Scanned/image-only PDFs are not supported in Phase 2 (no OCR).
- Some PDFs may produce limited text due to layout/encoding.

### DOCX extraction errors

- If `word/document.xml` is missing, extraction fails with a clear error.

### Shared library load errors

Linux:

```bash
export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
```

macOS:

```bash
export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
```
