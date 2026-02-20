# Context-Squeezer

Deterministic Go CLI + C++ core for document compression.

## Version

Current release: **1.1.0**.

## Build & Test

```bash
./scripts/build.sh
./scripts/test.sh
./scripts/format.sh
```

## Supported Formats

- PDF
- DOCX
- HTML
- TXT/MD

## Core Usage

```bash
./build/bin/contextsqueeze --max-tokens 8000 --profile api doc.pdf
./build/bin/contextsqueeze --json page.html > out.json
./build/bin/contextsqueeze stats --source docx --max-tokens 2000 report.docx
./build/bin/contextsqueeze bench --max-tokens 3000 ./testdata/sample.txt
./build/bin/contextsqueeze profile --max-memory-mb 512 ./testdata/sample.txt
```

## Phase 3 Scalability Additions

- Chunked streaming pipeline for large documents (section-aware, fallback sentence windows).
- Cross-chunk dedupe with deterministic signature registry.
- Soft memory guardrail (`--max-memory-mb`, default 1024MB) with warning-based degradation.
- `profile` command with deterministic stage timings:
  - total
  - segmentation
  - dedupe
  - prune
  - reassembly
  - peak memory estimate

## JSON Schema (Stable)

`--json` output is unchanged and includes:

- `bytes_in`, `bytes_out`
- `tokens_in_approx`, `tokens_out_approx`
- `reduction_pct`
- `aggressiveness`, `profile`
- `budget_applied`, `truncated`
- `source_type`, `warnings`
- `text` (valid UTF-8) OR `text_b64`

## Token Approximation

`approx_tokens = ceil(bytes/4) + whitespace_word_count`

## Guardrails

- Input size cap default: `50MB` (`CSQ_MAX_BYTES` to override)
- CLI timeout: `5s`
- Unknown binary-like files rejected
- Soft memory ceiling default: `1024MB` (`--max-memory-mb`)

## Troubleshooting

- Scanned/image-only PDFs are not OCR-processed.
- DOCX must contain `word/document.xml` (or parseable XML fallback fixture format).
- Linux runtime linking:
  ```bash
  export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
  ```
- macOS runtime linking:
  ```bash
  export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
  ```
