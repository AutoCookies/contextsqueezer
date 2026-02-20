# Context-Squeezer

Deterministic Go CLI + C++ core for document compression.

## Version

Current release: **1.2.0**.

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
```

## Benchmark Harness (Phase 4)

```bash
./build/bin/contextsqueeze bench --suite default --runs 5 --warmup 1 --aggr 0..9 --max-tokens 0
./build/bin/contextsqueeze bench --file testdata/bench/large.txt --runs 10 --aggr 6
./build/bin/contextsqueeze bench --dir testdata/bench --pattern "*.txt" --runs 3
```

Bench output includes:

- per-run rows
- aggregate rows (min/median/p95)
- SHA-256 digest checks for determinism (bench fails on mismatch)
- optional JSON via `--json`

## Profiling

```bash
./build/bin/contextsqueeze profile --cpu out/cpu.pprof --heap out/heap.pprof --seconds 10 testdata/bench/large.txt
```

Open pprof UI:

```bash
go tool pprof -http=:0 out/cpu.pprof
```

## Stats/Stage Timings

`stats` prints stage timing and counters, including:

- ingest
- segmentation
- tokenization
- candidate filtering
- similarity
- pruning
- assembly
- cross-chunk registry
- budgeting

## JSON Schema (Stable)

`--json` output remains stable and includes:

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

See `docs/perf.md` for reproducible performance workflows.
