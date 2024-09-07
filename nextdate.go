package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func NextDate(now time.Time, date string, repeat string) (string, error) {
	var (
		err         error
		result_time time.Time
	)

	re := regexp.MustCompile(`^d ([1-9]\d?|1[0-4]\d|400)$`)
	if repeat == "" {
		return "", errors.New("repeat is missing")
	}

	format_time, err := time.Parse("20060102", date)
	if err != nil {
		return "", errors.New("error of parsing date")
	}

	if re.MatchString(repeat) {
		days_str := strings.Fields(repeat)[1]
		days, _ := strconv.Atoi(days_str)
		result_time = format_time.AddDate(0, 0, days)
		for {

			if result_time.After(now) {
				break
			}
			result_time = result_time.AddDate(0, 0, days)
		}
	} else if repeat == "y" {
		result_time = format_time.AddDate(1, 0, 0)
		for {
			if result_time.After(now) {
				break
			}
			result_time = result_time.AddDate(1, 0, 0)
		}
	} else {
		return "", errors.New("repeat has a wrong format")
	}

	return result_time.Format("20060102"), nil
}
