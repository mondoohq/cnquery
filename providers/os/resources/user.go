// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/spf13/afero"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/users"
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

func initUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) != 1 {
		return args, nil, nil
	}

	// if user is only initialized with a name or uid, we will try to look it up
	rawName, nameOk := args["name"]
	rawUID, idOk := args["uid"]
	if !nameOk && !idOk {
		return args, nil, errors.New("cannot find user, no search criteria provided")
	}

	raw, err := CreateResource(runtime, "users", nil)
	if err != nil {
		return nil, nil, errors.New("cannot get list of users: " + err.Error())
	}
	users := raw.(*mqlUsers)
	usersList := users.GetList()
	if usersList.Error != nil {
		return nil, nil, usersList.Error
	}

	if nameOk {
		name, ok := rawName.Value.(string)
		if !ok {
			return nil, nil, errors.New("cannot detect user, invalid type for name (expected string)")
		}
		user, err := users.findName(name)
		if err != nil {
			return nil, nil, errors.New("cannot find user with name '" + name + "'")
		}
		return nil, user, nil
	}

	if idOk {
		id, ok := rawUID.Value.(int64)
		if !ok {
			return nil, nil, errors.New("cannot detect user, invalid type for name (expected int)")
		}
		user, err := users.findID(id)
		if err != nil {
			return nil, nil, err
		}
		return nil, user, nil
	}

	return nil, nil, errors.New("cannot find user, no search criteria provided")
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
	ak, err := NewResource(u.MqlRuntime, "authorizedkeys", map[string]*llx.RawData{
		"path": llx.StringData(authorizedKeysPath),
	})
	if err != nil {
		return nil, err
	}
	return ak.(*mqlAuthorizedkeys), nil
}

func (x *mqlUsers) list() ([]interface{}, error) {
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
		nu, err := CreateResource(x.MqlRuntime, "user", map[string]*llx.RawData{
			"name":    llx.StringData(user.Name),
			"uid":     llx.IntData(user.Uid),
			"gid":     llx.IntData(user.Gid),
			"sid":     llx.StringData(user.Sid),
			"home":    llx.StringData(user.Home),
			"shell":   llx.StringData(user.Shell),
			"enabled": llx.BoolData(user.Enabled),
		})
		if err != nil {
			return nil, err
		}

		res = append(res, nu)
	}

	return res, nil
}

func (x *mqlUsers) findID(uid int64) (*mqlUser, error) {
	list := x.GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	for _, u := range list.Data {
		user := u.(*mqlUser)
		if user.Uid.Data == uid {
			return user, nil
		}
	}
	return nil, errors.New("cannot find user with UID " + strconv.Itoa(int(uid)))
}

func (x *mqlUsers) findName(name string) (*mqlUser, error) {
	list := x.GetList()
	if list.Error != nil {
		return nil, list.Error
	}

	for _, u := range list.Data {
		user := u.(*mqlUser)
		if user.Name.Data == name {
			return user, nil
		}
	}
	return nil, errors.New("cannot find user with name " + name)
}

func (u *mqlUser) sshkeys() ([]interface{}, error) {
	res := []interface{}{}

	userSshPath := path.Join(u.Home.Data, ".ssh")

	conn := u.MqlRuntime.Connection.(shared.Connection)
	afutil := afero.Afero{Fs: conn.FileSystem()}

	// check if use ssh directory exists
	exists, err := afutil.Exists(userSshPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		return res, nil
	}

	filter := []string{"config"}

	// walk dir and search for all private keys
	potentialPrivateKeyFiles := []string{}
	err = afutil.Walk(userSshPath, func(path string, f os.FileInfo, err error) error {
		if f == nil || f.IsDir() {
			return nil
		}

		// eg. matches google_compute_known_hosts and known_hosts
		if strings.HasSuffix(f.Name(), ".pub") || strings.HasSuffix(f.Name(), "known_hosts") {
			return nil
		}

		for i := range filter {
			if f.Name() == filter[i] {
				return nil
			}
		}

		potentialPrivateKeyFiles = append(potentialPrivateKeyFiles, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// iterate over files and check if the content is there
	for i := range potentialPrivateKeyFiles {
		path := potentialPrivateKeyFiles[i]
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		content := string(data)

		// check if content contains PRIVATE KEY
		isPrivateKey := strings.Contains(content, "PRIVATE KEY")
		// check if the key is encrypted ENCRYPTED
		isEncrypted := strings.Contains(content, "ENCRYPTED")

		if isPrivateKey {
			upk, err := CreateResource(u.MqlRuntime, "privatekey", map[string]*llx.RawData{
				"pem":       llx.StringData(content),
				"encrypted": llx.BoolData(isEncrypted),
				"path":      llx.StringData(path),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, upk.(*mqlPrivatekey))
		}
	}

	return res, nil
}
