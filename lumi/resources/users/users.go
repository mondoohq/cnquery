package users

import (
	"bufio"
	"io"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type User struct {
	Uid         int64
	Gid         int64
	Username    string
	Description string
	Shell       string
	Home        string
	Enabled     bool
}

var (
	USER_OSX_DSCL_REGEX = regexp.MustCompile(`(?m)^(\S*)\s*(\S*)$`)
)

// a good description of this file is available at:
// https://www.cyberciti.biz/faq/understanding-etcpasswd-file-format/
func ParseEtcPasswd(input io.Reader) ([]*User, error) {
	var users []*User
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()
		m := strings.Split(line, ":")

		// parse uid
		uid, err := strconv.ParseInt(m[2], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", m[0]).Msg("could not parse uid")
		}
		gid, err := strconv.ParseInt(m[3], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", m[0]).Msg("could not parse gid")
		}

		// bin:x:1:1:bin:/bin:/sbin/nologin
		users = append(users, &User{
			Username:    m[0],
			Uid:         uid,
			Gid:         gid,
			Description: m[4],
			Home:        m[5],
			Shell:       m[6],
		})
	}

	return users, nil
}

func ParseDsclListResult(input io.Reader) (map[string]string, error) {
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	userMap := make(map[string]string)
	m := USER_OSX_DSCL_REGEX.FindAllStringSubmatch(string(content), -1)
	for i := range m {
		key := m[i][1]
		value := m[i][2]

		if len(key) > 0 {
			userMap[key] = value
		}
	}
	return userMap, nil

}
