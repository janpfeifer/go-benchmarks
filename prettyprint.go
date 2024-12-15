package benchmarks

import (
	"fmt"
	"time"
)

func PrettyPrint(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", int(d/time.Nanosecond))
	}
	if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d)/float64(time.Microsecond))
	}
	if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	}
	if d < 5*time.Minute {
		return fmt.Sprintf("%.1fs", float64(d)/float64(time.Second))
	}
	if d < time.Hour {
		minutes := d / time.Minute
		seconds := float64(d-(minutes*time.Minute)) / float64(time.Second)
		return fmt.Sprintf("%dm%04.1fs", minutes, seconds)
	}
	if d < 24*time.Hour {
		hours := d / time.Hour
		remainder := d - (hours * time.Hour)
		minutes := remainder / time.Minute
		remainder -= minutes * time.Minute
		seconds := remainder / time.Second
		return fmt.Sprintf("%dh%02dm%02ds", hours, minutes, seconds)
	}

	// Multi-day duration:
	Day := 24 * time.Hour
	days := d / Day
	remainder := d - (days * Day)
	hours := remainder / time.Hour
	remainder -= hours * time.Hour
	minutes := remainder / time.Minute
	remainder -= minutes * time.Minute
	seconds := remainder / time.Second
	return fmt.Sprintf("%dd %dh%02dm%02ds", days, hours, minutes, seconds)
}
