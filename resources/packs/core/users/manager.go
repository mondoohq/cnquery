package users

import (
	"errors"

	"go.mondoo.io/mondoo/motor/providers/os"

	"go.mondoo.io/mondoo/motor"
)

type User struct {
	ID          string
	Uid         int64
	Gid         int64
	Sid         string
	Name        string
	Description string
	Shell       string
	Home        string
	Enabled     bool
}

type OSUserManager interface {
	Name() string
	User(id string) (*User, error)
	List() ([]*User, error)
}

func ResolveManager(motor *motor.Motor) (OSUserManager, error) {
	var um OSUserManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("process manager is not supported for platform: " + pf.Name)
	}

	// check darwin before unix since darwin is also a unix
	if pf.IsFamily("darwin") {
		um = &OSXUserManager{provider: osProvider}
	} else if pf.IsFamily("unix") {
		um = &UnixUserManager{provider: osProvider}
	} else if pf.IsFamily("windows") {
		um = &WindowsUserManager{provider: osProvider}
	}

	if um == nil {
		return nil, errors.New("could not detect suitable group manager for platform: " + pf.Name)
	}

	return um, nil
}

func findUser(users []*User, id string) (*User, error) {
	// search for id
	for i := range users {
		user := users[i]
		if user.ID == id {
			return user, nil
		}
	}

	return nil, errors.New("user> " + id + " does not exist")
}
