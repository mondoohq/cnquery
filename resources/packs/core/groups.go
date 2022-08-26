package core

import (
	"errors"
	"strconv"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/groups"
)

func (g *mqlGroup) init(args *resources.Args) (*resources.Args, Group, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	id := (*args)["id"]
	if id == nil {
		return args, nil, nil
	}

	idS, ok := id.(string)
	if !ok {
		return args, nil, nil
	}

	// initialize groups resource
	obj, err := g.MotorRuntime.CreateResource("groups")
	if err != nil {
		return nil, nil, err
	}
	groups := obj.(Groups)

	_, err = groups.List()
	if err != nil {
		return nil, nil, err
	}

	c, ok := groups.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, nil, errors.New("cannot get map of groups")
	}
	cmap := c.Data.(map[string]Group)

	group := cmap[idS]
	if group != nil {
		return nil, group, nil
	}

	(*args)["gid"] = ""
	(*args)["sid"] = ""
	(*args)["name"] = ""
	(*args)["members"] = ""

	return args, nil, nil
}

func (g *mqlGroup) id() (string, error) {
	gid, err := g.Gid()
	if err != nil {
		return "", err
	}

	sid, err := g.Sid()
	if err != nil {
		return "", err
	}

	name, err := g.Name()
	if err != nil {
		return "", err
	}

	id := strconv.FormatInt(gid, 10)
	if len(sid) > 0 {
		id = sid
	}

	return "group/" + id + "/" + name, nil
}

func (g *mqlGroup) GetMembers() ([]interface{}, error) {
	// get cached users list
	obj, err := g.MotorRuntime.CreateResource("users")
	if err != nil {
		return nil, err
	}
	users := obj.(Users)

	_, err = users.List()
	if err != nil {
		return nil, err
	}

	c, ok := users.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, errors.New("cannot get map of groups")
	}
	cmap := c.Data.(map[string]User)

	// read members for this groups
	m, ok := g.MqlResource().Cache.Load("_members")
	if !ok {
		return nil, errors.New("cannot get map of group members")
	}
	groupMembers := m.Data.([]string)

	// TODO: we may want to reconsider to do this here, it should be an async method members()
	// therefore we may just want to store the references here
	var members []interface{}
	for i := range groupMembers {
		username := groupMembers[i]

		usr := cmap[username]
		if usr != nil {
			members = append(members, usr)
			continue
		}

		// if the user cannot be found, we init it as an empty user
		mqlUser, err := g.MotorRuntime.CreateResource("user",
			"username", username,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, mqlUser.(User))
	}

	return members, nil
}

func (g *mqlGroups) id() (string, error) {
	return "groups", nil
}

func (g *mqlGroups) GetList() ([]interface{}, error) {
	// find suitable groups manager
	gm, err := groups.ResolveManager(g.MotorRuntime.Motor)
	if gm == nil || err != nil {
		log.Warn().Err(err).Msg("mql[groups]> could not retrieve groups list")
		return nil, errors.New("cannot find groups manager")
	}

	// retrieve all system groups
	groups, err := gm.List()
	if err != nil {
		log.Warn().Err(err).Msg("mql[groups]> could not retrieve groups list")
		return nil, errors.New("could not retrieve groups list")
	}
	log.Debug().Int("groups", len(groups)).Msg("mql[groups]> found groups")

	// convert to interface{}{}
	mqlGroups := []interface{}{}
	namedMap := map[string]Group{}

	for i := range groups {
		group := groups[i]

		mqlGroup, err := g.MotorRuntime.CreateResource("group",
			"name", group.Name,
			"gid", group.Gid,
			"sid", group.Sid,
		)
		if err != nil {
			return nil, err
		}

		// store group members into group resources for later access
		lg := mqlGroup.(Group)
		lg.MqlResource().Cache.Store("_members", &resources.CacheEntry{Data: group.Members})

		mqlGroups = append(mqlGroups, lg)
		namedMap[group.ID] = mqlGroup.(Group)
	}

	g.Cache.Store("_map", &resources.CacheEntry{Data: namedMap})

	return mqlGroups, nil
}
