# go-benchmarks: a benchmark library with mean, median and arbitrary quantiles

[![GoDev](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/janpfeifer/go-benchmarks)
[![GitHub](https://img.shields.io/github/license/janpfeifer/go-benchmarks)](https://github.com/Kwynto/gosession/blob/master/LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/janpfeifer/go-benchmarks)](https://goreportcard.com/report/github.com/janpfeifer/go-benchmarks)
[![TestStatus](https://github.com/janpfeifer/go-benchmarks/actions/workflows/go.yaml/badge.svg)](https://github.com/janpfeifer/go-benchmarks/actions/workflows/go.yaml)

Can be incorporated in tests, or in stand-alone programs.

> [!Tip]
> For more stability in tests, consider the following:
> 
> 1. Fixing the CPU frequency on the box being used. Typically, in a linux box this can be done with the tool `cpupower`.
>    For instance `sudo cpupower frequency-set -r -d 3000000 -u 3000000` will set the min and max frequency to 3Ghz.
> 2. Set your benchmark to only use the same class of CPUs, if you have heterogeneous cores. Newer Intel CPUs have P-core and E-core,
>    and when some test falls on an E-core it "destroys" the benchmark. See `taskset` command-line tool to fix the affinity to a
>    set of cores.

> [!Note]
> For a full distribution display, checkout the alternative [github.com/loov/hrtime](https://github.com/loov/hrtime?tab=readme-ov-file#benchmarking)

## Highlights:

* Pretty-printing of the duration of each execution (to avoid having to count
  digits to interpret scale of Go's default benchmark framework).
  Duration pretty-printing function can be arbitrarily configured.
* Includes also median, and arbitrary percentiles (default to 5 and 99).

Outputs will look like this:

```
Benchmarks:                           Mean          Median         5%-tile        99%-tile      Count(x10)
TestBenchArena/arena/1               263ns           262ns           259ns           272ns          334047
TestBenchArena/arena/5               285ns           284ns           280ns           296ns          311296
TestBenchArena/arena/10              322ns           321ns           316ns           366ns          279129
TestBenchArena/arena/100             968ns           963ns           948ns           1.1µs           99799                                              
TestBenchArena/arenaPool/1           148ns           148ns           143ns           160ns          537448
TestBenchArena/arenaPool/5           173ns           172ns           168ns           185ns          477437
TestBenchArena/arenaPool/10          209ns           207ns           203ns           244ns          407099
TestBenchArena/arenaPool/100         877ns           851ns           826ns           1.1µs          109780
TestBenchArena/malloc/1              232ns           231ns           227ns           240ns          373248
TestBenchArena/malloc/5              890ns           887ns           881ns           1.0µs          108137
TestBenchArena/malloc/10             1.7µs           1.7µs           1.7µs           1.9µs           57526
TestBenchArena/malloc/100           16.4µs          16.3µs          16.2µs          16.7µs            6101
TestBenchArena/go+pinner/1           156ns           153ns           146ns           186ns          514560
TestBenchArena/go+pinner/5           557ns           545ns           535ns           703ns          168819
TestBenchArena/go+pinner/10          1.1µs           1.1µs           1.0µs           1.5µs           85634
TestBenchArena/go+pinner/100        12.3µs          11.8µs          11.5µs          34.4µs            8137
```

## Examples

### Simple Example

```go
import (
    benchmarks "github.com/janpfeifer/go-benchmarks"
)

…

	testParams := []int{1, 2, 3, 5, 8}
	testFns := make([]benchmarks.NamedFunc, len(testParams))
	for ii, param := range testParams {
		testFns.Name = fmt.Printf("Param=%d", param)
		testFns.Func() = func() {
			MyFunc(param)
		}
	}
	benchmarks.New(testFns...).Done()
```


### Benchmarking a minimal CGO call

For this we use the `WithInnerRepeats` option, to mitigate the time to call the function being tested itself
(`repeatedCGO`).

You can run it with something like: `go test . -run=TestBenchCGO -test.v`

```go
import (
    benchmarks "github.com/janpfeifer/go-benchmarks"
)

func TestBenchCGO(t *testing.T) {
  plugin := must1(GetPlugin(*flagPluginName))
  const repeats = 1000
  repeatedCGO := func() {
    for _ = range repeats {
      dummyCGO(unsafe.Pointer(plugin.api))
    }
  }
  benchmarks.New(benchmarks.NamedFunction{"CGOCall", repeatedCGO}).
      WithInnerRepeats(repeats).
	  Done()
}
```

Outputs (in an intel 12K900 with the frequency limited to 3Ghz):

```
Benchmarks:           Mean          Median         5%-tile        99%-tile      Runs(x1000)
CGOCall               67ns            66ns            65ns            72ns           14666
```

### Benchmarking Local Allocations To Use As Arguments To CGO functions

```go
func TestBenchArena(t *testing.T) {
	plugin := must1(GetPlugin(*flagPluginName))
	client := must1(plugin.NewClient(nil))
	defer runtime.KeepAlive(client)

	numAllocationsList := []int{1, 5, 10, 100}
	allocations := make([]*int, 100)
	testFns := make([]benchmarks.NamedFunction, 4*len(numAllocationsList))
	const repeats = 10
	idxFn := 0
	for _, allocType := range []string{"arena", "arenaPool", "malloc", "go+pinner"} {
		for _, numAllocations := range numAllocationsList {
			testFns[idxFn].Name = fmt.Sprintf("%s/%s/%d", t.Name(), allocType, numAllocations)
			var fn func()
			switch allocType {
			case "arena":
				fn = func() {
					for _ = range repeats {
						arena := newArena(1024)
						for idx := range numAllocations {
							allocations[idx] = arenaAlloc[int](arena)
						}
						dummyCGO(unsafe.Pointer(allocations[numAllocations-1]))
						arena.Free()
					}
				}
			case "arenaPool":
				fn = func() {
					for _ = range repeats {
						arena := getArenaFromPool()
						for idx := range numAllocations {
							allocations[idx] = arenaAlloc[int](arena)
						}
						dummyCGO(unsafe.Pointer(allocations[numAllocations-1]))
						returnArenaToPool(arena)
					}
				}
			case "malloc":
				fn = func() {
					for _ = range repeats {
						for idx := range numAllocations {
							allocations[idx] = cMalloc[int]()
						}
						dummyCGO(unsafe.Pointer(allocations[numAllocations-1]))
						for idx := range numAllocations {
							cFree(allocations[idx])
						}
					}
				}
			case "go+pinner":
				fn = func() {
					for _ = range repeats {
						var pinner runtime.Pinner
						for idx := range numAllocations {
							v := idx
							allocations[idx] = &v
							pinner.Pin(allocations[idx])
						}
						dummyCGO(unsafe.Pointer(allocations[numAllocations-1]))
						pinner.Unpin()
					}
				}
			}
			testFns[idxFn].Func = fn
			idxFn++
		}
	}
	benchmarks.New(testFns...).
		WithInnerRepeats(repeats).
		WithWarmUps(10).
		Done()
}
```

Results:

```
Benchmarks:                           Mean          Median         5%-tile        99%-tile      Count(x10)
TestBenchArena/arena/1               263ns           262ns           259ns           272ns          334047
TestBenchArena/arena/5               285ns           284ns           280ns           296ns          311296
TestBenchArena/arena/10              322ns           321ns           316ns           366ns          279129
TestBenchArena/arena/100             968ns           963ns           948ns           1.1µs           99799                                              
TestBenchArena/arenaPool/1           148ns           148ns           143ns           160ns          537448
TestBenchArena/arenaPool/5           173ns           172ns           168ns           185ns          477437
TestBenchArena/arenaPool/10          209ns           207ns           203ns           244ns          407099
TestBenchArena/arenaPool/100         877ns           851ns           826ns           1.1µs          109780
TestBenchArena/malloc/1              232ns           231ns           227ns           240ns          373248
TestBenchArena/malloc/5              890ns           887ns           881ns           1.0µs          108137
TestBenchArena/malloc/10             1.7µs           1.7µs           1.7µs           1.9µs           57526
TestBenchArena/malloc/100           16.4µs          16.3µs          16.2µs          16.7µs            6101
TestBenchArena/go+pinner/1           156ns           153ns           146ns           186ns          514560
TestBenchArena/go+pinner/5           557ns           545ns           535ns           703ns          168819
TestBenchArena/go+pinner/10          1.1µs           1.1µs           1.0µs           1.5µs           85634
TestBenchArena/go+pinner/100        12.3µs          11.8µs          11.5µs          34.4µs            8137
```

