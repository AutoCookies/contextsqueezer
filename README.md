# Context-Squeezer (Phase 0)

Phase 0 establishes the build/test foundation for a Go CLI orchestrator that calls a C++ core library through cgo.

**Current behavior:** identity squeeze. Input bytes are returned unchanged. Real compression starts in Phase 1.

## Requirements

- Go 1.22+
- CMake 3.16+
- C++17 compiler:
  - Linux: `g++`
  - macOS: `clang++`
- C compiler for cgo:
  - Linux: `gcc`
  - macOS: `clang`

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

- C++ unit tests (`contextsqueeze_tests`)
- `go test ./...`
- End-to-end smoke test validating CLI output matches input exactly

Optional formatting/lint gate:

```bash
./scripts/format.sh
```

## Run examples

Print version:

```bash
./build/bin/contextsqueeze --version
```

Squeeze file (identity in Phase 0):

```bash
./build/bin/contextsqueeze ./path/to/input.txt
```

## Troubleshooting

### `error while loading shared libraries: libcontextsqueeze.so`

Linux runtime linker cannot find the shared library. Run via scripts (`build.sh` and `test.sh`) or set:

```bash
export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
```

### `Library not loaded: libcontextsqueeze.dylib`

macOS runtime linker cannot find the shared library. Run via scripts or set:

```bash
export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
```

### cgo compiler errors

Ensure toolchain is installed and available on PATH:

- Linux: `gcc`, `g++`
- macOS: Xcode command line tools (`clang`, `clang++`)
