# Changelog

## 1.0.0
- Phase 5 release hardening and packaging:
  - repository hygiene enforcement (`scripts/verify_no_binaries.sh`)
  - release artifact generation (`scripts/release.sh`) with `SHA256SUMS`
  - release CI workflow (`.github/workflows/release.yml`)
  - sanitizer gate (`scripts/sanitize.sh`) and CI sanitizer job
  - acceptance suite with runtime-generated fixtures (`scripts/acceptance.sh`)
  - fuzz targets for ingest and API (`internal/ingest/fuzz_test.go`, `pkg/api/fuzz_test.go`)
  - strict exit code/logging policy with `--quiet`, `--verbose`, `CSQ_DEBUG`
  - JSON schema lock with `schema_version=1` and `engine_version`
- Includes prior benchmark/profiling/streaming capabilities from earlier phases.
