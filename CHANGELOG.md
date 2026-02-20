# Changelog

## 1.1.0
- Added streaming/chunked pipeline for large documents with deterministic chunk reassembly.
- Added cross-chunk dedupe using deterministic sentence signatures and fixed-size LRU registry.
- Added complexity lock notes and eliminated full-document pairwise comparisons.
- Added memory guardrails with soft ceiling and `--max-memory-mb` support.
- Added `contextsqueeze profile <file>` timing/memory instrumentation output.
- Added large-document deterministic/performance tests and heading continuity tests.

## 1.0.0
- Added ingestion for PDF, DOCX, HTML, TXT/MD with source detection and size guardrails.
- Added deterministic max-token budget controller and JSON output packaging.

## 0.2.0
- Added deterministic Phase 1 compression core algorithms in C++ and aggressiveness wiring.

## 0.1.0
- Initial foundation with Go CLI + C++ shared library bridge, CI, and smoke tests.
