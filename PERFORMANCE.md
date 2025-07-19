# Kuysor Performance Analysis

This document provides detailed performance analysis and optimization guidance for Kuysor.

## 📊 Benchmark Environment

**Test Platform:**
- **CPU**: Apple M1 (ARM64)
- **Cores**: 8 cores
- **Go Version**: 1.21.5
- **OS**: macOS (darwin)

## 🚀 Detailed Performance Metrics

### Core Operations Performance

| Operation | Time (μs) | Memory (KB) | Allocations | Ops/sec | Notes |
|-----------|-----------|-------------|-------------|---------|-------|
| **Basic Query Build** | 64.1 | 46.8 | 473 | 15,600 | Standard cursor pagination |
| **Query with Cursor** | 131.3 | 83.2 | 809 | 7,600 | Includes cursor parsing |
| **Complex Query** | 160.2 | 53.2 | 488 | 6,200 | Multi-table JOIN |
| **Offset Query** | 90.7 | 63.7 | 620 | 11,000 | Traditional pagination |

### Component-Level Performance

| Component | Time | Allocations | Efficiency Rating |
|-----------|------|-------------|-------------------|
| **Cursor Parsing** | 1.65μs | 20 | ⭐⭐⭐⭐⭐ |
| **Result Sanitization** | 1.45μs | 22 | ⭐⭐⭐⭐⭐ |
| **Sort Parsing (1 col)** | 113ns | 3 | ⭐⭐⭐⭐⭐ |
| **Sort Parsing (8 cols)** | 688ns | 13 | ⭐⭐⭐⭐⭐ |

## 📈 Scalability Analysis

### Sort Column Scaling
```
Columns: 1      2      4      8
Time:    113ns  206ns  352ns  688ns  (Linear: O(n))
Memory:  88B    184B   376B   760B   (Linear: O(n))
Allocs:  3      5      8      13     (Linear: O(n))
```

**Analysis**: Excellent linear scaling. Each additional column adds ~85ns and ~95B memory.

### Nullable Sort Method Comparison

| Method | Time (μs) | Memory (KB) | Database Support | Recommendation |
|--------|-----------|-------------|------------------|----------------|
| **FirstLast** | 70.3 | 47.5 | PostgreSQL, Oracle | Use when supported |
| **CaseWhen** | 72.4 | 48.3 | MySQL < 8.0 | Legacy MySQL |
| **BoolSort** | 83.4 | 48.1 | Most databases | Default choice |

### SQL Modification Performance

| Operation | Time (μs) | Memory (KB) | Use Case |
|-----------|-----------|-------------|----------|
| **Append WHERE** | 118.7 | 37.2 | Adding conditions |
| **Complex WHERE** | 188.0 | 40.2 | Multiple conditions |
| **Set ORDER BY** | 58-87 | 27-28 | Sorting setup |
| **Set LIMIT** | 33.1 | 24.8 | Pagination limit |
| **Set OFFSET** | 32.1 | 22.7 | Offset pagination |
| **Convert to COUNT** | 18-45 | 13-16 | Count queries |

## 💾 Memory Analysis

### Allocation Patterns

**Basic Query Building** (473 allocations):
- SQL parsing: ~200 allocations
- Sort processing: ~100 allocations  
- Query building: ~173 allocations

**Cursor Operations** (+336 allocations):
- Cursor parsing: +20 allocations
- Additional conditions: +316 allocations

### Memory Optimization Opportunities

1. **String Builder Usage**: Replace concatenations in hot paths
2. **Object Pooling**: Reuse frequently allocated structs
3. **Slice Pre-allocation**: Size slices appropriately
4. **Cursor Caching**: Cache parsed cursors for reuse

## ⚡ Throughput Projections

### Single-Core Performance

| Scenario | Ops/sec | Requests/day | CPU % at 10M req/day |
|----------|---------|--------------|---------------------|
| **Basic Queries** | 15,600 | 1.3B | 0.7% |
| **With Cursors** | 7,600 | 656M | 1.5% |
| **Complex Queries** | 6,200 | 535M | 1.9% |

### Multi-Core Scaling

Kuysor operations are stateless and scale linearly:
- **8-core M1**: ~124,800 ops/sec (basic queries)
- **16-core server**: ~249,600 ops/sec (basic queries)
- **32-core server**: ~499,200 ops/sec (basic queries)

## 🔧 Performance Optimization Guide

### Hot Path Optimizations

1. **Query Building Path**:
   ```
   NewQuery() → WithOrderBy() → WithLimit() → Build()
   64.1μs total, optimize: SQL parsing (30%), sort processing (20%)
   ```

2. **Cursor Path**:
   ```
   Cursor parsing: 1.65μs → Query building: 129.6μs
   Optimize: Complex WHERE generation (major bottleneck)
   ```

### Placeholder Performance

| Type | Time (ns) | Relative Performance |
|------|-----------|---------------------|
| **Question** | 909 | 100% (baseline) |
| **Dollar** | 1,316 | 145% (45% slower) |
| **At** | 1,530 | 168% (68% slower) |

**Recommendation**: Use Question mark placeholders for maximum performance.

### Database-Specific Optimizations

**PostgreSQL**:
- Use `FirstLast` null sort method (fastest)
- Dollar placeholders are native but slower in Kuysor

**MySQL**:
- Use `BoolSort` for MySQL 8.0+
- Use `CaseWhen` for older versions
- Question mark placeholders optimal

**SQLite**:
- Use `BoolSort` null sort method
- Question mark placeholders optimal

## 📊 Profiling Guidelines

### CPU Profiling
```bash
go test -bench=BenchmarkQueryBuild -cpuprofile=cpu.prof
go tool pprof cpu.prof
```

### Memory Profiling
```bash
go test -bench=BenchmarkMemoryAllocations -memprofile=mem.prof
go tool pprof mem.prof
```

### Heap Analysis
```bash
go test -bench=. -memprofile=heap.prof
go tool pprof -alloc_space heap.prof
```

## 🎯 Performance Targets

### Current Targets (Maintained)
- Basic query building: < 100μs
- Cursor operations: < 200μs  
- Sort parsing: Linear O(n) scaling
- Memory allocations: < 500 per operation

### Future Optimization Goals
- Basic query building: < 50μs
- Cursor operations: < 100μs
- Memory allocations: < 300 per operation
- Zero-allocation cursor parsing

## 🔍 Performance Regression Prevention

### CI Performance Checks
```bash
# Baseline benchmark
make bench > baseline.txt

# After changes
make bench > current.txt

# Compare (should implement automated comparison)
# Fail if >20% regression in critical paths
```

### Critical Performance Paths
1. **NewQuery().Build()** - Core user operation
2. **WithCursor().Build()** - Pagination performance
3. **SanitizeMap()** - Result processing
4. **Cursor parsing** - Navigation efficiency

## 📚 Further Reading

- [Go Performance Best Practices](https://github.com/golang/go/wiki/Performance)
- [Profiling Go Programs](https://blog.golang.org/profiling-go-programs)
- [Memory Management in Go](https://blog.golang.org/ismmkeynote)

---

**Note**: Performance characteristics may vary based on hardware, Go version, and query complexity. Always benchmark in your specific environment for production planning. 