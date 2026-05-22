// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

// VrrpGroup represents a single VRRP virtual router as returned by
// `show vrrp all | json`. Field tags follow the EOS eAPI JSON schema.
type VrrpGroup struct {
	Interface             string   `json:"interface"`
	GroupId               int64    `json:"groupId"`
	Version               int64    `json:"version"`
	Priority              int64    `json:"priority"`
	Preempt               bool     `json:"preempt"`
	PreemptDelay          int64    `json:"preemptDelay"`
	State                 string   `json:"groupState"`
	PrimaryIp             string   `json:"primaryIp"`
	VirtualMac            string   `json:"virtualMac"`
	AdvertisementInterval float64  `json:"advertisementInterval"`
	SkewTime              float64  `json:"skewTime"`
	VirtualIps            []string `json:"virtualIps"`
}

type showVrrp struct {
	VirtualRouters []VrrpGroup `json:"virtualRouters"`
}

func (s *showVrrp) GetCmd() string {
	return "show vrrp all"
}

// VrrpGroups returns the VRRP groups configured on the device. Returns an
// empty slice if VRRP is not configured (the EOS response is an empty
// virtualRouters array, not an error).
func (eos *Eos) VrrpGroups() ([]VrrpGroup, error) {
	shRsp := &showVrrp{}

	handle, err := eos.node.GetHandle("json")
	if err != nil {
		return nil, err
	}
	if err := handle.AddCommand(shRsp); err != nil {
		return nil, err
	}
	if err := handle.Call(); err != nil {
		return nil, err
	}
	handle.Close()

	return shRsp.VirtualRouters, nil
}
