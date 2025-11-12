// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"net"
	"os"

	"github.com/ipinfo/go/v2/ipinfo"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ipinfo/connection"
)

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Get ipinfo client from connection
	conn := runtime.Connection.(*connection.IpinfoConnection)
	client := conn.Client()
	if client == nil {
		// Initialize client if not already set
		token := os.Getenv("IPINFO_TOKEN")
		ipinfoClient := ipinfo.NewClient(nil, nil, token)
		conn.SetClient(ipinfoClient)
		client = ipinfoClient
	}

	ipinfoClient, ok := client.(*ipinfo.Client)
	if !ok {
		return nil, nil, errors.New("failed to get ipinfo client")
	}

	var queryIP net.IP
	var inputIP *llx.RawData
	var isLocalIP bool

	// Check if an IP was provided as input
	if ip, ok := args["ip"]; ok {
		ipVal, ok := ip.Value.(llx.RawIP)
		if !ok {
			return nil, nil, errors.New("ip must be of type ip")
		}
		if ipVal.IP == nil {
			return nil, nil, errors.New("ip cannot be empty")
		}
		inputIP = ip
		queryIP = ipVal.IP

		// Check if this is a local/private IP (bogon)
		// Local IPs won't have meaningful data from ipinfo.io
		// So we'll query for YOUR public IP instead
		if queryIP.IsLoopback() || queryIP.IsLinkLocalUnicast() || queryIP.IsPrivate() {
			isLocalIP = true
			queryIP = nil // Query for YOUR public IP instead
		}
	} else {
		// No IP provided - get YOUR public IP (the one visible from internet)
		// According to https://github.com/ipinfo/go, GetIPInfo(nil) returns your public IP
		queryIP = nil
	}

	// Call ipinfo.io API
	// If queryIP is nil, it returns YOUR public IP
	// If queryIP is provided, it returns info about that specific IP
	info, err := ipinfoClient.GetIPInfo(queryIP)
	if err != nil {
		return nil, nil, err
	}

	// Set the IP field:
	// - If input was a local IP, return YOUR public IP (the one visible from internet)
	// - If input was a public IP, return that public IP
	// - If no input, return YOUR public IP
	if isLocalIP && info.IP != nil {
		// Local IP was provided, but we queried for public IP
		// Return the public IP that ipinfo.io sees
		args["ip"] = llx.IPData(llx.ParseIP(info.IP.String()))
	} else if inputIP != nil {
		// Public IP was provided, keep it
		args["ip"] = inputIP
	} else if info.IP != nil {
		// No input, use the public IP returned by ipinfo.io
		args["ip"] = llx.IPData(llx.ParseIP(info.IP.String()))
	}

	// Set the hostname from API response
	// For local/private IPs, this will be empty (bogon IPs)
	args["hostname"] = llx.StringData(info.Hostname)

	return args, nil, nil
}

func (c *mqlIpinfo) id() (string, error) {
	if c.Ip.Error != nil {
		return "", c.Ip.Error
	}
	return "ipinfo\x00" + c.Ip.Data.String(), nil
}
