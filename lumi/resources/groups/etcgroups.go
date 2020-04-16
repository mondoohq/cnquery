package groups

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

// a good description of this file is available at:
// https://www.cyberciti.biz/faq/understanding-etcgroup-file/
func ParseEtcGroup(input io.Reader) ([]*Group, error) {
	var groups []*Group
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// check if line starts with #
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		m := strings.Split(line, ":")
		if len(m) >= 4 {
			// parse gid
			gid, err := strconv.ParseInt(m[2], 10, 0)
			if err != nil {
				log.Error().Err(err).Str("group", m[0]).Msg("could not parse gid")
			}

			// extract usernames
			members := []string{}
			if len(m[3]) > 0 {
				members = strings.Split(m[3], ",")
			}

			// vagrant:x:1000:vagrant
			groups = append(groups, &Group{
				ID:      m[2],
				Gid:     gid,
				Name:    m[0],
				Members: members,
			})
		} else {
			log.Warn().Str("line", line).Msg("cannot parse etc group entry")
		}
	}

	return groups, nil
}

type UnixGroupManager struct {
	motor *motor.Motor
}

func (s *UnixGroupManager) Name() string {
	return "Unix Group Manager"
}

func (s *UnixGroupManager) Group(id string) (*Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	return findGroup(groups, id)
}

func (s *UnixGroupManager) List() ([]*Group, error) {
	f, err := s.motor.Transport.File("/etc/group")
	if err != nil {
		return nil, err
	}

	defer f.Close()
	return ParseEtcGroup(f)
}
