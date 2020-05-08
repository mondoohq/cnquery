package groups

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

var (
	GROUP_OSX_DSCACHEUTIL_REGEX = regexp.MustCompile(`^(\S+):\s(.*?)$`)
)

func ParseDscacheutilResult(input io.Reader) ([]*Group, error) {

	groups := map[string]*Group{}

	add := func(group Group) {
		// a group must have a username, otherwise it is not valid
		// this will happen at least for the last item, where we got an empty row
		// we also need to eliminate duplicates, it happens on macos with dscacheutil -q group
		if len(group.ID) != 0 {
			groups[group.ID] = &group
		}
	}

	scanner := bufio.NewScanner(input)
	group := Group{Members: []string{}}

	var key string
	for scanner.Scan() {
		line := scanner.Text()

		// reset group definition once we reach an empty line
		if len(line) == 0 {
			add(group)
			group = Group{Members: []string{}}
		}

		m := GROUP_OSX_DSCACHEUTIL_REGEX.FindStringSubmatch(line)
		key = ""
		if m != nil {
			key = m[1]
		}

		// Parse the group content
		switch key {
		case "name":
			group.Name = strings.TrimSpace(m[2])
		case "password":
			// we ignore the password for now
		case "gid":
			gid, err := strconv.ParseInt(m[2], 10, 0)
			if err != nil {
				log.Error().Err(err).Str("group", m[0]).Msg("could not parse gid")
			}
			group.ID = m[2]
			group.Gid = gid
		case "users":
			content := strings.TrimSpace(m[2])
			if len(content) > 0 {
				group.Members = strings.Split(content, " ")
			}
		}
	}

	// if the last line is not an empty line we have things in flight, lets check it
	add(group)

	res := []*Group{}
	for k := range groups {
		res = append(res, groups[k])
	}

	return res, nil
}

type OSXGroupManager struct {
	motor *motor.Motor
}

func (s *OSXGroupManager) Name() string {
	return "macOS Group Manager"
}

func (s *OSXGroupManager) Group(id string) (*Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	return findGroup(groups, id)
}

func (s *OSXGroupManager) List() ([]*Group, error) {
	c, err := s.motor.Transport.RunCommand("dscacheutil -q group")
	if err != nil {
		return nil, err
	}
	return ParseDscacheutilResult(c.Stdout)
}
