package groups

import (
	"errors"
	"strconv"

	"go.mondoo.io/mondoo/motor"
)

func ResolveManager(motor *motor.Motor) (OSGroupManager, error) {
	var gm OSGroupManager

	platform, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			gm = &UnixGroupManager{motor: motor}
			break
		} else if platform.Family[i] == "darwin" {
			gm = &OSXGroupManager{motor: motor}
			break
		}
	}

	return gm, nil
}

type OSGroupManager interface {
	Name() string
	Group(gid int64) (*Group, error)
	List() ([]*Group, error)
}

type UnixGroupManager struct {
	motor *motor.Motor
}

func (s *UnixGroupManager) Name() string {
	return "Unix Group Manager"
}

func (s *UnixGroupManager) Group(gid int64) (*Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for gid
	for i := range groups {
		group := groups[i]
		if group.Gid == gid {
			return group, nil
		}
	}

	return nil, errors.New("group> " + strconv.FormatInt(gid, 10) + " does not exist")
}

func (s *UnixGroupManager) List() ([]*Group, error) {
	f, err := s.motor.Transport.File("/etc/group")
	if err != nil {
		return nil, err
	}

	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ParseEtcGroup(r)
}

type OSXGroupManager struct {
	motor *motor.Motor
}

func (s *OSXGroupManager) Name() string {
	return "macOS Group Manager"
}

func (s *OSXGroupManager) Group(gid int64) (*Group, error) {
	groups, err := s.List()
	if err != nil {
		return nil, err
	}

	// search for gid
	for i := range groups {
		group := groups[i]
		if group.Gid == gid {
			return group, nil
		}
	}

	return nil, errors.New("group> " + strconv.FormatInt(gid, 10) + " does not exist")
}

func (s *OSXGroupManager) List() ([]*Group, error) {
	c, err := s.motor.Transport.RunCommand("dscacheutil -q group")
	if err != nil {
		return nil, err
	}
	return ParseDscacheutilResult(c.Stdout)
}
