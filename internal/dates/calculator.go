package dates

import (
	"fmt"
	"time"
)

func GetWeekMonday(year, month, weekNumber int, timezone string) (time.Time, error) {
	if weekNumber < 1 || weekNumber > 5 {
		return time.Time{}, fmt.Errorf("week number must be between 1 and 5, got %d", weekNumber)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone %s: %w", timezone, err)
	}

	firstDay := time.Date(year, time.Month(month), 1, 8, 0, 0, 0, loc)

	firstMonday := firstDay
	for firstMonday.Weekday() != time.Monday {
		firstMonday = firstMonday.AddDate(0, 0, 1)
	}

	targetMonday := firstMonday.AddDate(0, 0, (weekNumber-1)*7)

	if targetMonday.Month() != time.Month(month) {
		return time.Time{}, fmt.Errorf("week %d extends beyond month %d", weekNumber, month)
	}

	return targetMonday, nil
}

func ParseMonth(monthStr string) (year, month int, err error) {
	var parsed time.Time
	parsed, err = time.Parse("2006-01", monthStr)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month format %s (expected YYYY-MM): %w", monthStr, err)
	}

	return parsed.Year(), int(parsed.Month()), nil
}

func FormatForJira(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000-0700")
}
