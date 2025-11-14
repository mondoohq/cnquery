// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
)

// createHTTPClientForInterface creates an HTTP client bound to a specific network interface
// by binding to the local IP address of that interface
func createHTTPClientForInterface(interfaceIP net.IP) (*http.Client, error) {
	// Find the interface that has this IP
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list interfaces")
	}

	var targetIP net.IP
	var ifaceName string
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && ip.Equal(interfaceIP) {
				// Found the interface, use this IP for binding
				targetIP = ip
				ifaceName = iface.Name
				break
			}
		}
		if targetIP != nil {
			break
		}
	}

	if targetIP == nil {
		return nil, errors.New("interface with IP not found")
	}

	// Create TCP address with the IP (port 0 means let OS choose)
	// For IPv6 link-local addresses, we need to use the zone identifier
	var localAddr *net.TCPAddr
	if targetIP.To4() != nil {
		localAddr = &net.TCPAddr{IP: targetIP.To4(), Port: 0}
	} else {
		// For IPv6, check if it's link-local and needs zone identifier
		if targetIP.IsLinkLocalUnicast() && ifaceName != "" {
			// Try to resolve with zone identifier
			// Format: fe80::1%en0
			zoneAddr := targetIP.String() + "%" + ifaceName
			ipAddr, err := net.ResolveIPAddr("ip6", zoneAddr)
			if err == nil {
				localAddr = &net.TCPAddr{IP: ipAddr.IP, Zone: ifaceName, Port: 0}
			} else {
				// Fallback to IP without zone (may not work for link-local)
				localAddr = &net.TCPAddr{IP: targetIP.To16(), Port: 0}
			}
		} else {
			localAddr = &net.TCPAddr{IP: targetIP.To16(), Port: 0}
		}
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

// callIpinfoAPI makes a direct HTTP call to ipinfo.io API
func callIpinfoAPI(client *http.Client, queryIP net.IP, token string) (*ipinfoResponse, error) {
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

	// Add token if provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	log.Debug().Msg("initIpinfo called")
	// token := os.Getenv("IPINFO_TOKEN")
	token := ""

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
			// For local IPs, try to bind the HTTP request to this interface
			// Skip loopback addresses as they can't reach the internet
			// For link-local IPv6, we'll use the zone identifier from the interface
			if !queryIP.IsLoopback() {
				var err error
				httpClient, err = createHTTPClientForInterface(queryIP)
				if err != nil {
					log.Debug().Err(err).Str("ip", queryIP.String()).Msg("failed to bind to interface, using default client")
					// Continue with default client if binding fails
					httpClient = &http.Client{
						Timeout: 30 * time.Second,
					}
				} else {
					log.Debug().Str("ip", queryIP.String()).Msg("bound HTTP client to interface")
				}
			}
			queryIP = nil // Query for YOUR public IP from this interface (or default connection)
		}
	} else {
		// No IP provided - get YOUR public IP (the one visible from internet)
		queryIP = nil
	}

	// Call ipinfo.io API directly
	queryIPStr := "nil (public IP)"
	if queryIP != nil {
		queryIPStr = queryIP.String()
	}
	tokenStatus := "empty (using free API)"
	if token != "" {
		tokenStatus = "set"
	}

	log.Debug().
		Str("queryIP", queryIPStr).
		Bool("isLocalIP", isLocalIP).
		Str("token", tokenStatus).
		Bool("usingInterfaceBinding", httpClient != nil && httpClient.Transport != nil).
		Msg("calling ipinfo.io API")

	info, err := callIpinfoAPI(httpClient, queryIP, token)
	if err != nil {
		// If binding failed and we got an error, it might be because the interface
		// can't reach the internet. For local IPs, we can still return the public IP
		// from the default connection as a fallback.
		if isLocalIP && httpClient != nil && httpClient.Transport != nil {
			// Try again with default client as fallback
			log.Debug().Err(err).Str("ip", queryIPStr).Msg("interface-bound request failed, retrying with default client")
			defaultClient := &http.Client{
				Timeout: 30 * time.Second,
			}
			info, err = callIpinfoAPI(defaultClient, queryIP, token)
		}

		if err != nil {
			log.Debug().Err(err).
				Str("queryIP", queryIPStr).
				Msg("ipinfo.io API call failed")
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
		Msg("ipinfo.io API response")

	// Set the IP field:
	// - If input was a local IP, return the public IP for that interface
	// - If input was a public IP, return that public IP
	// - If no input, return YOUR public IP
	if isLocalIP && info.IP != "" {
		// Local IP was provided, we queried for public IP from that interface
		// Return the public IP that ipinfo.io sees from that interface
		args["ip"] = llx.IPData(llx.ParseIP(info.IP))
	} else if inputIP != nil {
		// Public IP was provided, keep it
		args["ip"] = inputIP
	} else if info.IP != "" {
		// No input, use the public IP returned by ipinfo.io
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
