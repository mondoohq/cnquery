package resources

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/users"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

const (
	USER_CACHE_UID      = "uid"
	USER_CACHE_USERNAME = "username"
	USER_CACHE_GID      = "gid"
	USER_CACHE_HOME     = "home"
	USER_CACHE_SHELL    = "shell"
	USER_CACHE_ENABLED  = "enabled"
)

func (p *lumiUser) init(args *lumi.Args) (*lumi.Args, error) {
	uidValue, ok := (*args)[USER_CACHE_UID]

	// check if additional userdata is provided
	_, gok := (*args)[USER_CACHE_GID]
	usernameValue, uok := (*args)[USER_CACHE_USERNAME]

	// if only uid was provided, lets collect the info for the user
	if ok && !gok && !uok {
		uid, ok := uidValue.(int64)
		if !ok {
			return nil, errors.New("user> uid has invalid type")
		}

		// lets do minimal IO in initialize
		um, err := resolveOSUserManager(p.Runtime.Motor)
		if err != nil {
			return nil, errors.New("user> cannot find user manager")
		}

		// search for the user
		user, err := um.User(uid)
		if err != nil {
			return nil, err
		}

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(user, args)
	} else if uok && !ok {
		username, ok := usernameValue.(string)
		if !ok {
			return nil, errors.New("user> username has invalid type")
		}

		// we go a username as an initizator, which eg. is used by the groups resource
		// lets do minimal IO in initialize
		um, err := resolveOSUserManager(p.Runtime.Motor)
		if err != nil {
			return nil, errors.New("user> cannot find user manager")
		}

		userList, err := um.List()
		if err != nil {
			return nil, err
		}

		var foundUser *users.User

		// search for username
		for i := range userList {
			user := userList[i]
			if user.Username == username {
				foundUser = user
				break
			}
		}

		if foundUser == nil {
			return nil, errors.New("user> " + username + " does not exist")
		}

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(foundUser, args)
	}

	return args, nil
}

func (p *lumiUser) id() (string, error) {
	uid, err := p.Uid()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(uid, 10), nil
}

func (s *lumiUser) GetUsername() (string, error) {
	return "", errors.New("not implemented")
}

func (s *lumiUser) GetHome() (string, error) {
	return "", errors.New("not implemented")
}

func (s *lumiUser) GetShell() (string, error) {
	return "", errors.New("not implemented")
}

func (s *lumiUser) GetEnabled() (bool, error) {
	return false, errors.New("not implemented")
}

func (p *lumiUsers) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiUsers) id() (string, error) {
	return "users", nil
}

func (s *lumiUsers) GetList() ([]interface{}, error) {
	// find suitable user manager
	um, err := resolveOSUserManager(s.Runtime.Motor)
	if um == nil || err != nil {
		log.Warn().Err(err).Msg("lumi[users]> could not retrieve users list")
		return nil, errors.New("cannot find users manager")
	}

	// retrieve all system users
	users, err := um.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[users]> could not retrieve users list")
		return nil, errors.New("could not retrieve users list")
	}
	log.Debug().Int("users", len(users)).Msg("lumi[users]> found users")

	// convert to ]interface{}{}
	lumiUsers := []interface{}{}
	for i := range users {
		user := users[i]

		// set init arguments for the lumi user resource
		args := make(lumi.Args)

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(user, &args)

		e, err := newUser(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("user", user.Username).Msg("lumi[users]> could not create user resource")
			continue
		}

		lumiUsers = append(lumiUsers, e.(User))
	}

	return lumiUsers, nil
}

func copyUserDataToLumiArgs(user *users.User, args *lumi.Args) error {
	(*args)[USER_CACHE_USERNAME] = user.Username
	(*args)[USER_CACHE_UID] = user.Uid
	(*args)[USER_CACHE_GID] = user.Gid
	(*args)[USER_CACHE_HOME] = user.Home
	(*args)[USER_CACHE_SHELL] = user.Shell
	(*args)[USER_CACHE_ENABLED] = user.Enabled
	return nil
}

func resolveOSUserManager(motor *motor.Motor) (OSUserManager, error) {
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
	User(uid int64) (*users.User, error)
	List() ([]*users.User, error)
}

type LinuxUserManager struct {
	motor *motor.Motor
}

func (s *LinuxUserManager) Name() string {
	return "Linux User Manager"
}

func (s *LinuxUserManager) User(uid int64) (*users.User, error) {
	userList, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for uid
	for i := range userList {
		user := userList[i]
		if user.Uid == uid {
			return user, nil
		}
	}

	return nil, errors.New("user> " + strconv.FormatInt(uid, 10) + " does not exist")
}

func (s *LinuxUserManager) List() ([]*users.User, error) {
	f, err := s.motor.Transport.File("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return users.ParseEtcPasswd(f)
}

type OSXUserManager struct {
	motor *motor.Motor
}

func (s *OSXUserManager) Name() string {
	return "macOS User Manager"
}

func (s *OSXUserManager) User(uid int64) (*users.User, error) {
	userList, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for uid
	for i := range userList {
		user := userList[i]
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
func (s *OSXUserManager) List() ([]*users.User, error) {
	userMap := make(map[string]*users.User)

	// fetch all uids first
	f, err := s.motor.Transport.RunCommand("dscl . -list /Users UniqueID")
	if err != nil {
		return nil, err
	}

	m, err := users.ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		uid, err := strconv.ParseInt(m[k], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", k).Msg("could not parse uid")
		}

		userMap[k] = &users.User{
			Username: k,
			Uid:      uid,
		}
	}

	// fetch shells
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users UserShell")
	if err != nil {
		return nil, err
	}

	m, err = users.ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		userMap[k].Shell = m[k]
	}

	// fetch home
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users NFSHomeDirectory")
	if err != nil {
		return nil, err
	}

	m, err = users.ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		userMap[k].Home = m[k]
	}

	// fetch usernames
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users RealName")
	if err != nil {
		return nil, err
	}

	m, err = users.ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		userMap[k].Description = m[k]
	}

	// fetch gid
	f, err = s.motor.Transport.RunCommand("dscl . -list /Users PrimaryGroupID")
	if err != nil {
		return nil, err
	}

	m, err = users.ParseDsclListResult(f.Stdout)
	if err != nil {
		return nil, err
	}
	for k := range m {
		gid, err := strconv.ParseInt(m[k], 10, 0)
		if err != nil {
			log.Error().Err(err).Str("user", k).Msg("could not parse gid")
		}
		userMap[k].Gid = gid
	}

	// convert map to slice
	res := make([]*users.User, len(userMap))

	i := 0
	for k := range userMap {
		res[i] = userMap[k]
		i++
	}

	return res, nil
}
