package uptime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	UnixUptimeRegex = regexp.MustCompile(`^.*up[\s]*(\d+)\s(day[s]*|min),(?:\s+([\d:]+),)*\s+(\d+)\susers,\s+load\s+average[s]*:\s+([\d\.]+)[,\s]+([\d\.]+)[,\s]+([\d\.]+)$`)
)

type UnixUptime struct {
	Time               int64
	Users              int
	LoadOneMinute      float64
	LoadFiveMinutes    float64
	LoadFifteenMinutes float64
}

func ParseUnixUptime(uptime string) (*UnixUptime, error) {

	m := UnixUptimeRegex.FindStringSubmatch(uptime)

	if len(m) != 8 {
		return nil, fmt.Errorf("could not parse uptime: %s", uptime)
	}

	// caclulate the time x * days / minutes + hours ( m[1]*m[2] + m[3])
	duration, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil, err
	}

	switch m[2] {
	case "day":
		fallthrough
	case "days":
		duration = duration * 24 * int64(time.Hour)
	case "min":
		duration = duration * int64(time.Minute)
	}

	// add optional hours
	if len(m[3]) > 0 {
		hours := strings.Split(m[3], ":")
		if len(hours) == 2 {
			hh, err := strconv.ParseInt(hours[0], 10, 64)
			if err != nil {
				return nil, err
			}

			mm, err := strconv.ParseInt(hours[1], 10, 64)
			if err != nil {
				return nil, err
			}

			duration = duration + hh*int64(time.Hour) + mm*int64(time.Minute)

		} else {
			return nil, fmt.Errorf("could not parse uptime hours: %s", uptime)
		}

	}

	users, err := strconv.Atoi(m[4])
	if err != nil {
		return nil, err
	}

	loadOneMinute, err := strconv.ParseFloat(m[5], 64)
	if err != nil {
		return nil, err
	}

	loadFiveMinutes, err := strconv.ParseFloat(m[6], 64)
	if err != nil {
		return nil, err
	}

	loadFifteenMinutes, err := strconv.ParseFloat(m[7], 64)
	if err != nil {
		return nil, err
	}

	return &UnixUptime{
		Time:               duration,
		Users:              users,
		LoadOneMinute:      loadOneMinute,
		LoadFiveMinutes:    loadFiveMinutes,
		LoadFifteenMinutes: loadFifteenMinutes,
	}, nil
}
