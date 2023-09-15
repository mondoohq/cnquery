// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package ids

const (
	IdDetector_Hostname    = "hostname"
	IdDetector_MachineID   = "machine-id"
	IdDetector_CloudDetect = "cloud-detect"
	IdDetector_SshHostkey  = "ssh-host-key"
	IdDetector_AwsEcs      = "aws-ecs"

	// FIXME: DEPRECATED, remove in v9.0 vv
	// this is now cloud-detect
	IdDetector_AwsEc2 = "aws-ec2"
	// ^^

	// IdDetector_PlatformID = "transport-platform-id" // TODO: how does this work?
)
