# Context-Squeezer

Phase 1 provides deterministic local compression with a Go CLI orchestrating a C++ core through cgo.

## Requirements

- Go 1.22+
- CMake 3.16+
- Linux: `gcc`, `g++`
- macOS: `clang`, `clang++`

## Build

```bash
./scripts/build.sh
```

Artifacts:

- `build/native/lib/libcontextsqueeze.so` (Linux) or `build/native/lib/libcontextsqueeze.dylib` (macOS)
- `build/native/bin/contextsqueeze_tests`
- `build/bin/contextsqueeze`

## Test

```bash
./scripts/test.sh
```

Runs native tests, `go test ./...`, identity smoke (`--aggr 0`), and bench validation on `testdata/sample.txt`.

Optional format/lint gate:

```bash
./scripts/format.sh
```

## Run

Version:

```bash
./build/bin/contextsqueeze --version
```

Squeeze a file:

```bash
./build/bin/contextsqueeze --aggr 6 --profile local ./path/to/input.txt
```

Stats mode:

```bash
./build/bin/contextsqueeze --stats --aggr 6 ./path/to/input.txt
```

Bench mode (aggr 0..9 markdown table + recommendation):

```bash
./build/bin/contextsqueeze bench ./testdata/sample.txt
```

## Compression behavior (Phase 1)

Implemented deterministic algorithms:

- rule-based sentence segmentation (`. ? !` and paragraph boundaries)
- repeated/boilerplate paragraph removal
- near-duplicate sentence removal via TF cosine similarity with candidate buckets
- low-information pruning with TF-IDF scoring and anchor protection

Anchors are always protected (`#` headings, code fences, URLs, numeric-heavy lines, all-caps heading-like lines).

## Troubleshooting

Linux linker error (`libcontextsqueeze.so` not found):

```bash
export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
```

macOS linker error (`libcontextsqueeze.dylib` not found):

```bash
export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
```

## Performance & Quality Notes

Guaranteed:

- deterministic output for same input/options
- no sentence reordering (only drops)
- bench quality gates are deterministic: anchor retention, section coverage, keyword recall

Not guaranteed in Phase 1:

- semantic rewriting/paraphrasing
- language-specific NLP beyond simple ASCII tokenization
- cross-file streaming/chunking
