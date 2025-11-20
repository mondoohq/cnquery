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
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
)

// createHTTPClientForIP creates an HTTP client bound to a specific local IP address
// Binds directly to the provided IP without looking up interfaces
func createHTTPClientForIP(runtime *plugin.Runtime, interfaceIP net.IP) (*http.Client, error) {
	var localAddr *net.TCPAddr

	// Create TCP address with the IP (port 0 means let OS choose)
	if interfaceIP.To4() != nil {
		// IPv4 - bind directly
		localAddr = &net.TCPAddr{IP: interfaceIP.To4(), Port: 0}
	} else {
		// IPv6 - bind directly
		// Note: For link-local addresses, the zone identifier should already be in the IP
		// if it was provided that way. Otherwise, binding may fail for link-local addresses.
		localAddr = &net.TCPAddr{IP: interfaceIP.To16(), Port: 0}
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
	log.Debug().Str("args", fmt.Sprintf("%+v", args)).Msg("initIpinfo called")

	// Get token from environment variable
	token := os.Getenv("IPINFO_TOKEN")

	var queryIP net.IP
	var interfaceIP net.IP

	// Check if an IP was provided as input
	if ip, ok := args["ip"]; ok {
		ipVal, ok := ip.Value.(llx.RawIP)
		if !ok {
			return nil, nil, errors.New("ip must be of type ip")
		}
		if ipVal.IP == nil {
			return nil, nil, errors.New("ip cannot be empty")
		}

		if isBogonIP(ipVal.IP) {
			interfaceIP = ipVal.IP
			queryIP = nil
		} else {
			queryIP = ipVal.IP
			interfaceIP = nil
		}
	}

	log.Debug().
		Str("queryIP", queryIP.String()).
		Str("interfaceIP", interfaceIP.String()).
		Str("withSDK", fmt.Sprintf("%t", token != "")).
		Msg("querying ipinfo")

	// Query IP information using the appropriate method
	info, err := queryIPInfo(runtime, queryIP, interfaceIP, token)
	if err != nil {
		log.Debug().Err(err).
			Str("queryIP", queryIP.String()).
			Str("interfaceIP", interfaceIP.String()).
			Msg("ipinfo query failed")
		return nil, nil, err
	}

	if info == nil {
		return nil, nil, errors.New("ipinfo query returned no data")
	}

	log.Debug().
		Str("response_ip", info.IP).
		Str("response_hostname", info.Hostname).
		Interface("full_response", info).
		Msg("ipinfo response")

	args["ip"] = llx.IPData(llx.ParseIP(info.IP))
	args["hostname"] = llx.StringData(info.Hostname)

	return args, nil, nil
}

func (c *mqlIpinfo) id() (string, error) {
	if c.Ip.Error != nil {
		return "", c.Ip.Error
	}
	return "ipinfo\x00" + c.Ip.Data.String(), nil
}
