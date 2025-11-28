// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"net"

	"github.com/cockroachdb/errors"
	"github.com/ipinfo/go/v2/ipinfo"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/ipinfo/connection"
)

// queryIPWithSDK queries IP information using the ipinfo Go SDK
func queryIPWithSDK(token string, queryIP net.IP) (*ipinfo.Core, error) {
	sdkClient := ipinfo.NewClient(nil, nil, token)

	// Query the IP
	var info *ipinfo.Core
	var err error
	if queryIP == nil {
		info, err = sdkClient.GetIPInfo(nil)
	} else {
		info, err = sdkClient.GetIPInfo(queryIP)
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to query with ipinfo SDK")
	}

	return info, nil
}

func initIpinfo(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	log.Debug().Str("args", fmt.Sprintf("%+v", args)).Msg("initIpinfo called")

	conn := runtime.Connection.(*connection.IpinfoConnection)
	token := conn.Token()

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
		Msg("querying ipinfo")

	// Query IP information using the appropriate method
	info, err := queryIPWithSDK(token, queryIP)
	if err != nil {
		log.Debug().Err(err).Msg("ipinfo query failed")
		return nil, nil, err
	}

	if info == nil {
		return nil, nil, errors.New("ipinfo query returned no data")
	}

	log.Debug().
		Str("response_ip", info.IP.String()).
		Str("response_hostname", info.Hostname).
		Bool("response_bogon", info.Bogon).
		Interface("full_response", info).
		Msg("ipinfo response")

	res := make(map[string]*llx.RawData)
	if requestedIP != nil {
		res["requested_ip"] = llx.IPData(llx.RawIP{IP: requestedIP})
	} else {
		res["requested_ip"] = llx.NilData
	}

	res["returned_ip"] = llx.IPData(llx.ParseIP(info.IP.String()))
	res["hostname"] = llx.StringData(info.Hostname)
	res["bogon"] = llx.BoolData(info.Bogon)
	res["city"] = llx.StringData(info.City)
	res["region"] = llx.StringData(info.Region)
	res["country"] = llx.StringData(info.Country)
	res["country_name"] = llx.StringData(info.CountryName)
	res["is_eu"] = llx.BoolData(info.IsEU)
	res["location"] = llx.StringData(info.Location)
	res["org"] = llx.StringData(info.Org)
	res["postal"] = llx.StringData(info.Postal)
	res["timezone"] = llx.StringData(info.Timezone)

	return res, nil, nil
}

func (c *mqlIpinfo) id() (string, error) {
	if c.Returned_ip.Error != nil {
		return "", c.Returned_ip.Error
	}
	return "ipinfo\x00" + c.Returned_ip.Data.String(), nil
}
