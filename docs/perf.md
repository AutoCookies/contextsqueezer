# Performance Guide

## Benchmark Suite

```bash
./build/bin/contextsqueeze bench --suite default --runs 3 --warmup 1 --aggr 0..9
```

## Single-File Bench

```bash
./build/bin/contextsqueeze bench --file testdata/bench/large.txt --runs 10 --aggr 6
```

## CPU/Heap Profiling

```bash
./build/bin/contextsqueeze profile --cpu out/cpu.pprof --heap out/heap.pprof --seconds 10 testdata/bench/large.txt
```

View profile UI:

```bash
go tool pprof -http=:0 out/cpu.pprof
```

## Hotspot-Driven Optimization Notes

Top hotspots observed from `go tool pprof -top` on `testdata/bench/large.txt`:

1. `runtime.cgocall`
2. `internal/pipeline.segmentSentences`
3. token/word parsing path (`bytes.Fields` before optimization)

Applied optimizations:

- Replaced `bytes.Fields` in token approximation with a byte-scan word counter.
- Reworked signature construction to top-k hashed token selection (avoid full token-list sorting).
- Kept deterministic hashing and stable ordering.

Measured improvement on the same machine:

- `profile --seconds 1` loop count increased from ~57k to ~79k iterations (**~37% improvement**).

## Regression Gate Guidance

`./scripts/test.sh` enforces conservative non-flaky counter limits on `testdata/bench/medium.txt`:

- similarity pairs per 10k sentences
- candidate checks per 10k sentences

This avoids direct wall-time assertions in CI.
