# Network Stack Benchmarks

Comprehensive performance benchmarks for the TCP/IP network stack implementation.

## Running Benchmarks

### Run All Benchmarks
```bash
go test -bench=. -benchmem ./tests/benchmarks/
```

### Run Specific Benchmark Category
```bash
# TCP benchmarks
go test -bench=BenchmarkTCP -benchmem ./tests/benchmarks/

# Routing benchmarks
go test -bench=BenchmarkRouting -benchmem ./tests/benchmarks/

# Checksum benchmarks
go test -bench=BenchmarkChecksum -benchmem ./tests/benchmarks/

# Memory benchmarks
go test -bench=BenchmarkMemory -benchmem ./tests/benchmarks/
```

### Run with CPU Profiling
```bash
go test -bench=BenchmarkTCPThroughput -cpuprofile=cpu.prof ./tests/benchmarks/
go tool pprof cpu.prof
```

### Run with Memory Profiling
```bash
go test -bench=BenchmarkMemoryAllocation -memprofile=mem.prof ./tests/benchmarks/
go tool pprof mem.prof
```

### Run with Detailed Output
```bash
go test -bench=. -benchmem -benchtime=10s ./tests/benchmarks/
```

## Benchmark Categories

### TCP Benchmarks (`tcp_benchmark_test.go`)
- **BenchmarkTCPThroughput**: Measures throughput with various payload sizes (64B - 64KB)
- **BenchmarkTCPLatency**: Measures round-trip latency
- **BenchmarkTCPConcurrentConnections**: Tests performance with 10-10,000 concurrent connections
- **BenchmarkTCPHandshake**: Measures 3-way handshake performance
- **BenchmarkTCPRetransmission**: Tests retransmission logic overhead
- **BenchmarkTCPCongestionControl**: Measures congestion control overhead
- **BenchmarkTCPBufferOperations**: Tests buffer read/write performance
- **BenchmarkTCPRealWorld**: Simulates HTTP-like request/response patterns
- **BenchmarkTCPPipelined**: Tests pipelined requests (HTTP/2-like)
- **BenchmarkTCPLargeTransfer**: Simulates 1MB file transfers
- **BenchmarkTCPMemoryAllocation**: Tracks memory allocations per operation

### Routing Benchmarks (`routing_benchmark_test.go`)
- **BenchmarkRoutingTableLookup**: Tests lookup performance with 10-10,000 routes
- **BenchmarkRoutingTableLookupWorstCase**: Worst-case lookup (default route only)
- **BenchmarkRoutingTableLookupBestCase**: Best-case lookup (first match)
- **BenchmarkRoutingTableAdd**: Route addition performance
- **BenchmarkRoutingTableRemove**: Route removal performance
- **BenchmarkRoutingTableLongestPrefixMatch**: Tests longest prefix matching logic
- **BenchmarkRoutingTableRandomLookup**: Random IP lookup patterns
- **BenchmarkRoutingTableConcurrent**: Concurrent lookup performance
- **BenchmarkRoutingTableCIDRLookup**: CIDR-based lookups with various prefix lengths

### Checksum Benchmarks (`checksum_benchmark_test.go`)
- **BenchmarkChecksum**: Checksum calculation for 20B - 64KB payloads
- **BenchmarkVerifyChecksum**: Checksum verification performance
- **BenchmarkUpdateChecksum**: Incremental checksum updates (RFC 1624)
- **BenchmarkChecksumWithPseudoHeader**: TCP/UDP pseudo-header checksums
- **BenchmarkChecksumAlignment**: Impact of data alignment
- **BenchmarkChecksumOddLength**: Performance with odd vs even length data
- **BenchmarkChecksumParallel**: Concurrent checksum calculation
- **BenchmarkChecksumVsRecalculation**: Update vs full recalculation comparison
- **BenchmarkChecksumRealWorld**: Real packet processing simulation

### Memory Benchmarks (`memory_benchmark_test.go`)
- **BenchmarkMemoryAllocation**: Overall memory allocation patterns
- **BenchmarkBufferAllocation**: Buffer allocation overhead
- **BenchmarkBufferReuse**: Buffer reuse vs new allocation
- **BenchmarkGCPressure**: GC impact under high/low allocation
- **BenchmarkConnectionMemory**: Per-connection memory overhead
- **BenchmarkSegmentMemory**: Per-segment memory overhead
- **BenchmarkReceiveBufferMemory**: Receive buffer memory usage
- **BenchmarkMemoryFragmentation**: Memory fragmentation patterns
- **BenchmarkHeapVsStack**: Heap vs stack allocation comparison
- **BenchmarkPooledBuffers**: Buffer pool performance
- **BenchmarkMemoryLeakDetection**: Checks for potential memory leaks

## Target Metrics

### Performance Targets
- **TCP Throughput**: ≥ 1 Gbps (1,000,000,000 bps)
- **TCP Latency**: < 100 µs round-trip
- **Concurrent Connections**: ≥ 10,000 simultaneous connections
- **Memory per Connection**: < 10 KB
- **Checksum Performance**: > 1 GB/s
- **Routing Lookup**: < 1 µs with 1,000 routes

### Optimization Priorities
1. Buffer pooling (20-30% improvement expected)
2. Checksum optimization (SIMD or incremental updates)
3. Routing table optimization (trie-based structure)
4. Lock granularity improvements

## Interpreting Results

### Throughput
- Measured in Gbps (gigabits per second)
- Higher is better
- Compare with standard library implementation

### Latency
- Measured in µs (microseconds)
- Lower is better
- Should be < 100 µs for local connections

### Memory Allocations
- Shown as `allocs/op` in benchmark output
- Lower is better
- Zero allocations in hot paths is ideal

### Memory Usage
- Shown as `B/op` (bytes per operation)
- Lower is better
- Watch for memory leaks (increasing usage over time)

## Example Output

```
BenchmarkTCPThroughput/PayloadSize_1024B-8    1000000    1234 ns/op    0.83 Gbps    1024 B/op    2 allocs/op
BenchmarkTCPLatency-8                        10000000     120 ns/op     120 µs/op      0 B/op    0 allocs/op
BenchmarkRoutingTableLookup/Routes_1000-8     5000000     250 ns/op       0 B/op    0 allocs/op
BenchmarkChecksum/Size_1500-8                 2000000     850 ns/op    1.76 GB/s   1500 B/op    0 allocs/op
```

## Continuous Benchmarking

Track benchmark results over time to detect performance regressions:

```bash
# Save baseline
go test -bench=. -benchmem ./tests/benchmarks/ > baseline.txt

# After changes, compare
go test -bench=. -benchmem ./tests/benchmarks/ > current.txt
benchcmp baseline.txt current.txt
```

## Profiling Commands

### CPU Profile
```bash
go test -bench=BenchmarkTCPThroughput -cpuprofile=cpu.prof ./tests/benchmarks/
go tool pprof -http=:8080 cpu.prof
```

### Memory Profile
```bash
go test -bench=BenchmarkMemoryAllocation -memprofile=mem.prof ./tests/benchmarks/
go tool pprof -http=:8080 mem.prof
```

### Block Profile (Contention)
```bash
go test -bench=BenchmarkTCPConcurrent -blockprofile=block.prof ./tests/benchmarks/
go tool pprof -http=:8080 block.prof
```

## Notes

- Benchmarks are designed to measure isolated components
- Real-world performance may vary based on system load and configuration
- Run benchmarks multiple times and average results for consistency
- Disable CPU throttling and other system optimizations for accurate results
- Use `-benchtime=10s` for more stable results on fast operations
