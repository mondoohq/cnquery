// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package uptime

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/providers/os/connection/shared"
)

// UnixUptimeRegex parses `uptime` output across the formats the tools
// actually print: "up 1 day, 5:29", "up 16 min", and — when the minutes
// component is exactly zero — "up 11 days, 16 hrs" (macOS prints "N hrs"
// instead of "N:00" for one minute every hour, so missing that form made
// the uptime resource error once an hour).
//
// Solaris pluralizes with a parenthesized suffix instead of a bare "s",
// printing "up 26 min(s)" and "up 5 day(s), 21:03", so each unit accepts
// that spelling too.
var UnixUptimeRegex = regexp.MustCompile(`^.*up[\s]*(?:\s*(\d+)\s(day(?:s|\(s\))?),)*(?:\s*(\d+)\s(hr(?:s|\(s\))?),)*(?:\s*(\d+)\s(min(?:s|\(s\))?),)*(?:\s+([\d:]+),\s)*\s*(?:(\d+)\suser[s]*,\s)*\s*load\s+average[s]*:\s+(\d+[\.,]\d+)[,\s]+(\d+[\.,]\d+)[,\s]+(\d+[\.,]\d+)\s*$`)

type UnixUptimeResult struct {
	Duration           int64
	Users              int
	LoadOneMinute      float64
	LoadFiveMinutes    float64
	LoadFifteenMinutes float64
}

func unixDuration(date, measure string) (int64, error) {
	// calculate the time x * days / minutes + hours ( m[1]*m[2] + m[3])
	duration, err := strconv.ParseInt(date, 10, 64)
	if err != nil {
		return 0, err
	}

	// "days", "day(s)" and "day" all mean the same unit
	measure = strings.TrimSuffix(measure, "(s)")
	measure = strings.TrimSuffix(measure, "s")

	switch measure {
	case "day":
		duration = duration * 24 * int64(time.Hour)
	case "hr":
		duration = duration * int64(time.Hour)
	case "min":
		duration = duration * int64(time.Minute)
	}
	return duration, nil
}

func ParseUnixUptime(uptime string) (*UnixUptimeResult, error) {
	log.Debug().Str("uptime", uptime).Msg("parse")
	m := UnixUptimeRegex.FindStringSubmatch(uptime)

	if len(m) != 12 {
		return nil, fmt.Errorf("could not parse uptime: %s", uptime)
	}

	var duration int64
	var err error

	// parse days
	if len(m[2]) > 0 {
		unixDuration, err := unixDuration(m[1], m[2])
		if err != nil {
			return nil, err
		}
		duration = duration + unixDuration
	}

	// parse whole hours ("16 hrs" — printed instead of "16:00" when the
	// minutes component is exactly zero)
	if len(m[4]) > 0 {
		unixDuration, err := unixDuration(m[3], m[4])
		if err != nil {
			return nil, err
		}
		duration = duration + unixDuration
	}

	// parse mins
	if len(m[6]) > 0 {
		unixDuration, err := unixDuration(m[5], m[6])
		if err != nil {
			return nil, err
		}
		duration = duration + unixDuration
	}

	// add optional hours
	if len(m[7]) > 0 {
		hours := strings.Split(m[7], ":")
		if len(hours) == 2 {
			// log.Debug().Msg("parse hour")
			hh, err := strconv.ParseInt(hours[0], 10, 64)
			if err != nil {
				return nil, err
			}

			// log.Debug().Msg("parse minutes")
			mm, err := strconv.ParseInt(hours[1], 10, 64)
			if err != nil {
				return nil, err
			}

			duration = duration + hh*int64(time.Hour) + mm*int64(time.Minute)
		} else {
			return nil, fmt.Errorf("could not parse uptime hours: %s", uptime)
		}
	}

	// users is optional and is not returned on alpine
	users := 0
	if len(m[8]) > 0 {
		users, err = strconv.Atoi(m[8])
		if err != nil {
			return nil, err
		}
	}

	loadOneMinute, err := strconv.ParseFloat(strings.Replace(m[9], ",", ".", 1), 64)
	if err != nil {
		return nil, err
	}

	loadFiveMinutes, err := strconv.ParseFloat(strings.Replace(m[10], ",", ".", 1), 64)
	if err != nil {
		return nil, err
	}

	loadFifteenMinutes, err := strconv.ParseFloat(strings.Replace(m[11], ",", ".", 1), 64)
	if err != nil {
		return nil, err
	}

	return &UnixUptimeResult{
		Duration:           duration,
		Users:              users,
		LoadOneMinute:      loadOneMinute,
		LoadFiveMinutes:    loadFiveMinutes,
		LoadFifteenMinutes: loadFifteenMinutes,
	}, nil
}

type Unix struct {
	conn shared.Connection
}

func (s *Unix) Name() string {
	return "Unix Uptime"
}

func (s *Unix) Duration() (time.Duration, error) {
	cmd, err := s.conn.RunCommand("uptime")
	if err != nil {
		return 0, err
	}

	ut, err := s.parse(cmd.Stdout)
	if err != nil {
		return 0, err
	}

	return time.Duration(ut.Duration), nil
}

func (s *Unix) parse(r io.Reader) (*UnixUptimeResult, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseUnixUptime(string(content))
}
