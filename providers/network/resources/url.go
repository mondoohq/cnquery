// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	"go.mondoo.com/cnquery/v11/llx"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

func initUrl(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if raws, ok := args["raw"]; ok {
		s := raws.Value.(string)
		delete(args, "raw")

		u, err := url.Parse(s)
		if err != nil {
			return nil, nil, errors.New("failed to parse url: " + err.Error())
		}

		if u.Scheme != "" || args["scheme"] == nil {
			args["scheme"] = llx.StringData(u.Scheme)
		}

		name := u.User.Username()
		if name != "" || args["user"] == nil {
			args["user"] = llx.StringData(name)
		}

		pass, _ := u.User.Password()
		if pass != "" || args["pass"] == nil {
			args["password"] = llx.StringData(pass)
		}

		host := strings.SplitN(u.Host, ":", 2)
		if host[0] != "" || args["host"] == nil {
			args["host"] = llx.StringData(host[0])
		}

		var port int
		if len(host) != 1 {
			port, err = strconv.Atoi(host[1])
			if err != nil {
				return nil, nil, errors.New("invalid port for url: " + s)
			}
		}
		if port != 0 || args["port"] == nil {
			args["port"] = llx.IntData(int64(port))
		}

		if u.Path != "" || args["path"] == nil {
			args["path"] = llx.StringData(u.Path)
		}

		if u.RawQuery != "" || args["rawQuery"] == nil {
			args["rawQuery"] = llx.StringData(u.RawQuery)
		}

		if u.RawFragment != "" || args["rawFragment"] == nil {
			args["rawFragment"] = llx.StringData(u.RawFragment)
		}
	}
	return args, nil, nil
}

func (x *mqlUrl) id() (string, error) {
	s := x.GetString()
	return s.Data, s.Error
}

func (x *mqlUrl) string() (string, error) {
	var user *url.Userinfo
	if x.Password.Data != "" {
		user = url.UserPassword(x.User.Data, x.Password.Data)
	} else if x.User.Data != "" {
		user = url.User(x.User.Data)
	}

	host := x.Host.Data
	isStandardPort := x.Port.Data == 80 && x.Scheme.Data == "http" || x.Port.Data == 443 && x.Scheme.Data == "https"
	if x.Port.Data != 0 && !isStandardPort {
		host += ":" + strconv.Itoa(int(x.Port.Data))
	}

	u := url.URL{
		Scheme:      x.Scheme.Data,
		User:        user,
		Host:        host,
		Path:        x.Path.Data,
		RawQuery:    x.RawQuery.Data,
		RawFragment: x.RawFragment.Data,
	}
	return u.String(), nil
}
