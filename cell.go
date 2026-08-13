package mapper

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Cell struct {
	Row   int
	Col   int
	Value string
}

func (c Cell) String() string { return c.Value }

func (c Cell) Int64() (int64, error) {
	value, err := strconv.ParseInt(strings.TrimSpace(c.Value), 10, 64)
	if err != nil {
		return 0, locatedError(KindConversion, "", "", "", c.Row, c.Col, err)
	}
	return value, nil
}

func (c Cell) Float64() (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(c.Value), 64)
	if err != nil {
		return 0, locatedError(KindConversion, "", "", "", c.Row, c.Col, err)
	}
	return value, nil
}

func (c Cell) Bool() (bool, error) {
	value, err := strconv.ParseBool(strings.TrimSpace(c.Value))
	if err != nil {
		return false, locatedError(KindConversion, "", "", "", c.Row, c.Col, err)
	}
	return value, nil
}

func (c Cell) List(separator string) []string {
	if c.Value == "" {
		return nil
	}
	parts := strings.Split(c.Value, separator)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func (c Cell) Time(layout string) (time.Time, error) {
	value, err := ParseDate(c.Value, layout, false)
	if err != nil {
		return time.Time{}, locatedError(KindConversion, "", "", "", c.Row, c.Col, err)
	}
	return value, nil
}

func CellName(row, col int) string {
	if row < 0 || col < 0 {
		return ""
	}
	name := ""
	for col >= 0 {
		name = string(rune('A'+col%26)) + name
		col = col/26 - 1
	}
	return fmt.Sprintf("%s%d", name, row+1)
}
