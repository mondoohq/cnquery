package processes

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"

	"github.com/rs/zerolog/log"
)

var (
	UNIX_PS_REGEX = regexp.MustCompile(`(?m)^\s*([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ]+)\s+([^ ].*)$`)
)

type ProcessEntry struct {
	Pid     int64
	CPU     string
	Mem     string
	Vsz     string
	Rss     string
	Tty     string
	Stat    string
	Start   string
	Time    string
	Uid     int64
	Command string
}

func ParseUnixPsResult(input io.Reader) ([]*ProcessEntry, error) {
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	m := UNIX_PS_REGEX.FindAllStringSubmatch(string(content), -1)

	var processes = []*ProcessEntry{}
	for k, value := range m {
		if value[1] == "PID" {
			// header
			continue
		}

		log.Debug().Int("key", k).Str("value", fmt.Sprintf("%v", value)).Msg("line")

		pid, err := strconv.ParseInt(value[1], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse pid " + value[1])
			continue
		}
		uid, err := strconv.ParseInt(value[10], 10, 64)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse uid " + value[10])
			continue
		}

		// PID %CPU %MEM    VSZ   RSS TT       STAT  STARTED     TIME   UID COMMAND
		p := &ProcessEntry{
			Pid:     pid,
			CPU:     value[2],
			Mem:     value[3],
			Vsz:     value[4],
			Rss:     value[5],
			Tty:     value[6],
			Stat:    value[7],
			Start:   value[8],
			Time:    value[9],
			Uid:     uid,
			Command: value[11],
		}
		processes = append(processes, p)
	}

	return processes, nil
}
