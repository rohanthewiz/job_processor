package util

import (
	"testing"
	"time"
)

func TestParseSchedule(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		input     string
		wantErr   bool
		checkFunc func(t *testing.T, result time.Time)
	}{
		{
			name:    "RFC3339 format",
			input:   "2024-01-15T14:30:00-08:00",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				if result.Format(time.RFC3339) != "2024-01-15T14:30:00-08:00" {
					t.Errorf("Expected 2024-01-15T14:30:00-08:00, got %s", result.Format(time.RFC3339))
				}
			},
		},
		{
			name:    "Date time with PST timezone",
			input:   "2024-01-15 14:30:00 PST",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				if result.IsZero() {
					t.Error("Expected non-zero time")
				}
			},
		},
		{
			name:    "US format with timezone",
			input:   "01/15/2024 2:30 PM EST",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				if result.IsZero() {
					t.Error("Expected non-zero time")
				}
			},
		},
		{
			name:    "Relative time: in 30m",
			input:   "in 30m",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				expectedTime := now.Add(30 * time.Minute)
				diff := result.Sub(expectedTime)
				if diff < -1*time.Second || diff > 1*time.Second {
					t.Errorf("Expected time around %v, got %v (diff: %v)", expectedTime, result, diff)
				}
			},
		},
		{
			name:    "Relative time: +1h",
			input:   "+1h",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				expectedTime := now.Add(1 * time.Hour)
				diff := result.Sub(expectedTime)
				if diff < -1*time.Second || diff > 1*time.Second {
					t.Errorf("Expected time around %v, got %v (diff: %v)", expectedTime, result, diff)
				}
			},
		},
		{
			name:    "Relative time: 5m",
			input:   "5m",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				expectedTime := now.Add(5 * time.Minute)
				diff := result.Sub(expectedTime)
				if diff < -1*time.Second || diff > 1*time.Second {
					t.Errorf("Expected time around %v, got %v (diff: %v)", expectedTime, result, diff)
				}
			},
		},
		{
			name:    "Named timezone: America/New_York",
			input:   "2024-01-15 14:30:00 America/New_York",
			wantErr: false,
			checkFunc: func(t *testing.T, result time.Time) {
				loc, _ := time.LoadLocation("America/New_York")
				expected := time.Date(2024, 1, 15, 14, 30, 0, 0, loc)
				if !result.Equal(expected) {
					t.Errorf("Expected %v, got %v", expected, result)
				}
			},
		},
		{
			name:    "Invalid format",
			input:   "not a valid time",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSchedule(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchedule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}
