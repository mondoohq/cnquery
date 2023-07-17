package resources

import (
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/cnquery/providers/os/connection/shared"
	"go.mondoo.com/cnquery/providers/os/resources/users"
)

func (x *mqlUser) id() (string, error) {
	var id string
	if len(x.Sid.Data) > 0 {
		id = x.Sid.Data
	} else {
		id = strconv.FormatInt(x.Uid.Data, 10)
	}

	return "user/" + id + "/" + x.Name.Data, nil
}

func (x *mqlUser) group(gid int64) (*mqlGroup, error) {
	raw, err := CreateResource(x.MqlRuntime, "groups", nil)
	if err != nil {
		return nil, errors.New("cannot get groups info for user: " + err.Error())
	}
	groups := raw.(*mqlGroups)
	return groups.findID(gid)
}

type mqlUsersInternal struct {
	lock        sync.Mutex
	usersByID   map[int64]*mqlUser
	usersByName map[string]*mqlUser
}

func (x *mqlUsers) list() ([]interface{}, error) {
	x.lock.Lock()
	defer x.lock.Unlock()

	// in the unlikely case that we get called twice into the same method,
	// any subsequent calls are locked until user detection finishes; at this point
	// we only need to return a non-nil error field and it will pull the data from cache
	if x.usersByID != nil {
		return nil, nil
	}
	x.usersByID = map[int64]*mqlUser{}
	x.usersByName = map[string]*mqlUser{}

	conn := x.MqlRuntime.Connection.(shared.Connection)
	um, err := users.ResolveManager(conn)
	if um == nil || err != nil {
		return nil, errors.New("cannot find users manager")
	}

	users, err := um.List()
	if err != nil {
		return nil, errors.New("could not retrieve users list")
	}

	var res []interface{}
	for i := range users {
		user := users[i]
		nu, err := CreateResource(x.MqlRuntime, "user", map[string]interface{}{
			"name":    user.Name,
			"uid":     user.Uid,
			"gid":     user.Gid,
			"sid":     user.Sid,
			"home":    user.Home,
			"shell":   user.Shell,
			"enabled": user.Enabled,
		})
		if err != nil {
			return nil, err
		}

		res = append(res, nu)

		u := nu.(*mqlUser)
		x.usersByID[u.Uid.Data] = u
		x.usersByName[u.Name.Data] = u
	}

	return res, nil
}

func (x *mqlUsers) findID(id int64) (*mqlUser, error) {
	if x := x.GetList(); x.Error != nil {
		return nil, x.Error
	}

	res, ok := x.mqlUsersInternal.usersByID[id]
	if !ok {
		return nil, errors.New("cannot find user for uid " + strconv.Itoa(int(id)))
	}
	return res, nil
}
