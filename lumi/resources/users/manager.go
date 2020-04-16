package users

import (
	"errors"

	motor "go.mondoo.io/mondoo/motor/motoros"
)

type User struct {
	ID          string
	Uid         int64
	Gid         int64
	Sid         string
	Username    string
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

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	// check darwin before unix since darwin is also a unix
	if platform.IsFamily("darwin") {
		um = &OSXUserManager{motor: motor}
	} else if platform.IsFamily("unix") {
		um = &UnixUserManager{motor: motor}
	} else if platform.IsFamily("windows") {
		um = &WindowsUserManager{motor: motor}
	}

	if um == nil {
		return nil, errors.New("could not detect suitable group manager for platform: " + platform.Name)
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
