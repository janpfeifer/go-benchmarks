package benchmarks

import (
	"testing"
	"time"
)

func TestPrettyPrint(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{123 * time.Nanosecond, "123ns"},
		{1270 * time.Nanosecond, "1.3µs"},
		{1230 * time.Microsecond, "1.2ms"},
		{180280 * time.Millisecond, "180.3s"},
		{5*time.Minute + 7200*time.Millisecond, "5m07.2s"},
		{3*time.Hour + 5*time.Minute + 7200*time.Millisecond, "3h05m07s"},
		{100*24*time.Hour + 3*time.Hour + 5*time.Minute + 7200*time.Millisecond, "100d 3h05m07s"},
	}

	for _, tt := range tests {
		got := PrettyPrint(tt.input)
		if got != tt.want {
			t.Errorf("PrettyPrint(%v) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
