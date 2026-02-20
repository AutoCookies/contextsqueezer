# Changelog

## 1.0.0
- Added Phase 2 ingestion for PDF, DOCX, HTML, TXT/MD with source detection and size guardrails.
- Added deterministic max-token budget controller with bounded attempts and sentence-boundary truncation.
- Added JSON output schema (`--json`) with metadata and UTF-8/base64 text fields.
- Added Phase 2 CLI flags: `--input`, `--out`, `--max-tokens`, `--source`, `stats`, and updated `bench` support.

## 0.2.0
- Added deterministic Phase 1 compression core algorithms in C++ and aggressiveness wiring.
- Added stats/bench command support and deterministic quality-gate reporting.

## 0.1.0
- Initial foundation with Go CLI + C++ shared library bridge, CI, and smoke tests.
