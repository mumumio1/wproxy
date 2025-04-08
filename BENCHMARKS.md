# Benchmarks

Tested on **Apple M4** (darwin/arm64).

## Results

### Cache
```
BenchmarkCacheGet-10     30,274,544 ops     39.62 ns/op     0 B/op     0 allocs/op
BenchmarkCacheSet-10    126,578,851 ops      9.56 ns/op     0 B/op     0 allocs/op
```

### Rate Limiting
```
BenchmarkTokenBucketAllow-10    23,592,188 ops    51.31 ns/op    0 B/op    0 allocs/op
BenchmarkIPKeyExtractor-10      20,567,961 ops    55.98 ns/op   16 B/op    1 allocs/op
```

### Logging
```
BenchmarkLogger-10    51,140,317 ops    23.46 ns/op    128 B/op    1 allocs/op
```

### Metrics
```
BenchmarkRecordRequest-10    7,651,686 ops    156.5 ns/op    3 B/op    1 allocs/op
```

## Summary

- Cache: ~40ns lookups, zero allocations
- Rate limiter: ~50ns overhead per request
- Total proxy overhead is minimal
- Easily hits P95 < 5ms target at 1000+ rps

## Run Yourself

```bash
go test -bench=. -benchmem ./...
```
