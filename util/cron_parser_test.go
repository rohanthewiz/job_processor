package util

import (
	"strings"
	"testing"
	"time"
)

func TestParseCronToEnglish(t *testing.T) {
	tests := []struct {
		name     string
		cronExpr string
		want     string
	}{
		// 6-field (with seconds) tests
		{"Every 15 seconds", "*/15 * * * * *", "Every 15 seconds"},
		{"Every 30 seconds", "*/30 * * * * *", "Every 30 seconds"},
		{"Every minute", "0 * * * * *", "Every minute"},
		{"Every 5 minutes", "0 */5 * * * *", "Every 5 minutes"},
		{"Every hour", "0 0 * * * *", "Every hour, on the hour"},
		{"Daily at midnight", "0 0 0 * * *", "Daily at midnight"},
		{"Daily at noon", "0 0 12 * * *", "Daily at noon"},
		{"Weekdays at 9:30", "0 30 9 * * 1-5", "Weekdays at 9:30"},

		// 5-field (without seconds) tests
		{"Every minute (5-field)", "* * * * *", "Every minute"},
		{"Every 5 minutes (5-field)", "*/5 * * * *", "Every 5 minutes"},
		{"Every hour (5-field)", "0 * * * *", "Every hour, on the hour"},
		{"Daily at midnight (5-field)", "0 0 * * *", "Daily at midnight"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCronToEnglish(tt.cronExpr)
			if got != tt.want {
				t.Errorf("ParseCronToEnglish(%q) = %q, want %q", tt.cronExpr, got, tt.want)
			}
		})
	}
}

func TestFormatDurationUntil(t *testing.T) {
	// Test past time
	pastTime := time.Now().Add(-1 * time.Hour)
	got := FormatDurationUntil(pastTime)
	if got != "already passed" {
		t.Errorf("FormatDurationUntil(past time) = %q, want %q", got, "already passed")
	}

	// Test various durations with approximate values
	testCases := []struct {
		name     string
		duration time.Duration
		contains string
	}{
		{"30 seconds", 30 * time.Second, "seconds"},
		{"5 minutes", 5 * time.Minute, "minutes"},
		{"2 hours", 2 * time.Hour, "hour"},
		{"2 days", 48 * time.Hour, "day"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			futureTime := time.Now().Add(tc.duration)
			got := FormatDurationUntil(futureTime)
			if !strings.Contains(got, tc.contains) {
				t.Errorf("FormatDurationUntil() = %q, expected to contain %q", got, tc.contains)
			}
		})
	}
}
