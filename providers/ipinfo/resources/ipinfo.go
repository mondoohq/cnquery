// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v12/providers/os/id/networki"
)

// getOSConnection tries to get the OS provider connection from the runtime
// Returns nil if OS provider is not available
// Since OS is a cross-provider, the runtime should have access to it when needed
func getOSConnection(runtime *plugin.Runtime) (shared.Connection, *inventory.Platform) {
	// Check if the current connection is an OS connection
	// This works when ipinfo is used as a cross-provider from an OS provider context
	if osConn, ok := runtime.Connection.(shared.Connection); ok {
		// shared.Connection has an Asset() method
		asset := osConn.Asset()
		if asset != nil {
			return osConn, asset.Platform
		}
	}

	// If not, we can't access other providers' connections directly
	// The runtime's cross-provider mechanism handles OS provider access
	// when OS resources are queried, but we don't have direct access to the connection here
	// We'll fall back to direct IP binding

	return nil, nil
}

// findInterfaceForIP finds the network interface that has the given IP address
// Uses OS provider's networki.Interfaces() if available, otherwise returns nil
func findInterfaceForIP(runtime *plugin.Runtime, targetIP net.IP) *networki.Interface {
	osConn, platform := getOSConnection(runtime)
	if osConn == nil || platform == nil {
		log.Debug().Str("ip", targetIP.String()).Msg("OS provider connection not available, cannot find interface")
		return nil
	}

	interfaces, err := networki.Interfaces(osConn, platform)
	if err != nil {
		log.Debug().Err(err).Str("ip", targetIP.String()).Msg("failed to get network interfaces from OS provider")
		return nil
	}

	// Find the interface that has this IP
	for _, iface := range interfaces {
		for _, ipAddr := range iface.IPAddresses {
			if ipAddr.IP != nil && ipAddr.IP.Equal(targetIP) {
				return &iface
			}
		}
	}

	return nil
}

// createHTTPClientForIP creates an HTTP client bound to a specific local IP address
// Tries to use OS provider's network interfaces if available, otherwise binds directly to IP
func createHTTPClientForIP(runtime *plugin.Runtime, interfaceIP net.IP) (*http.Client, error) {
	var localAddr *net.TCPAddr
	var ifaceName string

	// Try to find the interface using OS provider's network interfaces
	if iface := findInterfaceForIP(runtime, interfaceIP); iface != nil {
		ifaceName = iface.Name
		log.Debug().Str("ip", interfaceIP.String()).Str("interface", ifaceName).Msg("found interface using OS provider")
	}

	// Create TCP address with the IP (port 0 means let OS choose)
	if interfaceIP.To4() != nil {
		localAddr = &net.TCPAddr{IP: interfaceIP.To4(), Port: 0}
	} else {
		// For IPv6 link-local addresses, we need the zone identifier
		if interfaceIP.IsLinkLocalUnicast() && ifaceName != "" {
			// Try to resolve with zone identifier
			// Format: fe80::1%en0
			zoneAddr := interfaceIP.String() + "%" + ifaceName
			ipAddr, err := net.ResolveIPAddr("ip6", zoneAddr)
			if err == nil {
				localAddr = &net.TCPAddr{IP: ipAddr.IP, Zone: ifaceName, Port: 0}
			} else {
				// Fallback to IP without zone (may not work for link-local)
				log.Debug().Err(err).Str("ip", interfaceIP.String()).Msg("failed to resolve IPv6 with zone, using without zone")
				localAddr = &net.TCPAddr{IP: interfaceIP.To16(), Port: 0}
			}
		} else {
			localAddr = &net.TCPAddr{IP: interfaceIP.To16(), Port: 0}
		}
	}

	// Create a custom dialer that binds to the specific IP
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

// ipinfoResponse represents the JSON response from ipinfo.io API
type ipinfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

// isBogonIP checks if an IP is a bogon (private, loopback, link-local, etc.)
func isBogonIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() || ip.IsMulticast() || ip.IsUnspecified()
}

// queryIPWithSDK queries IP information using the ipinfo Go SDK
// The SDK automatically handles bogon detection and returns appropriate responses
func queryIPWithSDK(runtime *plugin.Runtime, token string, queryIP net.IP, interfaceIP net.IP) (*ipinfoResponse, error) {
	// Create HTTP client bound to interface if we have a local IP
	var httpClient *http.Client
	var err error
	if interfaceIP != nil && !interfaceIP.IsLoopback() {
		httpClient, err = createHTTPClientForIP(runtime, interfaceIP)
		if err != nil {
			log.Debug().Err(err).Str("ip", interfaceIP.String()).Msg("failed to create interface-bound client, using default")
			httpClient = nil
		}
	}

	// Create SDK client with custom HTTP client if needed
	sdkClient := ipinfo.NewClient(httpClient, nil, token)

	// Query the IP
	var info *ipinfo.Core
	if queryIP == nil {
		// Query for YOUR public IP
		info, err = sdkClient.GetIPInfo(nil)
	} else {
		info, err = sdkClient.GetIPInfo(queryIP)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to query ipinfo SDK")
	}

	// Convert SDK response to our response format
	response := &ipinfoResponse{
		IP:       info.IP.String(),
		Hostname: info.Hostname,
		City:     info.City,
		Region:   info.Region,
		Country:  info.Country,
		Loc:      info.Location,
		Org:      info.Org,
		Postal:   info.Postal,
		Timezone: info.Timezone,
	}

	return response, nil
}

