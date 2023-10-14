// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"io"
	"net/http"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mondoo.com/cnquery/v9/llx"
	"go.mondoo.com/cnquery/v9/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v9/types"
)

type mqlHttpGetInternal struct {
	lock sync.Mutex
	resp plugin.TValue[*http.Response]
}

func initHttpGet(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if rawUrl, ok := args["rawUrl"]; ok {
		// We add a default prefix if it is missing. Otherwise the URL parser
		// will return unintuitive results. For example: "mondoo.com" would
		// return a host="" with path="mondoo.com", correctly following the spec.
		if !strings.Contains(rawUrl.Value.(string), "://") {
			rawUrl.Value = "http://" + rawUrl.Value.(string)
		}

		url, err := NewResource(runtime, "url", map[string]*llx.RawData{
			"raw": rawUrl,
			// Additional arguments for this resource are used only if parsing
			// of the raw url doesn't overwrite them. They are defaults.
			"scheme": llx.StringData("http"),
		})
		if err != nil {
			return nil, nil, err
		}

		delete(args, "rawUrl")
		args["url"] = llx.ResourceData(url, "url")
	}
	return args, nil, nil
}

func (x *mqlHttpGet) id() (string, error) {
	return x.Url.Data.__id, nil
}

func (x *mqlHttpGet) do() error {
	x.lock.Lock()
	defer x.lock.Unlock()

	if x.resp.State&plugin.StateIsSet != 0 {
		return x.resp.Error
	}

	resp, err := http.Get(x.Url.Data.String.Data)
	x.resp.State = plugin.StateIsSet
	x.resp.Data = resp
	x.resp.Error = err
	if err != nil {
		return err
	}

	x.StatusCode.Data = int64(resp.StatusCode)
	x.StatusCode.State = plugin.StateIsSet

	x.Version.Data = strconv.Itoa(resp.ProtoMajor) + "." + strconv.Itoa(resp.ProtoMinor)
	x.Version.State = plugin.StateIsSet
	return nil
}

func (x *mqlHttpGet) header() (*mqlHttpHeader, error) {
	if err := x.do(); err != nil {
		return nil, err
	}

	header := x.resp.Data.Header
	params := make(map[string]interface{}, len(header))
	for key := range header {
		mkey := textproto.CanonicalMIMEHeaderKey(key)
		vals := header[mkey]
		ivals := make([]interface{}, len(vals))
		for j := range vals {
			ivals[j] = vals[j]
		}
		params[mkey] = ivals
	}

	o, err := CreateResource(x.MqlRuntime, "http.header", map[string]*llx.RawData{
		"params": llx.MapData(params, types.Array(types.String)),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeader), nil
}

func (x *mqlHttpGet) statusCode() (int64, error) {
	return 0, x.do()
}

func (x *mqlHttpGet) version() (string, error) {
	return "", x.do()
}

func (x *mqlHttpGet) body() (string, error) {
	if err := x.do(); err != nil {
		return "", err
	}
	raw, err := io.ReadAll(x.resp.Data.Body)
	return string(raw), err
}

func (x *mqlHttpHeader) sts() (*mqlHttpHeaderSts, error) {
	params, ok := x.Params.Data["Strict-Transport-Security"]
	if !ok {
		x.Sts.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	preload := false
	includeSubDomains := false
	maxAge := llx.NilData

	sts := params.([]interface{})
	for i := range sts {
		h := sts[i].(string)
		fields := strings.Split(h, ";")
		for j := range fields {
			field := strings.TrimSpace(fields[j])
			switch field {
			case "preload":
				preload = true
			case "includeSubDomains":
				includeSubDomains = true
			default:
				s := strings.SplitN(field, "=", 2)
				// only max-age supported at the time of writing
				if s[0] != "max-age" {
					continue
				}

				if len(s) != 2 {
					maxAge = llx.TimeData(time.Time{})
					maxAge.Error = errors.New("maxAge is invalid: " + field)
				} else {
					age, _ := strconv.Atoi(s[1])
					maxAge = llx.TimeData(llx.DurationToTime(int64(age)))
				}
			}
		}
	}

	o, err := CreateResource(x.MqlRuntime, "http.header.sts", map[string]*llx.RawData{
		"preload":           llx.BoolData(preload),
		"includeSubDomains": llx.BoolData(includeSubDomains),
		"maxAge":            maxAge,
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeaderSts), nil
}
