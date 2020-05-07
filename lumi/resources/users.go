package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/users"
)

const (
	USER_CACHE_ID       = "id"
	USER_CACHE_UID      = "uid"
	USER_CACHE_USERNAME = "username"
	USER_CACHE_GID      = "gid"
	USER_CACHE_SID      = "sid"
	USER_CACHE_HOME     = "home"
	USER_CACHE_SHELL    = "shell"
	USER_CACHE_ENABLED  = "enabled"
)

func copyUserDataToLumiArgs(user *users.User, args *lumi.Args) error {
	(*args)[USER_CACHE_ID] = user.ID
	(*args)[USER_CACHE_USERNAME] = user.Username
	(*args)[USER_CACHE_UID] = user.Uid
	(*args)[USER_CACHE_GID] = user.Gid
	(*args)[USER_CACHE_SID] = user.Sid
	(*args)[USER_CACHE_HOME] = user.Home
	(*args)[USER_CACHE_SHELL] = user.Shell
	(*args)[USER_CACHE_ENABLED] = user.Enabled
	return nil
}

func (u *lumiUser) init(args *lumi.Args) (*lumi.Args, User, error) {
	idValue, ok := (*args)[USER_CACHE_ID]

	// check if additional userdata is provided
	_, gok := (*args)[USER_CACHE_GID]
	usernameValue, uok := (*args)[USER_CACHE_USERNAME]

	// if only uid was provided, lets collect the info for the user
	if ok && !gok && !uok {
		// lets do minimal IO in initialize
		um, err := users.ResolveManager(u.Runtime.Motor)
		if err != nil {
			return nil, nil, errors.New("user> cannot find user manager")
		}

		id := idValue.(string)

		// search for the user
		user, err := um.User(id)
		if err != nil {
			return nil, nil, err
		}

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(user, args)
	} else if uok && !ok {
		username, ok := usernameValue.(string)
		if !ok {
			return nil, nil, errors.New("user> username has invalid type")
		}

		// we go a username as an initizator, which eg. is used by the groups resource
		// lets do minimal IO in initialize
		um, err := users.ResolveManager(u.Runtime.Motor)
		if err != nil {
			return nil, nil, errors.New("user> cannot find user manager")
		}

		userList, err := um.List()
		if err != nil {
			return nil, nil, err
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
			return nil, nil, errors.New("user> " + username + " does not exist")
		}

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(foundUser, args)
	}

	return args, nil, nil
}

func (u *lumiUser) id() (string, error) {
	return u.Id()
}

func (u *lumiUsers) id() (string, error) {
	return "users", nil
}

func (u *lumiUsers) GetList() ([]interface{}, error) {
	// find suitable user manager
	um, err := users.ResolveManager(u.Runtime.Motor)
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

	// convert to interface{}{}
	lumiUsers := []interface{}{}
	for i := range users {
		user := users[i]

		lumiUser, err := u.Runtime.CreateResource("user",
			USER_CACHE_ID, user.ID,
			USER_CACHE_USERNAME, user.Username,
			USER_CACHE_UID, user.Uid,
			USER_CACHE_GID, user.Gid,
			USER_CACHE_SID, user.Sid,
			USER_CACHE_HOME, user.Home,
			USER_CACHE_SHELL, user.Shell,
			USER_CACHE_ENABLED, user.Enabled,
		)
		if err != nil {
			return nil, err
		}

		lumiUsers = append(lumiUsers, lumiUser.(User))
	}

	return lumiUsers, nil
}
