package groups

import (
	"errors"
)

type Group struct {
	ID      string // is the string representation of gid on linux/unix and sid on windows
	Gid     int64
	Sid     string
	Name    string
	Members []string
}

type OSGroupManager interface {
	Name() string
	Group(id string) (*Group, error)
	List() ([]*Group, error)
}

func findGroup(groups []*Group, id string) (*Group, error) {
	// search for id
	for i := range groups {
		group := groups[i]
		if group.ID == id {
			return group, nil
		}
	}

	return nil, errors.New("group> " + id + " does not exist")
}
