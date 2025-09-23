// Package uss provides time utilities for stable DATETIME(6) handling and flexible parsing.
package uss

import (
	"fmt"
	"time"
)

// ToMicroUTC standardizes a time.Time to UTC at microsecond precision, suitable for
// comparisons and for formats like MySQL's DATETIME(6).
func ToMicroUTC(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// ToDBLocalMicro converts a time.Time to the local timezone and truncates it to
// microsecond precision. For writing to a DB with a DSN configured with loc=Local,
// it is safest to pass the "local wall time" to avoid accidental timezone
// conversions by the database driver.
func ToDBLocalMicro(t time.Time) time.Time {
	return t.In(time.Local).Truncate(time.Microsecond)
}

var layouts = []string{
	time.RFC3339Nano,             // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,                 // 2006-01-02T15:04:05Z07:00
	"2006-01-02 15:04:05.999999", // MySQL DATETIME(6)
	"2006-01-02 15:04:05",        // MySQL DATETIME
	"2006-01-02",                 // Date-only (assume midnight)
}

// ParseFlexible tries common timestamp formats and returns UTC at microsecond precision.
func ParseFlexible(ts string) (time.Time, error) {
	for _, l := range layouts {
		if t, err := time.Parse(l, ts); err == nil {
			// If it came without zone (e.g., MySQL-like), it was parsed as UTC.
			// Normalize to UTC µs.
			return ToMicroUTC(t), nil
		}
	}
	// Try parsing in local time for formats without zone info
	for _, l := range []string{
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(l, ts, time.Local); err == nil {
			return ToMicroUTC(t), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format %q", ts)
}
