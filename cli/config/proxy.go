// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package config

import (
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/viper"
	"go.mondoo.com/ranger-rpc"
)

// GetAPIProxy returns the proxy URL from the environment variable MONDOO_API_PROXY, HTTPS_PROXY and cli flag --api-proxy.
// It should only be used when the options are not yet parsed, see CommonCliConfig.GetAPIProxy().
func GetAPIProxy() (*url.URL, error) {
	proxy, envSet := os.LookupEnv("MONDOO_API_PROXY")
	if envSet {
		return url.Parse(proxy)
	}

	proxy = viper.GetString("api_proxy")
	if proxy != "" {
		return url.Parse(proxy)
	}

	proxy, envSet = os.LookupEnv("HTTPS_PROXY")
	if envSet {
		return url.Parse(proxy)
	}

	return nil, nil
}

func (c *CommonOpts) GetAPIProxy() (*url.URL, error) {
	proxy, err := GetAPIProxy()
	if err != nil {
		return nil, err
	}
	if proxy != nil {
		return proxy, nil
	}

	// fallback to proxy from config file
	if c.APIProxy != "" {
		return url.Parse(c.APIProxy)
	}

	return nil, nil
}

func (c *CommonOpts) GetHttpClient() (*http.Client, error) {
	proxy, err := c.GetAPIProxy()
	if err != nil {
		return nil, err
	}
	if proxy == nil {
		return ranger.DefaultHttpClient(), nil
	}
	return ranger.NewHttpClient(ranger.WithProxy(proxy)), nil
}
