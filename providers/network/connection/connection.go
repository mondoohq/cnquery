// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package connection

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/plugin"
)

type HostConnection struct {
	plugin.Connection
	Conf       *inventory.Config
	asset      *inventory.Asset
	httpClient *http.Client
}

func NewHostConnection(id uint32, asset *inventory.Asset, conf *inventory.Config) *HostConnection {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if conf.Insecure {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &HostConnection{
		Connection: plugin.NewConnection(id, asset),
		Conf:       conf,
		asset:      asset,
		httpClient: &http.Client{Transport: transport},
	}
}

func (h *HostConnection) Name() string {
	return "host"
}

func (p *HostConnection) Asset() *inventory.Asset {
	return p.asset
}

func (p *HostConnection) FQDN() string {
	if p.Conf == nil {
		return ""
	}
	return p.Conf.Host
}

func (p *HostConnection) Client() *http.Client {
	return p.httpClient
}
