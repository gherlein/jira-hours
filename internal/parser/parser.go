package parser

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type WeekHours struct {
	Code  string `yaml:"code"`
	Week1 int    `yaml:"week1"`
	Week2 int    `yaml:"week2"`
	Week3 int    `yaml:"week3"`
	Week4 int    `yaml:"week4"`
}

type MonthlyLog struct {
	Month string      `yaml:"month"`
	Hours []WeekHours `yaml:"hours"`
}

type WorkEntry struct {
	Code       string
	Week       int
	Hours      int
	TicketKey  string
}

func ParseMonthlyLog(filepath string) (*MonthlyLog, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading monthly log: %w", err)
	}

	var log MonthlyLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("parsing monthly log: %w", err)
	}

	if err := validateMonthlyLog(&log); err != nil {
		return nil, fmt.Errorf("invalid monthly log: %w", err)
	}

	return &log, nil
}

func (log *MonthlyLog) GetTotalHours() int {
	total := 0
	for _, entry := range log.Hours {
		total += entry.Week1 + entry.Week2 + entry.Week3 + entry.Week4
	}
	return total
}

func (log *MonthlyLog) GetHoursForWeek(week int) map[string]int {
	result := make(map[string]int)
	for _, entry := range log.Hours {
		switch week {
		case 1:
			result[entry.Code] = entry.Week1
		case 2:
			result[entry.Code] = entry.Week2
		case 3:
			result[entry.Code] = entry.Week3
		case 4:
			result[entry.Code] = entry.Week4
		}
	}
	return result
}

func (wh *WeekHours) GetWeekHours(week int) int {
	switch week {
	case 1:
		return wh.Week1
	case 2:
		return wh.Week2
	case 3:
		return wh.Week3
	case 4:
		return wh.Week4
	default:
		return 0
	}
}

func validateMonthlyLog(log *MonthlyLog) error {
	if log.Month == "" {
		return fmt.Errorf("month field required")
	}

	seenCodes := make(map[string]bool)
	for i, entry := range log.Hours {
		if entry.Code == "" {
			return fmt.Errorf("entry %d missing code", i)
		}
		if seenCodes[entry.Code] {
			return fmt.Errorf("duplicate code: %s", entry.Code)
		}
		seenCodes[entry.Code] = true

		if entry.Week1 < 0 || entry.Week2 < 0 || entry.Week3 < 0 || entry.Week4 < 0 {
			return fmt.Errorf("code %s has negative hours", entry.Code)
		}

		weekTotal := entry.Week1 + entry.Week2 + entry.Week3 + entry.Week4
		if weekTotal > 240 {
			return fmt.Errorf("code %s has unreasonable total hours: %d", entry.Code, weekTotal)
		}
	}

	return nil
}
