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

	"go.mondoo.com/cnquery/v10/checksums"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers/network/connection"
	"go.mondoo.com/cnquery/v10/types"
	"go.mondoo.com/cnquery/v10/utils/sortx"
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

	if _, ok := args["url"]; !ok {
		conn := runtime.Connection.(*connection.HostConnection)
		if conn.Conf == nil {
			return nil, nil, errors.New("missing URL for http.get")
		}

		scheme := conn.Conf.Runtime
		if scheme == "" {
			// At this point we are in best effort territory. Which means we will
			// go HTTP unless port 443 is specified. Users can always provide a
			// scheme to be prescriptive.
			if conn.Conf.Port == 443 {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		url, err := NewResource(runtime, "url", map[string]*llx.RawData{
			"host":   llx.StringData(conn.Conf.Host),
			"port":   llx.IntData(int64(conn.Conf.Port)),
			"scheme": llx.StringData(scheme),
		})
		if err != nil {
			return nil, nil, err
		}
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

	if x.Url.Data == nil {
		return errors.New("missing URL for http.get")
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
		params[normalizeHeaderKey(mkey)] = ivals
	}

	o, err := CreateResource(x.MqlRuntime, "http.header", map[string]*llx.RawData{
		"__id":   llx.StringData(x.__id),
		"params": llx.MapData(params, types.Array(types.String)),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeader), nil
}

var normHeaderKeys = map[string]string{
	"X-Xss-Protection": "X-XSS-Protection",
}

// adds a few more key normalizations
func normalizeHeaderKey(key string) string {
	if res, ok := normHeaderKeys[key]; ok {
		return res
	}
	return key
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

func parseHeaderFields(raw []interface{}, f func(key string, value string)) {
	parseHeaderFieldsD(raw, f, ";", "=")
}

func parseHeaderFieldsD(raw []interface{}, f func(key string, value string), delimFields string, delimKV string) {
	for i := range raw {
		h := raw[i].(string)
		fields := strings.Split(h, delimFields)
		for j := range fields {
			field := strings.TrimSpace(fields[j])
			s := strings.SplitN(field, delimKV, 2)
			if len(s) == 1 {
				f(s[0], "")
			} else {
				f(s[0], s[1])
			}
		}
	}
}

func parseSingleHeaderValue[T any](raw interface{}, found bool, field *plugin.TValue[T]) (string, error) {
	if !found {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}

	arr := raw.([]interface{})
	var res strings.Builder
	for i := range arr {
		if i != 0 {
			res.WriteByte(' ')
		}
		res.WriteString(arr[i].(string))
	}
	return res.String(), nil
}

func (x *mqlHttpHeader) id() (string, error) {
	return "", errors.New("http header not initialized")
}

func (x *mqlHttpHeader) sts() (*mqlHttpHeaderSts, error) {
	raw, ok := x.Params.Data["Strict-Transport-Security"]
	if !ok {
		x.Sts.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	preload := false
	includeSubDomains := false
	maxAge := llx.NilData

	parseHeaderFields(raw.([]interface{}), func(key string, value string) {
		switch key {
		case "preload":
			preload = true
		case "includeSubDomains":
			includeSubDomains = true
		case "max-age":
			age, err := strconv.Atoi(value)
			if err != nil {
				maxAge = llx.TimeData(time.Time{})
				maxAge.Error = errors.New("maxAge is invalid: " + value)
			} else {
				maxAge = llx.TimeData(llx.DurationToTime(int64(age)))
			}
		}
	})

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

func (x *mqlHttpHeaderSts) id() (string, error) {
	id := ""
	if x.MaxAge.Data != nil {
		id += "maxAge=" + strconv.Itoa(int(x.MaxAge.Data.Unix()))
	}
	if x.Preload.Data {
		id += ";preload"
	}
	if x.IncludeSubDomains.Data {
		id += ";includeSubdomains"
	}
	return id, nil
}

func (x *mqlHttpHeader) xFrameOptions() (string, error) {
	params, ok := x.Params.Data["X-Frame-Options"]
	return parseSingleHeaderValue(params, ok, &x.XFrameOptions)
}

func (x *mqlHttpHeader) xXssProtection() (*mqlHttpHeaderXssProtection, error) {
	raw, ok := x.Params.Data["X-XSS-Protection"]
	if !ok {
		x.XXssProtection.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	enabled := llx.NilData
	mode := llx.NilData
	report := llx.NilData
	parseHeaderFields(raw.([]interface{}), func(key string, value string) {
		switch key {
		case "0":
			enabled = llx.BoolFalse
		case "1":
			enabled = llx.BoolTrue
		case "mode":
			mode = llx.StringData(value)
		case "max-age":
			report = llx.StringData(value)
		}
	})

	o, err := CreateResource(x.MqlRuntime, "http.header.xssProtection", map[string]*llx.RawData{
		"enabled": enabled,
		"mode":    mode,
		"report":  report,
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeaderXssProtection), nil
}

func (x *mqlHttpHeaderXssProtection) id() (string, error) {
	var id string
	if x.Enabled.Data {
		id = "1"
	} else {
		id = "0"
	}
	if x.Mode.Data != "" {
		id += ";mode=" + x.Mode.Data
	}
	if x.Report.Data != "" {
		id += ";report=" + x.Report.Data
	}
	return id, nil
}

func (x *mqlHttpHeader) xContentTypeOptions() (string, error) {
	params, ok := x.Params.Data["X-Content-Type-Options"]
	return parseSingleHeaderValue(params, ok, &x.XContentTypeOptions)
}

func (x *mqlHttpHeader) referrerPolicy() (string, error) {
	params, ok := x.Params.Data["Referrer-Policy"]
	return parseSingleHeaderValue(params, ok, &x.ReferrerPolicy)
}

func (x *mqlHttpHeader) contentType() (*mqlHttpHeaderContentType, error) {
	raw, ok := x.Params.Data["Content-Type"]
	if !ok {
		x.ContentType.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	typ := llx.NilData
	params := llx.NilData
	parseHeaderFields(raw.([]interface{}), func(key string, value string) {
		if typ.Value == nil && value == "" {
			typ = llx.StringData(key)
			return
		}
		if params.Value == nil {
			params = llx.MapData(map[string]interface{}{}, types.String)
		}
		params.Value.(map[string]interface{})[key] = value
	})

	o, err := CreateResource(x.MqlRuntime, "http.header.contentType", map[string]*llx.RawData{
		"type":   typ,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeaderContentType), nil
}

func (x *mqlHttpHeaderContentType) id() (string, error) {
	id := x.Type.Data
	if x.Params.Data != nil {
		keys := sortx.Keys(x.Params.Data)
		for _, key := range keys {
			id += ";" + key + "=" + x.Params.Data[key].(string)
		}
	}
	return id, nil
}

func (x *mqlHttpHeader) setCookie() (*mqlHttpHeaderSetCookie, error) {
	raw, ok := x.Params.Data["Set-Cookie"]
	if !ok {
		x.SetCookie.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	cname := llx.NilData
	cval := llx.NilData
	params := llx.NilData
	parseHeaderFields(raw.([]interface{}), func(key string, value string) {
		if cname.Value == nil && value != "" {
			cname = llx.StringData(key)
			cval = llx.StringData(value)
			return
		}
		if params.Value == nil {
			params = llx.MapData(map[string]interface{}{}, types.String)
		}
		params.Value.(map[string]interface{})[key] = value
	})

	o, err := CreateResource(x.MqlRuntime, "http.header.setCookie", map[string]*llx.RawData{
		"name":   cname,
		"value":  cval,
		"params": params,
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlHttpHeaderSetCookie), nil
}

func (x *mqlHttpHeader) csp() (map[string]interface{}, error) {
	raw, ok := x.Params.Data["Content-Security-Policy"]
	if !ok {
		x.Csp.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	m := map[string]interface{}{}
	parseHeaderFieldsD(raw.([]interface{}), func(key string, value string) {
		m[key] = value
	}, ";", " ")

	return m, nil
}

func (x *mqlHttpHeaderSetCookie) id() (string, error) {
	// cookies may be very long, so it's more efficient to checksum them
	res := checksums.New.Add(x.Name.Data).Add(x.Value.Data)
	if x.Params.Data != nil {
		keys := sortx.Keys(x.Params.Data)
		for _, key := range keys {
			res = res.
				Add(key).
				Add(x.Params.Data[key].(string))
		}
	}
	return res.String(), nil
}
