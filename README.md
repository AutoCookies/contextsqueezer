# Context-Squeezer

Phase 1 implementation of a deterministic text compression core.

**Important:** Phase 0 used identity squeeze only. This repository now includes Phase 1 deterministic compression while keeping `aggr=0` as identity.

## Requirements

- Go 1.22+
- CMake 3.16+
- Linux: gcc/g++ toolchain
- macOS: clang/clang++ toolchain

## Build

```bash
./scripts/build.sh
```

Artifacts:

- Native shared library: `build/native/lib/libcontextsqueeze.so` (Linux) or `build/native/lib/libcontextsqueeze.dylib` (macOS)
- Native tests: `build/native/bin/contextsqueeze_tests`
- CLI binary: `build/bin/contextsqueeze`

## Test

```bash
./scripts/test.sh
```

This runs:

- native C++ tests
- `go test ./...`
- CLI smoke test (input equals output for default `aggr=0`)

## Format and static checks

```bash
./scripts/format.sh
```

Includes:

- `gofmt`
- `go vet`
- trailing whitespace check
- newline-at-EOF check

## Run

Print version:

```bash
./build/bin/contextsqueeze --version
```

Squeeze file (default identity behavior unless `--aggr` or `--profile` pushes compression):

```bash
./build/bin/contextsqueeze testdata/sample.txt --aggr 6
```

Show stats:

```bash
./build/bin/contextsqueeze --stats testdata/sample.txt --aggr 6
```

Run benchmark table across aggressiveness levels:

```bash
./build/bin/contextsqueeze bench testdata/sample.txt
```

## Compression behavior in Phase 1

Implemented deterministic algorithms in C++ core:

- rule-based sentence segmentation
- repeated boilerplate block removal
- near-duplicate sentence removal by TF cosine similarity with deterministic candidate bucketing
- low-information sentence pruning by document-level TF-IDF score

Notes:

- `Aggressiveness` range is `0..9`
- `0` is strict identity
- no sentence reordering; only dropping content
- output bytes are always copied from original input bytes

## Troubleshooting

### Error loading shared library on Linux

If runtime linker cannot find `libcontextsqueeze.so`, run through the provided scripts or set:

```bash
export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
```

### Error loading shared library on macOS

Set:

```bash
export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
```

### cgo build errors

Ensure CMake build completed first:

```bash
./scripts/build.sh
```

The Go package links against the native library generated into `build/native/lib`.

## Performance & Quality Notes

Guaranteed by implementation/tests:

- deterministic output for same input/options
- anchor retention checks in bench mode (code fences, URLs, numeric-heavy lines, headings)
- section coverage and keyword recall checks in bench mode

Not included yet:

- PDF/DOCX parsing
- streaming/chunking pipelines
- embedding/LLM-based semantics
