package mapper

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ExcelSerialTime(serial float64, date1904 bool) (time.Time, error) {
	if date1904 {
		return time.Time{}, ErrDate1904
	}
	if serial < 0 {
		return time.Time{}, fmt.Errorf("invalid Excel serial %g", serial)
	}
	days := int64(serial)
	fraction := serial - float64(days)
	if days >= 60 {
		days--
	}
	base := time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)
	return base.AddDate(0, 0, int(days)).Add(time.Duration(fraction * float64(24*time.Hour))).Round(time.Second), nil
}

func ParseDate(value, layout string, date1904 bool) (time.Time, error) {
	value = strings.TrimSpace(value)
	if serial, err := strconv.ParseFloat(value, 64); err == nil {
		return ExcelSerialTime(serial, date1904)
	}
	if layout != "" {
		return time.Parse(layout, value)
	}
	for _, candidate := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.Parse(candidate, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date %q", value)
}