// queryIPWithFreeAPI queries IP information using the free ipinfo.io API
// This is the deprecated free API that doesn't require authentication
func queryIPWithFreeAPI(client *http.Client, queryIP net.IP) (*ipinfoResponse, error) {
	var url string
	if queryIP == nil {
		// Query for YOUR public IP
		url = "https://ipinfo.io"
	} else {
		// Query for specific IP
		url = fmt.Sprintf("https://ipinfo.io/%s", queryIP.String())
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create HTTP request")
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to make HTTP request to ipinfo.io")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.Errorf("ipinfo.io API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var info ipinfoResponse
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, errors.Wrap(err, "failed to parse ipinfo.io response")
	}

	return &info, nil
}

// queryIPInfo is a wrapper function that chooses between SDK and free API based on token availability
func queryIPInfo(runtime *plugin.Runtime, queryIP net.IP, interfaceIP net.IP, token string) (*ipinfoResponse, error) {
	// Check if token is available
	if token == "" {
		// Use free API
		var httpClient *http.Client
		var err error

		// Create HTTP client bound to interface if we have a local IP
		if interfaceIP != nil && !interfaceIP.IsLoopback() {
			httpClient, err = createHTTPClientForIP(runtime, interfaceIP)
			if err != nil {
				log.Debug().Err(err).Str("ip", interfaceIP.String()).Msg("failed to create interface-bound client, using default")
				httpClient = &http.Client{
					Timeout: 30 * time.Second,
				}
			}
		} else {
			httpClient = &http.Client{
				Timeout: 30 * time.Second,
			}
		}

		return queryIPWithFreeAPI(httpClient, queryIP)
	}

	// Use SDK
	return queryIPWithSDK(runtime, token, queryIP, interfaceIP)
}

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	log.Debug().Msg("initIpinfo called")

	// Get token from environment variable
	token := os.Getenv("IPINFO_TOKEN")

	var queryIP net.IP
	var inputIP *llx.RawData
	var interfaceIP net.IP
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

		// Check if this is a bogon IP (local/private/loopback/link-local)
		if isBogonIP(queryIP) {
			isLocalIP = true
			interfaceIP = queryIP
			// For local IPs, query for the public IP from this interface
			// The SDK or free API will handle this appropriately
			queryIP = nil
		}
	} else {
		// No IP provided - get YOUR public IP (the one visible from internet)
		queryIP = nil
	}

	// Determine which API to use and log
	queryIPStr := "nil (public IP)"
	if queryIP != nil {
		queryIPStr = queryIP.String()
	}
	tokenStatus := "empty (using free API)"
	if token != "" {
		tokenStatus = "set (using SDK)"
	}

	interfaceIPStr := "none"
	if interfaceIP != nil {
		interfaceIPStr = interfaceIP.String()
	}

	log.Debug().
		Str("queryIP", queryIPStr).
		Bool("isLocalIP", isLocalIP).
		Str("interfaceIP", interfaceIPStr).
		Str("token", tokenStatus).
		Msg("querying ipinfo")

	// Query IP information using the appropriate method
	info, err := queryIPInfo(runtime, queryIP, interfaceIP, token)
	if err != nil {
		// If interface binding failed and we got an error, try again with default client as fallback
		if isLocalIP && interfaceIP != nil && !interfaceIP.IsLoopback() {
			log.Debug().Err(err).Str("ip", queryIPStr).Msg("interface-bound request failed, retrying with default client")
			info, err = queryIPInfo(runtime, queryIP, nil, token)
		}

		if err != nil {
			log.Debug().Err(err).
				Str("queryIP", queryIPStr).
				Msg("ipinfo query failed")
			// Return nil values instead of error to allow query to continue
			// The resource will show "no data available" but won't crash
			return nil, nil, nil
		}
	}

	if info == nil {
		// Should not happen, but handle gracefully
		return nil, nil, nil
	}

	log.Debug().
		Str("response_ip", info.IP).
		Str("response_hostname", info.Hostname).
		Interface("full_response", info).
		Msg("ipinfo response")

	// Set the IP field:
	// - If input was a local IP, return the public IP for that interface
	// - If input was a public IP, return that public IP
	// - If no input, return YOUR public IP
	if isLocalIP && info.IP != "" {
		// Local IP was provided, we queried for public IP from that interface
		// Return the public IP that ipinfo sees from that interface
		args["ip"] = llx.IPData(llx.ParseIP(info.IP))
	} else if inputIP != nil {
		// Public IP was provided, keep it
		args["ip"] = inputIP
	} else if info.IP != "" {
		// No input, use the public IP returned by ipinfo
		args["ip"] = llx.IPData(llx.ParseIP(info.IP))
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
