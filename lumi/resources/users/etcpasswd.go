package users

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

// a good description of this file is available at:
// https://www.cyberciti.biz/faq/understanding-etcpasswd-file-format/
func ParseEtcPasswd(input io.Reader) ([]*User, error) {
	var users []*User
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := scanner.Text()

		// check if line starts with #
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		m := strings.Split(line, ":")

		if len(m) >= 7 {
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
				ID:          m[2],
				Name:        m[0],
				Uid:         uid,
				Gid:         gid,
				Description: m[4],
				Home:        m[5],
				Shell:       m[6],
			})
		}
	}

	return users, nil
}

type UnixUserManager struct {
	motor *motor.Motor
}

func (s *UnixUserManager) Name() string {
	return "Unix User Manager"
}

func (s *UnixUserManager) User(id string) (*User, error) {
	users, err := s.List()
	if err != nil {
		return nil, err
	}

	return findUser(users, id)
}

func (s *UnixUserManager) List() ([]*User, error) {
	f, err := s.motor.Transport.FS().Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseEtcPasswd(f)
}
