package resources

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/groups"
	motor "go.mondoo.io/mondoo/motor/motoros"
)

const (
	GROUP_CACHE_ID      = "id"
	GROUP_CACHE_NAME    = "name"
	GROUP_CACHE_GID     = "gid"
	GROUP_CACHE_SID     = "sid"
	GROUP_CACHE_MEMBERS = "members"

	GROUPS_MAP_ID = "groups_map"
)

func (p *lumiGroup) init(args *lumi.Args) (*lumi.Args, error) {
	if len(*args) > 2 {
		return args, nil
	}

	id := (*args)["id"]
	if id == nil {
		return args, nil
	}

	idS, ok := id.(string)
	if !ok {
		return args, nil
	}

	// initialize groups resource
	obj, err := p.Runtime.CreateResource("groups")
	if err != nil {
		return nil, err
	}
	groups := obj.(Groups)

	_, err = groups.List()
	if err != nil {
		return nil, err
	}

	c, ok := groups.LumiResource().Cache.Load(GROUPS_MAP_ID)
	if !ok {
		return nil, errors.New("Cannot get map of packages")
	}
	cmap := c.Data.(map[string]Group)

	group := cmap[idS]
	if group == nil {
		(*args)["gid"] = ""
		(*args)["sid"] = ""
		(*args)["name"] = ""
		(*args)["members"] = ""
	} else {
		// TODO: do this instead of duplicating it!
		// (*args)["id"] = pkg.LumiResource().Id
		(*args)["gid"], _ = group.Gid()
		(*args)["sid"], _ = group.Sid()
		(*args)["name"], _ = group.Name()
		(*args)["members"], _ = group.Members()
	}
	return args, nil

}

func (p *lumiGroup) id() (string, error) {
	return p.Id()
}

func (p *lumiGroups) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiGroups) id() (string, error) {
	return "groups", nil
}

func (s *lumiGroups) GetList() ([]interface{}, error) {
	// find suitable groups manager
	gm, err := resolveOSGroupManager(s.Runtime.Motor)
	if gm == nil || err != nil {
		log.Warn().Err(err).Msg("lumi[groups]> could not retrieve groups list")
		return nil, errors.New("cannot find groups manager")
	}

	// retrieve all system groups
	groups, err := gm.List()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[groups]> could not retrieve groups list")
		return nil, errors.New("could not retrieve groups list")
	}
	log.Debug().Int("groups", len(groups)).Msg("lumi[groups]> found groups")

	// convert to ]interface{}{}
	lumiGroups := []interface{}{}
	namedMap := map[string]Group{}

	for i := range groups {
		group := groups[i]

		// set init arguments for the lumi group resource
		args := make(lumi.Args)

		// copy parsed user info to lumi args
		s.copyGroupDataToLumiArgs(group, &args)

		e, err := newGroup(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("group", group.Name).Msg("lumi[users]> could not create group resource")
			continue
		}

		lumiGroups = append(lumiGroups, e.(Group))
		namedMap[group.ID] = e.(Group)
	}

	s.Cache.Store(GROUPS_MAP_ID, &lumi.CacheEntry{Data: namedMap})

	return lumiGroups, nil
}

func (s *lumiGroups) copyGroupDataToLumiArgs(group *groups.Group, args *lumi.Args) error {
	(*args)[GROUP_CACHE_ID] = group.ID
	(*args)[GROUP_CACHE_NAME] = group.Name
	(*args)[GROUP_CACHE_GID] = group.Gid
	(*args)[GROUP_CACHE_SID] = group.Sid

	var members []interface{}
	for i := range group.Members {
		username := group.Members[i]
		// convert group.members into lumi user objects
		args := make(lumi.Args)
		args[USER_CACHE_USERNAME] = username

		e, err := newUser(s.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("user", username).Msg("lumi[groups]> could not create user resource")
			continue
		}

		members = append(members, e.(User))
	}

	(*args)[GROUP_CACHE_MEMBERS] = members
	return nil
}

func resolveOSGroupManager(motor *motor.Motor) (OSGroupManager, error) {
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
	Group(gid int64) (*groups.Group, error)
	List() ([]*groups.Group, error)
}

type UnixGroupManager struct {
	motor *motor.Motor
}

func (s *UnixGroupManager) Name() string {
	return "Unix Group Manager"
}

func (s *UnixGroupManager) Group(gid int64) (*groups.Group, error) {
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

func (s *UnixGroupManager) List() ([]*groups.Group, error) {
	f, err := s.motor.Transport.File("/etc/group")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return groups.ParseEtcGroup(f)
}

type OSXGroupManager struct {
	motor *motor.Motor
}

func (s *OSXGroupManager) Name() string {
	return "macOS Group Manager"
}

func (s *OSXGroupManager) Group(gid int64) (*groups.Group, error) {
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

func (s *OSXGroupManager) List() ([]*groups.Group, error) {
	c, err := s.motor.Transport.RunCommand("dscacheutil -q group")
	if err != nil {
		return nil, err
	}
	return groups.ParseDscacheutilResult(c.Stdout)
}
