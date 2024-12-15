# go-benchmarks: a benchmark library with mean, median and arbitrary quantiles

Can be incorporated in tests, or in stand-alone programs.

## Highlights:

* Pretty-printing of the duration of each execution (to avoid having to count
  digits to interpret scale of Go's default benchmark framework).
  Duration pretty-printing function can be arbitrarily configured.
* Includes also median, and arbitrary percentiles (default to 5 and 99).

## Example

