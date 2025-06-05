package util

import (
	"fmt"
	"strings"
	"time"
)

// supportedFormats contains all the time formats we support
var supportedFormats = []string{
	"2006-01-02 15:04:05 MST",   // 2006-01-02 15:04:05 PST
	"2006-01-02T15:04:05 MST",   // 2006-01-02T15:04:05 PST
	"01/02/2006 3:04 PM MST",    // 01/02/2006 3:04 PM PST
	time.RFC3339,                // 2006-01-02T15:04:05Z07:00
	"Jan 2, 2006 3:04 PM MST",   // Jan 2, 2006 3:04 PM PST
	"2006-01-02 15:04:05 -0700", // 2006-01-02 15:04:05 -0800
	"2006-01-02T15:04:05 -0700", // 2006-01-02T15:04:05 -0800
	"01/02/2006 3:04 PM -0700",  // 01/02/2006 3:04 PM -0800
	"Jan 2, 2006 3:04 PM -0700", // Jan 2, 2006 3:04 PM -0800
	// time.RFC3339Nano,              // 2006-01-02T15:04:05.999999999Z07:00
}

// ParseSchedule parses a schedule string which can be:
// - An absolute time in various formats (with optional timezone)
// - A relative time (e.g., "in 30m", "+1h", "5m")
func ParseSchedule(scheduleStr string) (time.Time, error) {
	// First check for relative time
	if t, ok := parseRelativeTime(scheduleStr); ok {
		return t, nil
	}

	// Try absolute formats
	for _, format := range supportedFormats {
		if t, err := time.Parse(format, scheduleStr); err == nil {
			return t, nil
		}
	}

	// Try with location lookup for named timezones
	if t, err := parseWithLocation(scheduleStr); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", scheduleStr)
}

// parseRelativeTime attempts to parse relative time expressions
func parseRelativeTime(s string) (time.Time, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	now := time.Now()

	// Handle "in X duration" format
	if strings.HasPrefix(s, "in ") {
		durationStr := strings.TrimPrefix(s, "in ")
		if d, err := time.ParseDuration(durationStr); err == nil {
			return now.Add(d), true
		}
	}

	// Handle "+duration" format
	if strings.HasPrefix(s, "+") {
		if d, err := time.ParseDuration(s[1:]); err == nil {
			return now.Add(d), true
		}
	}

	// Handle "Xm", "Xh", etc without "+" prefix
	if d, err := time.ParseDuration(s); err == nil {
		return now.Add(d), true
	}

	return time.Time{}, false
}

// parseWithLocation attempts to parse time strings with location/timezone names
func parseWithLocation(scheduleStr string) (time.Time, error) {
	// Formats without timezone info that we'll try with locations
	layoutsNoTZ := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"01/02/2006 3:04 PM",
		"Jan 2, 2006 3:04 PM",
	}

	// Common timezone names to try
	timezones := []string{
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"Europe/London",
		"Europe/Paris",
		"Asia/Tokyo",
		"Australia/Sydney",
		"UTC",
	}

	// Check if string ends with a timezone name
	parts := strings.Fields(scheduleStr)
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]

		for _, layout := range layoutsNoTZ {
			// Remove timezone from string for parsing
			dateStr := strings.TrimSuffix(scheduleStr, " "+lastPart)

			// Try to load the location
			if loc, err := time.LoadLocation(lastPart); err == nil {
				if t, err := time.ParseInLocation(layout, dateStr, loc); err == nil {
					return t, nil
				}
			}

			// Try common timezone names if exact match fails
			for _, tz := range timezones {
				if strings.Contains(strings.ToLower(tz), strings.ToLower(lastPart)) {
					if loc, err := time.LoadLocation(tz); err == nil {
						if t, err := time.ParseInLocation(layout, dateStr, loc); err == nil {
							return t, nil
						}
					}
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("could not parse with location")
}
