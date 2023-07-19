package resources

import (
	"errors"
	"path"
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

func (x *mqlUser) init(args map[string]interface{}) (map[string]interface{}, *mqlUser, error) {
	if len(args) != 1 {
		return args, nil, nil
	}

	// if user is only initialized with a name or uid, we will try to look it up
	rawName, nameOk := args["name"]
	rawUID, idOk := args["uid"]
	if !nameOk && !idOk {
		return args, nil, nil
	}

	raw, err := CreateResource(x.MqlRuntime, "users", nil)
	if err != nil {
		return nil, nil, errors.New("cannot get list of users: " + err.Error())
	}
	users := raw.(*mqlUsers)

	list := users.GetList()
	if list.Error != nil {
		return nil, nil, list.Error
	}

	var f func(user interface{}) bool
	if nameOk {
		name, ok := rawName.(string)
		if !ok {
			return nil, nil, errors.New("cannot detect user, invalid type for name (expected string)")
		}
		f = func(user interface{}) bool {
			return user.(*mqlUser).Name.Data == name
		}
	} else if idOk {
		id, ok := rawUID.(int64)
		if !ok {
			return nil, nil, errors.New("cannot detect user, invalid type for name (expected int)")
		}
		f = func(user interface{}) bool {
			return user.(*mqlUser).Uid.Data == id
		}
	}

	for i := range list.Data {
		if f(list.Data[i]) {
			return nil, list.Data[i].(*mqlUser), nil
		}
	}

	if nameOk {
		return nil, nil, errors.New("cannot find user with name '" + rawName.(string) + "'")
	} else {
		return nil, nil, errors.New("cannot find user with uid '" + strconv.FormatInt(rawUID.(int64), 10) + "'")
	}
}

func (x *mqlUser) group(gid int64) (*mqlGroup, error) {
	raw, err := CreateResource(x.MqlRuntime, "groups", nil)
	if err != nil {
		return nil, errors.New("cannot get groups info for user: " + err.Error())
	}
	groups := raw.(*mqlGroups)
	return groups.findID(gid)
}

func (u *mqlUser) authorizedkeys(home string) (*mqlAuthorizedkeys, error) {
	// TODO: we may need to handle ".ssh/authorized_keys2" too
	authorizedKeysPath := path.Join(home, ".ssh", "authorized_keys")
	ak, err := CreateResource(u.MqlRuntime, "authorizedkeys", map[string]interface{}{
		"path": authorizedKeysPath,
	})
	if err != nil {
		return nil, err
	}
	return ak.(*mqlAuthorizedkeys), nil
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
