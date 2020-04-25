package resources

import (
	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/groups"
)

const (
	GROUP_CACHE_ID      = "id"
	GROUP_CACHE_NAME    = "name"
	GROUP_CACHE_GID     = "gid"
	GROUP_CACHE_SID     = "sid"
	GROUP_CACHE_MEMBERS = "members"

	GROUPS_MAP_ID = "groups_map"
)

func (g *lumiGroup) init(args *lumi.Args) (*lumi.Args, error) {
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
	obj, err := g.Runtime.CreateResource("groups")
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

func (g *lumiGroup) id() (string, error) {
	return g.Id()
}

func (g *lumiGroups) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
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
			GROUP_CACHE_ID, group.ID,
			GROUP_CACHE_NAME, group.Name,
			GROUP_CACHE_GID, group.Gid,
			GROUP_CACHE_SID, group.Sid,
			GROUP_CACHE_MEMBERS, members,
		)
		if err != nil {
			return nil, err
		}

		lumiGroups = append(lumiGroups, lumiGroup.(Group))
		namedMap[group.ID] = lumiGroup.(Group)
	}

	g.Cache.Store(GROUPS_MAP_ID, &lumi.CacheEntry{Data: namedMap})

	return lumiGroups, nil
}
