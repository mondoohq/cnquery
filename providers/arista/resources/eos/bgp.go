// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package eos

import "github.com/aristanetworks/goeapi/module"

// showIPBgpSummary represents the response from "show ip bgp summary"
type showIPBgpSummary struct {
	VRFs map[string]showBgpVrf `json:"vrfs"`
}

func (s *showIPBgpSummary) GetCmd() string {
	return "show ip bgp summary"
}

type showBgpVrf struct {
	RouterID string                 `json:"routerId"`
	ASN      int64                  `json:"asn"`
	Peers    map[string]showBgpPeer `json:"peers"`
}

type showBgpPeer struct {
	PeerState        string  `json:"peerState"`
	InMsgQueue       int64   `json:"inMsgQueue"`
	OutMsgQueue      int64   `json:"outMsgQueue"`
	UpDownTime       float64 `json:"upDownTime"`
	PrefixAccepted   int64   `json:"prefixAccepted"`
	PrefixReceived   int64   `json:"prefixReceived"`
	ASN              string  `json:"asn"`
	UnderMaintenance bool    `json:"underMaintenance"`
}

// BGPSummary returns BGP summary information for all VRFs
func (eos *Eos) BGPSummary() (*showIPBgpSummary, error) {
	shRsp := &showIPBgpSummary{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shRsp)
	if err != nil {
		return nil, err
	}

	if err := handle.Call(); err != nil {
		return nil, err
	}

	handle.Close()

	return shRsp, nil
}

// BGPConfig returns BGP configuration using the goeapi module
func (eos *Eos) BGPConfig() *module.BgpConfig {
	bgpModule := module.Bgp(eos.node)
	return bgpModule.Get()
}

// showIPBgpNeighbors represents the response from "show ip bgp neighbors"
type showIPBgpNeighbors struct {
	VRFs map[string]showBgpNeighborsVrf `json:"vrfs"`
}

func (s *showIPBgpNeighbors) GetCmd() string {
	return "show ip bgp neighbors"
}

type showBgpNeighborsVrf struct {
	PeerList []showBgpNeighborDetail `json:"peerList"`
}

type showBgpNeighborDetail struct {
	PeerAddress      string `json:"peerAddress"`
	Description      string `json:"description"`
	ASN              string `json:"asn"`
	InboundRouteMap  string `json:"inboundRouteMap"`
	OutboundRouteMap string `json:"outboundRouteMap"`
}

// BGPNeighbors returns detailed BGP neighbor information
func (eos *Eos) BGPNeighbors() (*showIPBgpNeighbors, error) {
	shRsp := &showIPBgpNeighbors{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	err = handle.AddCommand(shRsp)
	if err != nil {
		return nil, err
	}

	if err := handle.Call(); err != nil {
		return nil, err
	}

	handle.Close()

	return shRsp, nil
}
