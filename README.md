# Context-Squeezer (Phase 0)

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
- End-to-end smoke test (input file content equals output)

## Format and lint checks
```bash
./scripts/format.sh
```

This runs:
- `gofmt -w` on Go files
- `go vet ./...`
- File hygiene checks (no trailing whitespace, newline at EOF)

## Run
Print version:
```bash
./build/bin/contextsqueeze --version
```

Squeeze file content to stdout:
```bash
./build/bin/contextsqueeze path/to/input.txt
```

## Troubleshooting
- **Runtime linker cannot find native library**
  - Re-run `./scripts/build.sh` so `build/native/lib` exists.
  - Ensure you run the built CLI from this repository context where embedded rpath resolves.
- **CMake not found**
  - Install CMake 3.16+ and rerun build.
- **Compiler errors on macOS**
  - Install Xcode Command Line Tools (`xcode-select --install`) and retry.

## Phase note
Phase 0 implements an identity squeeze only (output equals input). Real compression algorithms begin in Phase 1.
