package core

import (
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/resources/packs/core/users"
)

const (
	USER_CACHE_UID      = "uid"
	USER_CACHE_USERNAME = "name"
	USER_CACHE_GID      = "gid"
	USER_CACHE_SID      = "sid"
	USER_CACHE_HOME     = "home"
	USER_CACHE_SHELL    = "shell"
	USER_CACHE_ENABLED  = "enabled"
)

func copyUserDataToLumiArgs(user *users.User, args *lumi.Args) error {
	(*args)[USER_CACHE_USERNAME] = user.Name
	(*args)[USER_CACHE_UID] = user.Uid
	(*args)[USER_CACHE_GID] = user.Gid
	(*args)[USER_CACHE_SID] = user.Sid
	(*args)[USER_CACHE_HOME] = user.Home
	(*args)[USER_CACHE_SHELL] = user.Shell
	(*args)[USER_CACHE_ENABLED] = user.Enabled
	return nil
}

func (u *lumiUser) init(args *lumi.Args) (*lumi.Args, User, error) {
	idValue := ""
	uidValue, uidOk := (*args)[USER_CACHE_UID]
	if uidOk {
		idValue = strconv.FormatInt(uidValue.(int64), 10)
	}
	// NOTE: windows send uid -1 therefore the value is set, but linux does not return a value for sid
	sidValue, sidOk := (*args)[USER_CACHE_SID]
	if sidOk {
		sid := sidValue.(string)
		if len(sid) > 0 {
			idValue = sid
		}
	}
	ok := uidOk || sidOk

	// check if additional userdata is provided
	_, gok := (*args)[USER_CACHE_GID]
	usernameValue, uok := (*args)[USER_CACHE_USERNAME]

	// if only uid was provided, lets collect the info for the user
	if ok && !gok && !uok {
		// lets do minimal IO in initialize
		um, err := users.ResolveManager(u.MotorRuntime.Motor)
		if err != nil {
			return nil, nil, errors.New("user> cannot find user manager")
		}

		id := idValue

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
		um, err := users.ResolveManager(u.MotorRuntime.Motor)
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
			if user.Name == username {
				foundUser = user
				break
			}
		}

		if foundUser == nil {
			return nil, nil, errors.New("user '" + username + "' does not exist")
		}

		// copy parsed user info to lumi args
		copyUserDataToLumiArgs(foundUser, args)
	}

	return args, nil, nil
}

func (u *lumiUser) id() (string, error) {
	uid, err := u.Uid()
	if err != nil {
		return "", err
	}

	sid, err := u.Sid()
	if err != nil {
		return "", err
	}

	name, err := u.Name()
	if err != nil {
		return "", err
	}

	id := strconv.FormatInt(uid, 10)
	if len(sid) > 0 {
		id = sid
	}

	return "user/" + id + "/" + name, nil
}

func (u *lumiUsers) id() (string, error) {
	return "users", nil
}

func (u *lumiUsers) GetList() ([]interface{}, error) {
	// find suitable user manager
	um, err := users.ResolveManager(u.MotorRuntime.Motor)
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
	namedMap := map[string]User{}
	for i := range users {
		user := users[i]

		lumiUser, err := u.MotorRuntime.CreateResource("user",
			USER_CACHE_USERNAME, user.Name,
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
		namedMap[user.Name] = lumiUser.(User)
	}

	u.Cache.Store("_map", &lumi.CacheEntry{Data: namedMap})
	return lumiUsers, nil
}

func (u *lumiUser) GetGroup() (interface{}, error) {
	gid, err := u.Gid()
	group, err := u.MotorRuntime.CreateResource("group",
		"id", strconv.FormatInt(gid, 10),
		"gid", gid)
	// Don't return the error just null
	// This is so we can check if a user's group exists
	if err != nil {
		return nil, nil
	}
	return group, nil
}

func (u *lumiUser) GetSshkeys() ([]interface{}, error) {
	osProvider, err := osProvider(u.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}

	home, err := u.Home()
	if err != nil {
		return nil, err
	}

	userSshPath := path.Join(home, ".ssh")

	fs := osProvider.FS()
	afutil := afero.Afero{Fs: fs}

	// check if use ssh directory exists
	exists, err := afutil.Exists(userSshPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return res, nil
	}

	filter := []string{"config"}

	// walk dir and search for all private keys
	potentialPrivateKeyFiles := []string{}
	err = afutil.Walk(userSshPath, func(path string, f os.FileInfo, err error) error {
		if f == nil || f.IsDir() {
			return nil
		}

		// eg. matches google_compute_known_hosts and known_hosts
		if strings.HasSuffix(f.Name(), ".pub") || strings.HasSuffix(f.Name(), "known_hosts") {
			return nil
		}

		for i := range filter {
			if f.Name() == filter[i] {
				return nil
			}
		}

		potentialPrivateKeyFiles = append(potentialPrivateKeyFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// iterate over files and check if the content is there
	for i := range potentialPrivateKeyFiles {
		log.Debug().Msg("load content from file")
		path := potentialPrivateKeyFiles[i]
		f, err := fs.Open(path)
		if err != nil {
			return nil, err
		}

		data, err := ioutil.ReadAll(f)
		f.Close()
		if err != nil {
			return nil, err
		}

		content := string(data)

		// check if content contains PRIVATE KEY
		isPrivateKey := strings.Contains(content, "PRIVATE KEY")
		// check if the key is encrypted ENCRYPTED
		isEncrypted := strings.Contains(content, "ENCRYPTED")

		if isPrivateKey {
			upk, err := u.MotorRuntime.CreateResource("privatekey",
				"pem", content,
				"encrypted", isEncrypted,
				"path", path,
			)
			if err != nil {
				return nil, err
			}
			res = append(res, upk.(Privatekey))
		}
	}

	return res, nil
}
