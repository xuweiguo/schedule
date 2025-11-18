package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type CronExprCompiled struct {
	years     map[int]struct{}
	months    [13]bool
	days      [32]bool
	weeks     [8]bool
	hourMinOK [24][60]bool
}

func CompileCronExpr(expr string) (*CronExprCompiled, error) {
	fields := strings.Fields(expr)
	if len(fields) != 6 {
		return nil, fmt.Errorf("cron expression should have 6 fields, got %d", len(fields))
	}
	c := &CronExprCompiled{}
	if err := fillYears(c, fields[0]); err != nil {
		return nil, err
	}
	if err := fillBoolField(fields[1], 1, 12, c.months[:]); err != nil {
		return nil, fmt.Errorf("month: %w", err)
	}
	if err := fillBoolField(fields[2], 1, 31, c.days[:]); err != nil {
		return nil, fmt.Errorf("day: %w", err)
	}
	if err := fillBoolField(fields[3], 1, 7, c.weeks[:]); err != nil {
		return nil, fmt.Errorf("week: %w", err)
	}
	if err := fillHourMinute(c, fields[4], fields[5]); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *CronExprCompiled) Match(t time.Time) bool {
	year := t.Year()
	if len(c.years) != 0 {
		if _, ok := c.years[year]; !ok {
			return false
		}
	}
	month := int(t.Month())
	if !c.months[month] {
		return false
	}
	day := t.Day()
	if !c.days[day] {
		return false
	}
	week := int(t.Weekday())
	if week == 0 {
		week = 7
	}
	if !c.weeks[week] {
		return false
	}
	hour := t.Hour()
	minute := t.Minute()
	if !c.hourMinOK[hour][minute] {
		return false
	}
	return true
}

func fillYears(c *CronExprCompiled, field string) error {
	values, any, err := parseField(field, mathMinYear, mathMaxYear)
	if err != nil {
		return fmt.Errorf("year: %w", err)
	}
	if any {
		c.years = nil
		return nil
	}
	c.years = make(map[int]struct{}, len(values))
	for _, v := range values {
		c.years[v] = struct{}{}
	}
	return nil
}

const (
	mathMinYear = 1970
	mathMaxYear = 2099
)

func fillBoolField(field string, min, max int, target []bool) error {
	values, any, err := parseField(field, min, max)
	if err != nil {
		return err
	}
	if any {
		for i := min; i <= max; i++ {
			target[i] = true
		}
		return nil
	}
	for _, v := range values {
		target[v] = true
	}
	return nil
}

func fillHourMinute(c *CronExprCompiled, hourField, minuteField string) error {
	hourParts := strings.Split(hourField, "|")
	minuteParts := strings.Split(minuteField, "|")
	if len(hourParts) > 1 {
		if len(minuteParts) != len(hourParts) {
			return fmt.Errorf("hour and minute parts count mismatch")
		}
		for i := range hourParts {
			if err := fillHourMinutePair(&c.hourMinOK, hourParts[i], minuteParts[i]); err != nil {
				return err
			}
		}
		return nil
	}
	if len(minuteParts) > 1 {
		return fmt.Errorf("minute has multiple parts but hour does not")
	}
	return fillHourMinutePair(&c.hourMinOK, hourField, minuteField)
}

func fillHourMinutePair(target *[24][60]bool, hourField, minuteField string) error {
	hours, anyHour, err := parseField(hourField, 0, 23)
	if err != nil {
		return fmt.Errorf("hour: %w", err)
	}
	minutes, anyMinute, err := parseField(minuteField, 0, 59)
	if err != nil {
		return fmt.Errorf("minute: %w", err)
	}
	var hourValues []int
	if anyHour {
		for i := 0; i < 24; i++ {
			hourValues = append(hourValues, i)
		}
	} else {
		hourValues = hours
	}
	var minuteValues []int
	if anyMinute {
		for i := 0; i < 60; i++ {
			minuteValues = append(minuteValues, i)
		}
	} else {
		minuteValues = minutes
	}
	for _, h := range hourValues {
		for _, m := range minuteValues {
			target[h][m] = true
		}
	}
	return nil
}

func parseField(field string, min, max int) ([]int, bool, error) {
	field = strings.TrimSpace(field)
	if field == "*" || field == "" {
		return nil, true, nil
	}
	tokens := splitTokens(field)
	var result []int
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if strings.Contains(token, "/") {
			parts := strings.SplitN(token, "/", 2)
			if len(parts) != 2 {
				return nil, false, fmt.Errorf("invalid step expression: %s", token)
			}
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, false, err
			}
			step, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, false, err
			}
			if step <= 0 {
				return nil, false, fmt.Errorf("step must be >0")
			}
			if start < min || start > max {
				return nil, false, fmt.Errorf("value %d out of range", start)
			}
			for v := start; v <= max; v += step {
				result = append(result, v)
			}
			continue
		}
		if strings.Contains(token, "-") {
			parts := strings.SplitN(token, "-", 2)
			if len(parts) != 2 {
				return nil, false, fmt.Errorf("invalid range expression: %s", token)
			}
			start, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, false, err
			}
			end, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, false, err
			}
			if start > end {
				return nil, false, fmt.Errorf("invalid range %d-%d", start, end)
			}
			if start < min || end > max {
				return nil, false, fmt.Errorf("range %d-%d out of bounds", start, end)
			}
			for v := start; v <= end; v++ {
				result = append(result, v)
			}
			continue
		}
		value, err := strconv.Atoi(token)
		if err != nil {
			return nil, false, err
		}
		if value < min || value > max {
			return nil, false, fmt.Errorf("value %d out of range", value)
		}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, false, fmt.Errorf("no valid values in field '%s'", field)
	}
	return result, false, nil
}

func splitTokens(field string) []string {
	field = strings.ReplaceAll(field, ",", " ")
	field = strings.TrimSpace(field)
	if field == "" {
		return nil
	}
	return strings.Fields(field)
}
