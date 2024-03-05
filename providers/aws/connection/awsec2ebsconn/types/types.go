// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package awsec2ebstypes

import "path"

const (
	EBSTargetInstance = "instance"
	EBSTargetVolume   = "volume"
	EBSTargetSnapshot = "snapshot"
)

type SnapshotId struct {
	Id      string
	Region  string
	Account string
}

type EbsTransportTarget struct {
	Account string
	Region  string
	Id      string
	Type    string
}

type TargetInfo struct {
	PlatformId string
	AccountId  string
	Region     string
	Id         string
}

type InstanceId struct {
	Id             string
	Region         string
	Name           string
	Account        string
	Zone           string
	MarketplaceImg bool
}

type VolumeInfo struct {
	Id          string
	Region      string
	Account     string
	IsAvailable bool
	Tags        map[string]string
}

func (s *InstanceId) String() string {
	// e.g. account/999000999000/region/us-east-2/instances/i-0989478343232
	return path.Join("account", s.Account, "region", s.Region, "instances", s.Id)
}
