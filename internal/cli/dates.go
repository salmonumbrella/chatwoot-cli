package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Matches: "2h ago", "30m ago", "1d ago", "2w ago", "1mo ago"
var relativeAgoRegex = regexp.MustCompile(`^(\d+)(mo|w|d|h|m)\s*ago$`)

// Matches: "30m", "2h", "1d" (future, for reminders)
var relativeFutureRegex = regexp.MustCompile(`^(\d+)(mo|w|d|h|m)$`)

// ParseRelativeTime parses human-friendly time expressions.
// Supports: "2h ago", "yesterday", "monday", "next tue", "30m", RFC3339.
func ParseRelativeTime(s string, now time.Time) (time.Time, error) {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return time.Time{}, fmt.Errorf("empty time expression")
	}

	input := strings.ToLower(raw)

	switch input {
	case "yesterday":
		return startOfDay(now).AddDate(0, 0, -1), nil
	case "today":
		return startOfDay(now), nil
	case "tomorrow":
		return startOfDay(now).AddDate(0, 0, 1), nil
	}

	if t, ok := parseWeekday(input, now); ok {
		return t, nil
	}

	if matches := relativeAgoRegex.FindStringSubmatch(input); len(matches) == 3 {
		value, err := strconv.Atoi(matches[1])
		if err != nil || value < 1 {
			return time.Time{}, fmt.Errorf("invalid relative time %q", raw)
		}
		return applyRelative(now, value, matches[2], -1)
	}

	if matches := relativeFutureRegex.FindStringSubmatch(input); len(matches) == 3 {
		value, err := strconv.Atoi(matches[1])
		if err != nil || value < 1 {
			return time.Time{}, fmt.Errorf("invalid relative time %q", raw)
		}
		return applyRelative(now, value, matches[2], 1)
	}

	if t, err := time.ParseInLocation("2006-01-02", raw, now.Location()); err == nil {
		return startOfDay(t), nil
	}

	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t, nil
	}

	return time.Time{}, fmt.Errorf("invalid time expression %q", raw)
}

// Helper functions (from gogcli)
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func parseWeekday(expr string, now time.Time) (time.Time, bool) {
	input := strings.TrimSpace(expr)
	if input == "" {
		return time.Time{}, false
	}

	next := false
	if strings.HasPrefix(input, "next ") {
		next = true
		input = strings.TrimSpace(strings.TrimPrefix(input, "next "))
	} else if strings.HasPrefix(input, "this ") {
		input = strings.TrimSpace(strings.TrimPrefix(input, "this "))
	}

	weekday, ok := weekdayMap[input]
	if !ok {
		return time.Time{}, false
	}

	base := startOfDay(now)
	delta := (int(weekday) - int(base.Weekday()) + 7) % 7
	if next && delta == 0 {
		delta = 7
	}

	return base.AddDate(0, 0, delta), true
}

var weekdayMap = map[string]time.Weekday{
	"sun":       time.Sunday,
	"sunday":    time.Sunday,
	"mon":       time.Monday,
	"monday":    time.Monday,
	"tue":       time.Tuesday,
	"tues":      time.Tuesday,
	"tuesday":   time.Tuesday,
	"wed":       time.Wednesday,
	"weds":      time.Wednesday,
	"wednesday": time.Wednesday,
	"thu":       time.Thursday,
	"thur":      time.Thursday,
	"thurs":     time.Thursday,
	"thursday":  time.Thursday,
	"fri":       time.Friday,
	"friday":    time.Friday,
	"sat":       time.Saturday,
	"saturday":  time.Saturday,
}

func applyRelative(now time.Time, value int, unit string, direction int) (time.Time, error) {
	if value < 1 {
		return time.Time{}, fmt.Errorf("invalid relative time")
	}

	switch unit {
	case "mo":
		return now.AddDate(0, direction*value, 0), nil
	case "w":
		return now.Add(time.Duration(direction*value) * 7 * 24 * time.Hour), nil
	case "d":
		return now.Add(time.Duration(direction*value) * 24 * time.Hour), nil
	case "h":
		return now.Add(time.Duration(direction*value) * time.Hour), nil
	case "m":
		return now.Add(time.Duration(direction*value) * time.Minute), nil
	default:
		return time.Time{}, fmt.Errorf("invalid relative time unit %q", unit)
	}
}
