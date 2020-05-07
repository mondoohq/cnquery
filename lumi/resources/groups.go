package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/groups"
)

func (g *lumiGroup) init(args *lumi.Args) (*lumi.Args, Group, error) {
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
	obj, err := g.Runtime.CreateResource("groups")
	if err != nil {
		return nil, nil, err
	}
	groups := obj.(Groups)

	_, err = groups.List()
	if err != nil {
		return nil, nil, err
	}

	c, ok := groups.LumiResource().Cache.Load("_map")
	if !ok {
		return nil, nil, errors.New("Cannot get map of packages")
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

func (g *lumiGroup) id() (string, error) {
	return g.Id()
}

func (g *lumiGroups) id() (string, error) {
	return "groups", nil
}

func (g *lumiGroups) GetList() ([]interface{}, error) {
	// find suitable groups manager
	gm, err := groups.ResolveManager(g.Runtime.Motor)
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

	// convert to interface{}{}
	lumiGroups := []interface{}{}
	namedMap := map[string]Group{}

	for i := range groups {
		group := groups[i]

		// TODO: we may want to reconsider to do this here, it should be an async method members()
		// therefore we may just want to store the references here
		var members []interface{}
		for i := range group.Members {
			username := group.Members[i]

			lumiUser, err := g.Runtime.CreateResource("user",
				"username", username,
			)
			if err != nil {
				return nil, err
			}
			members = append(members, lumiUser.(User))
		}

		lumiGroup, err := g.Runtime.CreateResource("group",
			"id", group.ID,
			"name", group.Name,
			"gid", group.Gid,
			"sid", group.Sid,
			"members", members,
		)
		if err != nil {
			return nil, err
		}

		lumiGroups = append(lumiGroups, lumiGroup.(Group))
		namedMap[group.ID] = lumiGroup.(Group)
	}

	g.Cache.Store("_map", &lumi.CacheEntry{Data: namedMap})

	return lumiGroups, nil
}
