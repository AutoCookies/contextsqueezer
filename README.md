# Context-Squeezer (Phase 1)

## Requirements
- Go 1.22+
- CMake 3.16+
- C++17 compiler:
  - Linux: gcc/g++
  - macOS: clang/clang++

## Build
```bash
./scripts/build.sh
```

Build outputs:
- Native library: `build/native/lib/libcontextsqueeze.so` (Linux) or `build/native/lib/libcontextsqueeze.dylib` (macOS)
- Native tests: `build/native/bin/contextsqueeze_tests`
- CLI: `build/bin/contextsqueeze`

## Test
```bash
./scripts/test.sh
```

This runs:
- Native C++ unit tests
- `go test ./...`
- CLI smoke checks
- Bench and reduction gate checks on `testdata/sample.txt`

## Format and lint checks
```bash
./scripts/format.sh
```

## Run
Print version:
```bash
./build/bin/contextsqueeze --version
```

Squeeze file content to stdout:
```bash
./build/bin/contextsqueeze testdata/sample.txt --aggr 6 --profile local
```

Print stats:
```bash
./build/bin/contextsqueeze --stats testdata/sample.txt --aggr 6
```

Run bench sweep (`aggr=0..9`):
```bash
./build/bin/contextsqueeze bench testdata/sample.txt
```

## Output metrics
`--stats` reports:
- bytes in/out
- approx tokens in/out (`ceil(bytes/4)`)
- reduction %
- runtime ms

`bench` reports a markdown table and recommended aggressiveness based on the best reduction that still passes deterministic quality gates.

## Deterministic quality gates
Bench evaluates:
- Anchor retention = 100% (anchors: code fences, URLs, numeric-heavy lines, headings)
- Section coverage: each heading retains at least one following sentence
- Keyword recall of top TF-IDF terms:
  - `>= 70%` for aggr `<= 6`
  - `>= 60%` for aggr `>= 7`

If no aggressiveness level passes gates, bench exits non-zero.

## Troubleshooting
- **Runtime linker cannot find native library**
  - Re-run `./scripts/build.sh` so `build/native/lib` exists.
- **CMake not found**
  - Install CMake 3.16+ and rerun build.
- **Compiler errors on macOS**
  - Install Xcode Command Line Tools (`xcode-select --install`) and retry.

## Performance & Quality Notes
Guaranteed in Phase 1:
- Deterministic output for the same input/options.
- No sentence reordering; only dropping spans.
- Single-threaded processing with byte-safe behavior.

Not guaranteed in Phase 1:
- Perfect linguistic segmentation for every abbreviation/language.
- Semantic paraphrase understanding beyond TF/cosine similarity.
- Tokenizer parity with model-specific BPE token counts.
