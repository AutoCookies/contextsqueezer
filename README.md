<div align="center">

<img src="./assets/contextsqueezer-logo.png" alt="Context-Squeezer Logo" width="180"/>

# Context-Squeezer

**Deterministic document compression — squeeze tokens, not meaning.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![C++](https://img.shields.io/badge/C++-Core-00599C?style=flat-square&logo=c%2B%2B&logoColor=white)](https://isocpp.org/)
[![Build](https://img.shields.io/badge/build-passing-brightgreen?style=flat-square)](#build--test)

**Context-Squeezer** is a high-performance, local CLI tool and library designed to compress large documents to fit within LLM token budgets while maintaining semantic structure and determinism.

---

## 🚀 Key Features

- 🔒 **Deterministic**: Identical input and options always produce identical output.
- ✂️ **Drop-only**: No rewriting or reordering — only optimal omission of less relevant sections.
- 🔌 **Edge-Ready**: Supports static linking and SIMD optimizations (AVX2, NEON) for ultra-fast local inference.
- ⚓ **Anchor-aware**: Preserves critical sections (headings, code blocks, URLs) automatically.
- 📊 **Observability**: Built-in structured logging and fine-grained metrics for production monitoring.

---

## 🛠️ Installation & Building

### Standard Build (Shared Library)
```bash
make build
```

### Production Static Build (Edge Deployment)
Creates a standalone binary with no external C++ library dependencies.
```bash
make build-static
```

### Technical Requirements
- Go 1.22+
- CMake 3.16+
- GCC/Clang with C++17 support

---

## 📦 Third-Party Integration

Context-Squeezer is designed to be easily integrated into other projects.

### Go Integration
```go
import "contextsqueezer/pkg/api"

opt := api.Options{MaxTokens: 2000, Profile: "api"}
squeezed, err := api.SqueezeBytes(input, opt)
```

### C++ Integration (via CMake)
```cmake
find_package(ContextSqueeze REQUIRED)
target_link_libraries(your_app PRIVATE ContextSqueeze::contextsqueeze)
```

### New C API Features
- **Progress Callbacks**: Track long-running compression jobs.
- **Robust Error Handling**: Thread-local error strings via `csq_last_error()`.

---

## 🔧 Observability & Tuning

### Logging
Set `CSQ_DEBUG=1` to enable detailed stage-by-stage debug logging.

### Memory Management
Context-Squeezer implements a streaming chunked processor for large files. Use `MaxMemoryMB` in the pipeline configuration to control peak resident memory usage.

---

## 📈 Benchmarking

The tool includes a built-in benchmark suite to verify performance and determinism:
```bash
./build/bin/contextsqueeze bench --suite default --runs 5
```

---

## 📄 License
This project is licensed under the terms of the LICENSE file included in the repository.

</div>

---

## Supported Formats

<div align="center">

| Format | Extension |
|:---:|:---:|
| 📄 PDF | `.pdf` |
| 📝 Word | `.docx` |
| 🌐 HTML | `.html` |
| 📃 Plain Text / Markdown | `.txt`, `.md` |

</div>

---

## Quick Start

### Build

```bash
./scripts/build.sh
```

### Run

```bash
# Compress a PDF to 8 000 tokens, API profile
./build/bin/contextsqueeze --max-tokens 8000 --profile api doc.pdf

# Output as JSON
./build/bin/contextsqueeze --json page.html > out.json

# Show compression stats for a DOCX
./build/bin/contextsqueeze stats --source docx --max-tokens 2000 report.docx
```

> **Runtime linking (Linux)**
> ```
> export LD_LIBRARY_PATH="$(pwd)/build/native/lib:${LD_LIBRARY_PATH}"
> ```
>
> **Runtime linking (macOS)**
> ```
> export DYLD_LIBRARY_PATH="$(pwd)/build/native/lib:${DYLD_LIBRARY_PATH}"
> ```

---

## Output & Logging

| Stream | Content |
|---|---|
| `stdout` | Squeezed text (or JSON when `--json`) |
| `stderr` | Errors, warnings, stats, profiling |

**Flags:**

| Flag | Effect |
|---|---|
| `--quiet` | Suppress warnings |
| `--verbose` | Print per-stage timing |
| `CSQ_DEBUG=1` | Include stack traces on failure |

---

## JSON Output Schema

When using `--json`, the output contains these stable fields (schema locked at version `1`):

```json
{
  "schema_version": 1,
  "engine_version": "1.0.0",
  "build": { ... },
  "bytes_in": 0,
  "bytes_out": 0,
  "tokens_in": 0,
  "tokens_out": 0,
  "reduction_pct": 0.0,
  "aggressiveness": "",
  "profile": "",
  "budget_applied": true,
  "truncated": false,
  "source_type": "",
  "warnings": [],
  "text": "..."
}
```

> Token approximation formula: `approx_tokens = ceil(bytes / 4) + whitespace_word_count`

---

## Benchmarking

```bash
# Run with determinism checks (SHA-256 per run)
./build/bin/contextsqueeze bench --suite default --runs 5 --warmup 1 --aggr 0..9

# Benchmark a single file
./build/bin/contextsqueeze bench --file testdata/bench/large.txt --runs 10

# Benchmark an entire directory
./build/bin/contextsqueeze bench --dir testdata/bench --pattern "*.txt"
```

Bench checks deterministic SHA-256 digests per run and exits non-zero on mismatch.

---

## Profiling

```bash
# Capture CPU + heap profiles
./build/bin/contextsqueeze profile \
  --cpu out/cpu.pprof \
  --heap out/heap.pprof \
  --seconds 10 \
  testdata/bench/large.txt

# View interactive profile UI
go tool pprof -http=:0 out/cpu.pprof
```

See [`docs/perf.md`](docs/perf.md) for full benchmark and profiling workflows.

---

## Development

```bash
# Run tests
./scripts/test.sh

# Format code
./scripts/format.sh

# Verify no committed binaries
./scripts/verify_no_binaries.sh

# Run sanitizer gate
./scripts/sanitize.sh
```

---

## Release

Generate release artifacts locally (not committed to git):

```bash
./scripts/release.sh
```

Outputs are placed in `dist/`:

```
dist/
├── contextsqueeze_linux_amd64
├── contextsqueeze_darwin_amd64
├── contextsqueeze_darwin_arm64
├── SHA256SUMS
└── VERSION
```

---

## Exit Codes

| Code | Meaning |
|:---:|---|
| `0` | Success |
| `2` | Usage error |
| `3` | Input error |
| `4` | Parse error |
| `5` | Timeout |
| `6` | Internal error |

---

## Troubleshooting

- **Scanned / image-only PDFs** — OCR is not performed; text extraction will be empty.
- **DOCX files** — Must contain a valid `word/document.xml` entry.
- **Linux linking errors** — Set `LD_LIBRARY_PATH` as shown in the Quick Start above.
- **macOS linking errors** — Set `DYLD_LIBRARY_PATH` as shown in the Quick Start above.

---

<div align="center">

**Context-Squeezer** · v1.0.0 · License

</div>
