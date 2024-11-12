package c8y

import (
	"regexp"
	"strconv"
	"time"
)

// GetDateRange returns the dateFrom and dateTo based on an interval string, i.e. 1d, is 1 day
func GetDateRange(dateInterval string) (string, string) {

	pattern := regexp.MustCompile(`^(\d+)\s*([a-zA-Z]+)$`)
	result := pattern.FindStringSubmatch(dateInterval)

	if len(result) == 0 {
		Logger.Info("Invalid date interval. Using default '1d'")
		result = []string{"-", "1", "d"}
	}

	period, err := strconv.ParseFloat(result[1], 32)

	if err != nil {
		period = 1.0
	}
	unit := result[2]

	duration := convertToDuration(period, unit)

	dateTo := time.Now().Add(-1 * 10 * time.Second)

	dateFrom := dateTo.Add(-1 * duration)

	return dateFrom.Format(time.RFC3339), dateTo.Format(time.RFC3339)

}

func convertToDuration(period float64, unit string) time.Duration {
	duration := time.Duration(period) * time.Hour * 24

	switch unit {
	case "d":
		duration = time.Duration(period) * time.Hour * 24

	case "h":
		duration = time.Duration(period) * time.Hour

	case "min":
		duration = time.Duration(period) * time.Minute

	case "s":
		duration = time.Duration(period) * time.Second
	}
	return duration
}

// GetRoundedTime Get the rounded timestamp (i.e. start of the hour, start of the minute, start of the day)
func GetRoundedTime(date *time.Time, unit string) (roundTime time.Time) {
	var now time.Time
	if date != nil {
		now = *date
	} else {
		now = time.Now()
	}

	switch unit {
	case "d":
		roundTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	case "h":
		roundTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, time.Local)

	case "min":
		roundTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), 0, 0, time.Local)

	case "s":
		roundTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.Local)

	case "10min":
		rounded10Min := now.Minute() - (now.Minute() % 10)
		roundTime = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), rounded10Min, 0, 0, time.Local)

	default:
		panic("Invalid unit. Only d, h, min, s and 10min are valid units")
	}
	return
}

// ParseDate returns a Time object from a string
/* func ParseDate(value string) (dateObj *time.Time, err error) {
	var tempDate time.Time
	// Try to parse with ISO8801 format
	if tempDate, err = time.Parse("2006-01-02T15:04:05-07", value); err == nil {
		dateObj = &tempDate
		return dateObj, nil
	}

	// Try generic date parser
	if tempDate, err = dateparse.ParseAny(value); err == nil {
		dateObj = &tempDate
		return dateObj, nil
	}

	err = fmt.Errorf("Could not parse date")
	return nil, err
} */
