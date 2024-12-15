// Package benchmarks implements a benchmark library with mean, median and arbitrary quantiles.
// It includes a pretty-printing function of duration -- that can be arbitrarily configured.
//
// Example 1: Simple example:
//
//	testParams := []int{1, 2, 3, 5, 8}
//	testFns := make([]benchmarks.NamedFunc, len(testParams))
//	for ii, param := range testParams {
//		testFns.Name = fmt.Printf("Param=%d", param)
//		testFns.Func() = func() {
//			MyFunc(param)
//		}
//	}
//	benchmarks.New(testFns...).Done()
//
// Example 2: Measuring a CGO call -- we use inner-repeats because the time is very small.
//
//	const repeats = 1000
//	repeatedCGO := func() {
//		for _ = range repeats {
//			dummyCGO(unsafe.Pointer(plugin.api))
//		}
//	}
//	benchmarks.New(benchmarks.NamedFunction{"CGOCall", repeatedCGO}).
//		WithInnerRepeats(repeats).
//		Done()
package benchmarks

import (
	"fmt"
	"github.com/streadway/quantile"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type Options struct {
	fns                   []NamedFunction
	prettyPrintFn         func(time.Duration) string
	quantiles             []int
	warmUps, innerRepeats int
	duration              time.Duration
	tolerance             float64
	columnSize            int
}

// NamedFunction holds a function to be benchmarked and its name.
type NamedFunction struct {
	Name string
	Func func()
}

// DefaultQuantiles to use in benchmarking. It can be changed for a particular benchmark using Options.WithQuantiles.
var DefaultQuantiles = []int{5, 99}

// New sets up a benchmark for the list of named functions fns.
// You can further configure the returned Options, and call Done() when finished to execute the benchmark.
//
// Example 1: Simple example:
//
//	testParams := []int{1, 2, 3, 5, 8}
//	testFns := make([]benchmarks.NamedFunc, len(testParams))
//	for ii, param := range testParams {
//		testFns.Name = fmt.Printf("Param=%d", param)
//		testFns.Func() = func() {
//			MyFunc(param)
//		}
//	}
//	benchmarks.New(testFns...).Done()
//
// Example 2: Measuring a CGO call -- we use inner-repeats because the time is very small.
//
//	const repeats = 1000
//	repeatedCGO := func() {
//		for _ = range repeats {
//			dummyCGO(unsafe.Pointer(plugin.api))
//		}
//	}
//	benchmarks.New(benchmarks.NamedFunction{"CGOCall", repeatedCGO}).
//		WithInnerRepeats(repeats).
//		Done()
func New(fns ...NamedFunction) *Options {
	return &Options{
		fns:           fns,
		prettyPrintFn: PrettyPrint,
		quantiles:     DefaultQuantiles,
		warmUps:       5,
		innerRepeats:  1,
		duration:      1 * time.Second,
		tolerance:     0.001,
		columnSize:    10,
	}
}

// WithPrettyPrintFn sets a custom pretty-print function for formatting durations and returns the updated Options instance.
func (o *Options) WithPrettyPrintFn(fn func(time.Duration) string) *Options {
	o.prettyPrintFn = fn
	return o
}

// WithQuantiles sets the quantiles for the options and returns the updated Options instance.
// Default is given by DefaultQuantiles ({5, 99})
func (o *Options) WithQuantiles(quantiles ...int) *Options {
	o.quantiles = slices.Clone(quantiles)
	return o
}

// WithWarmUps sets the number of warm-up iterations (before starting the benchmark) and returns the updated Options instance.
func (o *Options) WithWarmUps(warmUps int) *Options {
	o.warmUps = warmUps
	return o
}

// WithInnerRepeats sets the expected number of inner repeats of the functions given for the benchmark and returns the updated Options instance.
//
// This only informs the inner repetitions inside the functions given, this library won't repeat any calls.
// If passing a value > 1 here, you must repeat the piece of code you want to benchmark inside the functions given to New.
//
// This is particularly important for things that run below a few microseconds, as it mitigates the time to call the functions passed.
// Default is 1.
//
// Notice that reported measures are divided by this number. That means changing this number shouldn't
// affect the reported mean.
func (o *Options) WithInnerRepeats(innerRepeats int) *Options {
	o.innerRepeats = innerRepeats
	return o
}

// WithDuration sets the benchmark duration for each function for the options and returns the updated Options instance.
// When running the benchmark (Options.Done) it will run each function for at least this amount time, collecting
// statistics.
func (o *Options) WithDuration(duration time.Duration) *Options {
	o.duration = duration
	return o
}

// WithTolerance sets the tolerance in the approximate quantiles calculations. The smaller the tolerance the larger
// the amount of memory used in approximating the quantiles -- which may impact the running time due to GC (??)
//
// Default is 0.001 which is good enough for most cases.
func (o *Options) WithTolerance(tolerance float64) *Options {
	o.tolerance = tolerance
	return o
}

// WithColumnSize sets the size (in number of runes) for each column reported by benchmark.
// Handy if setting WithPrettyPrint to something that uses more or less space.
//
// Default is 10.
// Notice that any value smaller than 9 may mis-align the header row.
func (o *Options) WithColumnSize(columnSize int) *Options {
	o.columnSize = columnSize
	return o
}

func nanosecondsEstimate(est *quantile.Estimator, quantile float64) time.Duration {
	return time.Duration(int(est.Get(quantile))) * time.Nanosecond
}

type results struct {
	mean, median time.Duration
	quantiles    []time.Duration
	count        int
}

func (o *Options) benchmarkOneFunc(fn func()) results {
	// Estimates for median & other quantiles
	estimates := make([]quantile.Estimate, 0, len(o.quantiles)+1)
	estimates = append(estimates, quantile.Known(0.50, o.tolerance))
	for _, pct := range o.quantiles {
		estimates = append(estimates, quantile.Known(float64(pct)/100.0, o.tolerance))
	}

	// Estimator for quantiles and mean.
	estimator := quantile.New(estimates...)
	var totalTime time.Duration
	var count int
	timer := time.NewTimer(o.duration)

collection:
	for {
		select {
		case <-timer.C:
			break collection
		default:
			start := time.Now()
			fn()
			elapsed := time.Since(start)
			estimator.Add(float64(elapsed) / float64(time.Nanosecond))
			totalTime += elapsed
			count++
		}
	}

	// Convert estimates back to time.Duration.
	r := results{
		mean:      totalTime / time.Duration(count),
		median:    nanosecondsEstimate(estimator, 0.50),
		quantiles: make([]time.Duration, len(o.quantiles)),
		count:     count,
	}
	for i, pct := range o.quantiles {
		r.quantiles[i] = nanosecondsEstimate(estimator, float64(pct)/100.0)
	}
	return r
}

func (o *Options) Done() {
	// First column
	header := "Benchmarks:"
	maxLen := len(header)
	runeCount := make([]int, len(o.fns))
	for ii, namedFn := range o.fns {
		runeCount[ii] = utf8.RuneCountInString(namedFn.Name)
		maxLen = max(maxLen, runeCount[ii])
	}

	// Header
	extraSpaces := maxLen - len(header)
	if extraSpaces > 0 {
		header = header + strings.Repeat(" ", extraSpaces)
	}
	fmt.Printf("%s\t%*s\t%*s", header, o.columnSize, "Mean", o.columnSize, "Median")
	for _, q := range o.quantiles {
		fmt.Printf("\t%*s", o.columnSize, fmt.Sprintf("%d%%-tile", q))
	}
	countStr := "Count"
	if o.innerRepeats > 1 {
		countStr = fmt.Sprintf("Runs(x%d)", o.innerRepeats)
	}
	fmt.Printf("\t%*s\n", o.columnSize, countStr)

	for ii, namedFn := range o.fns {
		// Warm-up
		for _ = range o.warmUps {
			namedFn.Func()
		}

		// Collect benchmark estimations.
		r := o.benchmarkOneFunc(namedFn.Func)
		repeats := o.innerRepeats
		if repeats > 1 {
			r.mean /= time.Duration(repeats)
			r.median /= time.Duration(repeats)
			for ii := range r.quantiles {
				r.quantiles[ii] /= time.Duration(repeats)
			}
		}

		// Pretty-print.
		name := namedFn.Name
		extraSpaces := maxLen - runeCount[ii]
		if extraSpaces > 0 {
			name = name + strings.Repeat(" ", extraSpaces)
		}
		fmt.Printf("%s\t%*s\t%*s", name, o.columnSize, o.prettyPrintFn(r.mean), o.columnSize, o.prettyPrintFn(r.median))
		for _, q := range r.quantiles {
			fmt.Printf("\t%*s", o.columnSize, o.prettyPrintFn(q))
		}
		fmt.Printf("\t%*d\n", o.columnSize, r.count)
	}
}
