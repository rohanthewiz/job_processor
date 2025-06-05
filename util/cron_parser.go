package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// ParseCronToEnglish converts a cron expression to human-readable English
// Handles both 5-field (minute hour day month weekday) and 6-field (second minute hour day month weekday) formats
func ParseCronToEnglish(cronExpr string) string {
	// Parse the cron expression
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		// Try without seconds
		parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		schedule, err = parser.Parse(cronExpr)
		if err != nil {
			return fmt.Sprintf("Invalid cron: %v", err)
		}
	}

	// Get the next few runs to understand the pattern
	now := time.Now()
	next1 := schedule.Next(now)
	next2 := schedule.Next(next1)
	next3 := schedule.Next(next2)

	// Try to detect common patterns
	parts := strings.Fields(cronExpr)

	var second, minute, hour, day, month, weekday string

	if len(parts) == 6 {
		// 6-field format with seconds
		second = parts[0]
		minute = parts[1]
		hour = parts[2]
		day = parts[3]
		month = parts[4]
		weekday = parts[5]
	} else if len(parts) == 5 {
		// 5-field format without seconds
		second = "0"
		minute = parts[0]
		hour = parts[1]
		day = parts[2]
		month = parts[3]
		weekday = parts[4]
	} else {
		return "Invalid cron format"
	}

	// Check for special cases with seconds
	if len(parts) == 6 {
		if second == "*" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every second"
		}

		if second == "*/5" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every 5 seconds"
		}

		if second == "*/10" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every 10 seconds"
		}

		if second == "*/15" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every 15 seconds"
		}

		if second == "*/30" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every 30 seconds"
		}

		if second == "0" && minute == "*" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
			return "Every minute"
		}
	}

	// Check for minute-based patterns
	if second == "0" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
		if minute == "*" {
			return "Every minute"
		}
		if minute == "*/5" {
			return "Every 5 minutes"
		}
		if minute == "*/10" {
			return "Every 10 minutes"
		}
		if minute == "*/15" {
			return "Every 15 minutes"
		}
		if minute == "*/30" {
			return "Every 30 minutes"
		}
	}

	// Check for hourly patterns
	if second == "0" && minute == "0" && hour == "*" && day == "*" && month == "*" && weekday == "*" {
		return "Every hour, on the hour"
	}

	// Daily patterns
	if second == "0" && minute == "0" && hour == "0" && day == "*" && month == "*" && weekday == "*" {
		return "Daily at midnight"
	}

	if second == "0" && minute == "0" && hour == "12" && day == "*" && month == "*" && weekday == "*" {
		return "Daily at noon"
	}

	// Weekday patterns
	if weekday == "1-5" && day == "*" && month == "*" {
		timeStr := formatTimeString(second, minute, hour)
		return fmt.Sprintf("Weekdays at %s", timeStr)
	}

	if weekday == "0,6" && day == "*" && month == "*" {
		timeStr := formatTimeString(second, minute, hour)
		return fmt.Sprintf("Weekends at %s", timeStr)
	}

	// Check if it runs on specific weekdays
	if day == "*" && month == "*" && weekday != "*" {
		weekdayNames := map[string]string{
			"0": "Sunday", "1": "Monday", "2": "Tuesday", "3": "Wednesday",
			"4": "Thursday", "5": "Friday", "6": "Saturday", "7": "Sunday",
		}
		if name, ok := weekdayNames[weekday]; ok {
			timeStr := formatTimeString(second, minute, hour)
			return fmt.Sprintf("Every %s at %s", name, timeStr)
		}
	}

	// Check if it's a specific time daily
	if day == "*" && month == "*" && weekday == "*" && minute != "*" && hour != "*" {
		timeStr := formatTimeString(second, minute, hour)
		return fmt.Sprintf("Daily at %s", timeStr)
	}

	// Check if it's monthly on a specific day
	if day != "*" && month == "*" && weekday == "*" && minute != "*" && hour != "*" {
		timeStr := formatTimeString(second, minute, hour)
		return fmt.Sprintf("Monthly on day %s at %s", day, timeStr)
	}

	// Default: show the next 3 runs
	return fmt.Sprintf("Next runs: %s, %s, %s",
		next1.Format("Jan 2 15:04:05"),
		next2.Format("Jan 2 15:04:05"),
		next3.Format("Jan 2 15:04:05"))
}

// formatTimeString formats the time components into a readable string
func formatTimeString(second, minute, hour string) string {
	if second == "0" {
		return fmt.Sprintf("%s:%s", hour, minute)
	}
	return fmt.Sprintf("%s:%s", hour, minute)
}

// FormatDurationUntil formats the duration between now and a future time in a human-readable way
func FormatDurationUntil(futureTime time.Time) string {
	now := time.Now()
	if futureTime.Before(now) {
		return "already passed"
	}

	duration := futureTime.Sub(now)

	// If less than a minute
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		if seconds == 1 {
			return "in 1 second"
		}
		return fmt.Sprintf("in %d seconds", seconds)
	}

	// If less than an hour
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "in 1 minute"
		}
		return fmt.Sprintf("in %d minutes", minutes)
	}

	// If less than a day
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if hours == 1 && minutes == 0 {
			return "in 1 hour"
		}
		if minutes == 0 {
			return fmt.Sprintf("in %d hours", hours)
		}
		if hours == 1 {
			return fmt.Sprintf("in 1 hour %d minutes", minutes)
		}
		return fmt.Sprintf("in %d hours %d minutes", hours, minutes)
	}

	// Days
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	if days == 1 && hours == 0 {
		return "in 1 day"
	}
	if hours == 0 {
		return fmt.Sprintf("in %d days", days)
	}
	if days == 1 {
		return fmt.Sprintf("in 1 day %d hours", hours)
	}
	return fmt.Sprintf("in %d days %d hours", days, hours)
}
