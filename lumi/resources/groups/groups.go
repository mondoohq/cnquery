package groups

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	GROUP_OSX_DSCACHEUTIL_REGEX = regexp.MustCompile(`^(\S+):\s(.*?)$`)
)

type Group struct {
	Gid     int64
	Name    string
	Members []string
}

// a good description of this file is available at:
// https://www.cyberciti.biz/faq/understanding-etcgroup-file/
func ParseEtcGroup(input io.Reader) ([]*Group, error) {
	var groups []*Group
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := strings.Split(line, ":")

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
			Gid:     gid,
			Name:    m[0],
			Members: members,
		})
	}

	return groups, nil
}

func ParseDscacheutilResult(input io.Reader) ([]*Group, error) {

	groups := []*Group{}

	add := func(group Group) {
		// a group must have a username, otherwise it is not valid
		// this will happen at least for the last item, where we got an empty row
		if len(group.Name) != 0 {
			groups = append(groups, &group)
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
	return groups, nil
}
