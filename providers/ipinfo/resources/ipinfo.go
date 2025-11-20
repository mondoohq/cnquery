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
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
)

var (
	ipInfoToken     string
	ipInfoTokenOnce sync.Once
)

// getIPInfoToken returns the IPINFO_TOKEN from environment variable, cached after first read
func getIPInfoToken() string {
	ipInfoTokenOnce.Do(func() {
		ipInfoToken = os.Getenv("IPINFO_TOKEN")
	})
	return ipInfoToken
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
	Bogon    bool   `json:"bogon"`
}

// queryIPWithSDK queries IP information using the ipinfo Go SDK
func queryIPWithSDK(runtime *plugin.Runtime, token string, queryIP net.IP) (*ipinfoResponse, error) {
	// Use default HTTP client - no interface binding
	sdkClient := ipinfo.NewClient(nil, nil, token)

	// Query the IP
	var info *ipinfo.Core
	var err error
	if queryIP == nil {
		// Query for YOUR public IP
		info, err = sdkClient.GetIPInfo(nil)
	} else {
		// Query for the specific IP
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
		Bogon:    info.Bogon,
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
func queryIPInfo(runtime *plugin.Runtime, queryIP net.IP, token string) (*ipinfoResponse, error) {
	if token == "" {
		// Use free API with default client
		httpClient := &http.Client{
			Timeout: 30 * time.Second,
		}
		return queryIPWithFreeAPI(httpClient, queryIP)
	}

	// Use SDK
	return queryIPWithSDK(runtime, token, queryIP)
}

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	log.Debug().Str("args", fmt.Sprintf("%+v", args)).Msg("initIpinfo called")

	token := getIPInfoToken()

	var queryIP net.IP
	var requestedIP net.IP

	// Check if an IP was provided as input
	if ip, ok := args["ip"]; ok {
		ipVal, ok := ip.Value.(llx.RawIP)
		if !ok {
			return nil, nil, errors.New("ip must be of type ip")
		}
		if ipVal.IP == nil {
			return nil, nil, errors.New("ip cannot be empty")
		}
		queryIP = ipVal.IP
		requestedIP = ipVal.IP
	}
	// If no IP provided, queryIP remains nil (query for your public IP)
	log.Debug().
		Str("queryIP", func() string {
			if queryIP == nil {
				return "nil (public IP)"
			}
			return queryIP.String()
		}()).
		Str("withSDK", fmt.Sprintf("%t", token != "")).
		Msg("querying ipinfo")

	// Query IP information using the appropriate method
	info, err := queryIPInfo(runtime, queryIP, token)
	if err != nil {
		log.Debug().Err(err).Msg("ipinfo query failed")
		return nil, nil, err
	}

	if info == nil {
		return nil, nil, errors.New("ipinfo query returned no data")
	}

	log.Debug().
		Str("response_ip", info.IP).
		Str("response_hostname", info.Hostname).
		Bool("response_bogon", info.Bogon).
		Interface("full_response", info).
		Msg("ipinfo response")

	// Create result map with output fields
	res := make(map[string]*llx.RawData)

	if requestedIP != nil {
		res["requested_ip"] = llx.IPData(llx.RawIP{IP: requestedIP})
	} else {
		res["requested_ip"] = llx.NilData
	}

	res["returned_ip"] = llx.IPData(llx.ParseIP(info.IP))
	res["hostname"] = llx.StringData(info.Hostname)
	res["bogon"] = llx.BoolData(info.Bogon)

	return res, nil, nil
}

func (c *mqlIpinfo) id() (string, error) {
	if c.Returned_ip.Error != nil {
		return "", c.Returned_ip.Error
	}
	return "ipinfo\x00" + c.Returned_ip.Data.String(), nil
}
