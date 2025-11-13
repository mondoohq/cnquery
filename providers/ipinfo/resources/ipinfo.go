// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ipinfo/go/v2/ipinfo"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ipinfo/connection"
)

// createHTTPClientForInterface creates an HTTP client bound to a specific network interface
// by binding to the local IP address of that interface
func createHTTPClientForInterface(interfaceIP net.IP) (*http.Client, error) {
	// Resolve the local TCP address with dynamic port (0)
	var localAddr net.Addr
	var err error
	
	if interfaceIP.To4() != nil {
		// IPv4
		localAddr, err = net.ResolveTCPAddr("tcp4", interfaceIP.String()+":0")
	} else {
		// IPv6
		localAddr, err = net.ResolveTCPAddr("tcp6", "["+interfaceIP.String()+"]:0")
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve local address")
	}

	// Create a custom dialer that binds to the specific interface
	dialer := &net.Dialer{
		LocalAddr: localAddr,
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Create a transport with the custom dialer
	transport := &http.Transport{
		DialContext:       dialer.DialContext,
		DisableKeepAlives: false,
	}

	// Create an HTTP client with the custom transport
	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return client, nil
}

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	token := os.Getenv("IPINFO_TOKEN")
	
	var queryIP net.IP
	var inputIP *llx.RawData
	var isLocalIP bool
	var httpClient *http.Client

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
		if queryIP.IsLoopback() || queryIP.IsLinkLocalUnicast() || queryIP.IsPrivate() {
			isLocalIP = true
			// For local IPs, bind the HTTP request to this interface
			// This allows us to get the public IP for this specific interface
			var err error
			httpClient, err = createHTTPClientForInterface(queryIP)
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to create HTTP client for interface")
			}
			queryIP = nil // Query for YOUR public IP from this interface
		} else {
			// Public IP provided - query info about that specific IP
			// No need to bind to interface for public IP queries
		}
	} else {
		// No IP provided - get YOUR public IP (the one visible from internet)
		// According to https://github.com/ipinfo/go, GetIPInfo(nil) returns your public IP
		queryIP = nil
	}

	// Create ipinfo client with custom HTTP client if we bound to an interface
	// Otherwise use default client from connection or create new one
	var ipinfoClient *ipinfo.Client
	conn := runtime.Connection.(*connection.IpinfoConnection)
	
	if httpClient != nil {
		// Use the interface-bound HTTP client
		ipinfoClient = ipinfo.NewClient(httpClient, nil, token)
	} else {
		// Use default client (from connection or create new)
		client := conn.Client()
		if client == nil {
			ipinfoClient = ipinfo.NewClient(nil, nil, token)
			conn.SetClient(ipinfoClient)
		} else {
			var ok bool
			ipinfoClient, ok = client.(*ipinfo.Client)
			if !ok {
				return nil, nil, errors.New("failed to get ipinfo client")
			}
		}
	}

	// Call ipinfo.io API
	// If queryIP is nil, it returns YOUR public IP (from the bound interface if specified)
	// If queryIP is provided, it returns info about that specific IP
	info, err := ipinfoClient.GetIPInfo(queryIP)
	if err != nil {
		return nil, nil, err
	}

	// Set the IP field:
	// - If input was a local IP, return the public IP for that interface
	// - If input was a public IP, return that public IP
	// - If no input, return YOUR public IP
	if isLocalIP && info.IP != nil {
		// Local IP was provided, we queried for public IP from that interface
		// Return the public IP that ipinfo.io sees from that interface
		args["ip"] = llx.IPData(llx.ParseIP(info.IP.String()))
	} else if inputIP != nil {
		// Public IP was provided, keep it
		args["ip"] = inputIP
	} else if info.IP != nil {
		// No input, use the public IP returned by ipinfo.io
		args["ip"] = llx.IPData(llx.ParseIP(info.IP.String()))
	}

	// Set the hostname from API response
	// For local/private IPs, this will be the hostname for the public IP of that interface
	args["hostname"] = llx.StringData(info.Hostname)

	return args, nil, nil
}

func (c *mqlIpinfo) id() (string, error) {
	if c.Ip.Error != nil {
		return "", c.Ip.Error
	}
	return "ipinfo\x00" + c.Ip.Data.String(), nil
}
