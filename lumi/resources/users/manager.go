package users

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

func ResolveManager(motor *motor.Motor) (OSUserManager, error) {
	var um OSUserManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			um = &LinuxUserManager{motor: motor}
			break
		} else if platform.Family[i] == "darwin" {
			um = &OSXUserManager{motor: motor}
			break
		}
	}

	return um, nil
}

type OSUserManager interface {
	Name() string
	User(uid int64) (*User, error)
	List() ([]*User, error)
}

type LinuxUserManager struct {
	motor *motor.Motor
}

func (s *LinuxUserManager) Name() string {
	return "Linux User Manager"
}

func (s *LinuxUserManager) User(uid int64) (*User, error) {
	users, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for uid
	for i := range users {
		user := users[i]
		if user.Uid == uid {
			return user, nil
		}
	}

	return nil, errors.New("user> " + strconv.FormatInt(uid, 10) + " does not exist")
}

func (s *LinuxUserManager) List() ([]*User, error) {
	f, err := s.motor.Transport.File("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseEtcPasswd(f)
}

type OSXUserManager struct {
	motor *motor.Motor
}

func (s *OSXUserManager) Name() string {
	return "macOS User Manager"
}

func (s *OSXUserManager) User(uid int64) (*User, error) {
	users, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for uid
	for i := range users {
		user := users[i]
		if user.Uid == uid {
			return user, nil
		}
	}

	return nil, errors.New("user> " + strconv.FormatInt(uid, 10) + " does not exist")
}

// To retrieve all user information, we have two options:
//
// 1. fetch all users via `dscl . list /Users`
// 2. iterate over each user and fetch the data via
//    dscl -q . -read /Users/nobody NFSHomeDirectory PrimaryGroupID RecordName UniqueID UserShell
//
// This approach is not very effective since it requires O(n), there we use the option to fetch one
// value per list, which requires us to do 5 calls to fetch all information:
// dscl . -list /Users UserShell
// dscl . -list /Users UniqueID
// dscl . -list /Users NFSHomeDirectory
// dscl . -list /Users RecordName
// dscl . -list /Users RealName
func (s *OSXUserManager) List() ([]*User, error) {
	users := make(map[string]*User)

	// fetch all uids first
	f, err := s.motor.Transport.RunCommand("dscl . -list /Users UniqueID")
	if err != nil {
		return nil, err
	}

	m, err := ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		uid, err := strconv.ParseInt(m[k], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", k).Msg("could not parse uid")
		}

		users[k] = &User{
			Username: k,
			Uid:      uid,
		}
	}

	// fetch shells
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users UserShell")
	if err != nil {
		return nil, err
	}

	m, err = ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		users[k].Shell = m[k]
	}

	// fetch home
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users NFSHomeDirectory")
	if err != nil {
		return nil, err
	}

	m, err = ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		users[k].Home = m[k]
	}

	// fetch usernames
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users RealName")
	if err != nil {
		return nil, err
	}

	m, err = ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		users[k].Description = m[k]
	}

	// fetch gid
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users PrimaryGroupID")
	if err != nil {
		return nil, err
	}

	m, err = ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		gid, err := strconv.ParseInt(m[k], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", k).Msg("could not parse gid")
		}
		users[k].Gid = gid
	}

	// convert map to slice
	res := make([]*User, len(users))

	i := 0
	for k := range users {
		res[i] = users[k]
		i++
	}

	return res, nil
}
