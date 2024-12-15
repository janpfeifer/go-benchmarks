package benchmarks

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPrettyPrint(t *testing.T) {
	require.Equal(t, "123ns", PrettyPrint(123*time.Nanosecond))
	require.Equal(t, "1.3µs", PrettyPrint(1270*time.Nanosecond))
	require.Equal(t, "1.2ms", PrettyPrint(1230*time.Microsecond))
	require.Equal(t, "180.3s", PrettyPrint(180280*time.Millisecond))
	require.Equal(t, "5m07.2s", PrettyPrint(5*time.Minute+7200*time.Millisecond))
	require.Equal(t, "3h05m07s", PrettyPrint(3*time.Hour+5*time.Minute+7200*time.Millisecond))
	require.Equal(t, "100d 3h05m07s", PrettyPrint(100*24*time.Hour+3*time.Hour+5*time.Minute+7200*time.Millisecond))
}
