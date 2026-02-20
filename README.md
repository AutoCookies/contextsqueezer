# Context-Squeezer

Deterministic Go CLI + C++ core for document compression.

## Version

Current release: **1.0.0**.

## Guarantees

- Deterministic output for identical input/options.
- Drop-only behavior (no sentence reordering, no rewriting).
- Anchors preserved unless budget truncation is required.
- `MaxTokens` is honored with deterministic fallback truncation.
- No external network/model calls; processing is local.

## Build & Test

```bash
./scripts/build.sh
./scripts/test.sh
./scripts/format.sh
./scripts/verify_no_binaries.sh
```

## Supported Formats

- PDF
- DOCX
- HTML
- TXT/MD

## CLI Usage

```bash
./build/bin/contextsqueeze --max-tokens 8000 --profile api doc.pdf
./build/bin/contextsqueeze --json page.html > out.json
./build/bin/contextsqueeze stats --source docx --max-tokens 2000 report.docx
```

## Bench Harness

```bash
./build/bin/contextsqueeze bench --suite default --runs 5 --warmup 1 --aggr 0..9
./build/bin/contextsqueeze bench --file testdata/bench/large.txt --runs 10
./build/bin/contextsqueeze bench --dir testdata/bench --pattern "*.txt"
```

Bench checks deterministic SHA-256 digests per run and exits non-zero on mismatch.

## Profiling

```bash
./build/bin/contextsqueeze profile --cpu out/cpu.pprof --heap out/heap.pprof --seconds 10 testdata/bench/large.txt
```

Open profile UI:

```bash
go tool pprof -http=:0 out/cpu.pprof
```

## Exit Codes

- `0` success
- `2` usage error
- `3` input error
- `4` parse error
- `5` timeout
- `6` internal error

## Logging Policy

- `stdout`: squeezed text only (or JSON when `--json`)
- `stderr`: errors, warnings, stats, profiling output
- `--quiet`: suppress warnings
- `--verbose`: print stage timing
- `CSQ_DEBUG=1`: include stack traces for failures

## JSON Schema Lock

`--json` includes stable fields:

- `schema_version` (locked to `1`)
- `engine_version` (`1.0.0`)
- `build` object
- existing compression metadata (`bytes_*`, `tokens_*`, `reduction_pct`, `aggressiveness`, `profile`, `budget_applied`, `truncated`, `source_type`, `warnings`)
- `text` or `text_b64`

## Token Approximation

`approx_tokens = ceil(bytes/4) + whitespace_word_count`

## Release Packaging

Generate local artifacts (not committed):

```bash
./scripts/release.sh
```

Outputs in `dist/`:

- `contextsqueeze_linux_amd64`
- `contextsqueeze_darwin_amd64`
- `contextsqueeze_darwin_arm64`
- `SHA256SUMS`
- `VERSION`

## Troubleshooting

- Scanned/image-only PDFs are not OCR-processed.
- DOCX must contain `word/document.xml`.
- Linux runtime linking:
  ```bash
  export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
  ```
- macOS runtime linking:
  ```bash
  export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
  ```

See `docs/perf.md` for benchmark/profiling workflows.
