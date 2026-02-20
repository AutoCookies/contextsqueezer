# Changelog

## 1.2.0
- Upgraded benchmark harness with suite/file/dir modes, warmups, multi-run stats (min/median/p95), per-run hashes, and optional JSON schema output.
- Added deterministic bench failure on hash mismatches across runs.
- Added CPU/heap profiling support in `profile` using Go pprof files (`--cpu`, `--heap`, `--seconds`).
- Added stage/counter metrics plumbing across Go pipeline and native core.
- Added profiling-driven hot path improvements for chunked dedupe signatures and candidate filtering metrics.
- Added conservative performance regression gate based on similarity counters.

## 1.1.0
- Added streaming/chunked pipeline for large documents with deterministic chunk reassembly.
- Added cross-chunk dedupe using deterministic sentence signatures and fixed-size LRU registry.
- Added memory guardrails with soft ceiling and `--max-memory-mb` support.

## 1.0.0
- Added ingestion for PDF, DOCX, HTML, TXT/MD with source detection and size guardrails.
- Added deterministic max-token budget controller and JSON output packaging.

## 0.2.0
- Added deterministic Phase 1 compression core algorithms in C++ and aggressiveness wiring.

## 0.1.0
- Initial foundation with Go CLI + C++ shared library bridge, CI, and smoke tests.
