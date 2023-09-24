// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources/groups"
)

type mqlGroupInternal struct {
	membersArr []string
}

func (x *mqlGroup) id() (string, error) {
	var id string
	if len(x.Sid.Data) > 0 {
		id = x.Sid.Data
	} else {
		id = strconv.FormatInt(x.Gid.Data, 10)
	}

	return "group/" + id + "/" + x.Name.Data, nil
}

func (x *mqlGroup) members() ([]interface{}, error) {
	raw, err := CreateResource(x.MqlRuntime, "users", nil)
	if err != nil {
		return nil, errors.New("cannot get users info for group: " + err.Error())
	}
	users := raw.(*mqlUsers)

	if err := users.refreshCache(nil); err != nil {
		return nil, err
	}

	res := make([]interface{}, len(x.membersArr))
	for i, name := range x.membersArr {
		res[i] = users.usersByName[name]
	}

	return res, nil
}

type mqlGroupsInternal struct {
	lock       sync.Mutex
	groupsByID map[int64]*mqlGroup
}

func (x *mqlGroups) list() ([]interface{}, error) {
	x.lock.Lock()
	defer x.lock.Unlock()

	// in the unlikely case that we get called twice into the same method,
	// any subsequent calls are locked until user detection finishes; at this point
	// we only need to return a non-nil error field and it will pull the data from cache
	if x.groupsByID != nil {
		return nil, nil
	}
	x.groupsByID = map[int64]*mqlGroup{}

	conn := x.MqlRuntime.Connection.(shared.Connection)
	gm, err := groups.ResolveManager(conn)
	if gm == nil || err != nil {
		return nil, errors.New("cannot find groups manager")
	}

	groups, err := gm.List()
	if err != nil {
		return nil, errors.New("could not retrieve groups list")
	}

	var res []interface{}
	for i := range groups {
		group := groups[i]
		nu, err := CreateResource(x.MqlRuntime, "group", map[string]*llx.RawData{
			"name": llx.StringData(group.Name),
			"gid":  llx.IntData(group.Gid),
			"sid":  llx.StringData(group.Sid),
		})
		if err != nil {
			return nil, err
		}

		res = append(res, nu)

		g := nu.(*mqlGroup)
		g.membersArr = group.Members
		x.groupsByID[g.Gid.Data] = g
	}

	return res, nil
}

func (x *mqlGroups) findID(id int64) (*mqlGroup, error) {
	if x := x.GetList(); x.Error != nil {
		return nil, x.Error
	}

	res, ok := x.mqlGroupsInternal.groupsByID[id]
	if !ok {
		return nil, errors.New("cannot find group for uid " + strconv.Itoa(int(id)))
	}
	return res, nil
}
