// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package azcompute

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v11/providers/os/resources/powershell"
	"go.mondoo.com/cnquery/v11/utils/multierr"
)

const (
	// https://learn.microsoft.com/en-us/azure/virtual-machines/instance-metadata-service?tabs=windows#supported-api-versions
	//
	// We are not using version 2023-11-15 since it is still being rolled out and it may not be available in some regions.
	IMDSApiVersion = "2023-07-01"

	instanceMetadataScriptUnix    = "curl --retry 5 --retry-delay 1 --connect-timeout 1 --retry-max-time 5 --max-time 10 --noproxy '*' -H Metadata:true http://169.254.169.254/metadata/instance?api-version=%s"
	metadataIdentityScriptWindows = `
$Headers = @{
    "Metadata" = "true"
}
Invoke-RestMethod -TimeoutSec 5 -Headers $Headers -URI http://169.254.169.254/metadata/instance?api-version=%s -UseBasicParsing | ConvertTo-Json
`

	loadbalancerMetadataScriptUnix    = "curl --retry 5 --retry-delay 1 --connect-timeout 1 --retry-max-time 5 --max-time 10 --noproxy '*' -H Metadata:true http://169.254.169.254/metadata/loadbalancer?api-version=%s"
	loadbalancerMetadataScriptWindows = `
$Headers = @{
    "Metadata" = "true"
}
Invoke-RestMethod -TimeoutSec 5 -Headers $Headers -URI http://169.254.169.254/metadata/loadbalancer?api-version=%s -UseBasicParsing | ConvertTo-Json
`
)

func MondooAzureInstanceID(instanceID string) string {
	return "//platformid.api.mondoo.app/runtime/azure" + instanceID
}

type instanceMetadata struct {
	Compute struct {
		ResourceID     string `json:"resourceId"`
		SubscriptionID string `json:"subscriptionId"`
		Tags           string `json:"tags"`
	} `json:"compute"`
}

type Identity struct {
	InstanceID string
	AccountID  string
}

type InstanceIdentifier interface {
	Identify() (Identity, error)
	RawMetadata() (any, error)
}

func Resolve(conn shared.Connection, pf *inventory.Platform) (InstanceIdentifier, error) {
	if pf.IsFamily(inventory.FAMILY_UNIX) || pf.IsFamily(inventory.FAMILY_WINDOWS) {
		return &commandInstanceMetadata{conn, pf}, nil
	}
	return nil, errors.New("azure compute id detector is not supported for your asset: " + pf.Name + " " + pf.Version)
}

type commandInstanceMetadata struct {
	conn     shared.Connection
	platform *inventory.Platform
}

func (m *commandInstanceMetadata) RawMetadata() (any, error) {
	metadata := map[string]any{}

	data, err := m.instanceDocument()
	if err != nil {
		return nil, err
	}

	var instanceMap map[string]interface{}
	if err = json.Unmarshal(data, &instanceMap); err != nil {
		return nil, err
	}
	metadata["instance"] = instanceMap

	data, err = m.loadbalancerDocument()
	if err != nil {
		return nil, err
	}

	var loadbalancerMap map[string]interface{}
	if err = json.Unmarshal(data, &loadbalancerMap); err != nil {
		return nil, err
	}
	metadata["loadbalancer"] = loadbalancerMap

	return metadata, nil
}

func (m *commandInstanceMetadata) Identify() (Identity, error) {
	document, err := m.instanceDocument()
	if err != nil {
		return Identity{}, err
	}
	// parse into struct
	md := instanceMetadata{}
	if err := json.NewDecoder(bytes.NewReader(document)).Decode(&md); err != nil {
		return Identity{}, multierr.Wrap(err, "failed to decode Azure Instance Metadata")
	}

	return Identity{
		InstanceID: MondooAzureInstanceID(md.Compute.ResourceID),
		AccountID:  "//platformid.api.mondoo.app/runtime/azure/subscriptions/" + md.Compute.SubscriptionID,
	}, nil
}

func (m *commandInstanceMetadata) instanceDocument() ([]byte, error) {
	var (
		cmd *shared.Command
		err error
	)

	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		cmd, err = m.conn.RunCommand(fmt.Sprintf(instanceMetadataScriptUnix, IMDSApiVersion))
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		cmd, err = m.conn.RunCommand(powershell.Encode(fmt.Sprintf(metadataIdentityScriptWindows, IMDSApiVersion)))
	default:
		err = errors.New("your platform is not supported by azure metadata identifier resource")
	}

	if err != nil {
		return nil, err
	}

	return io.ReadAll(cmd.Stdout)
}

func (m *commandInstanceMetadata) loadbalancerDocument() ([]byte, error) {
	var (
		cmd *shared.Command
		err error
	)

	switch {
	case m.platform.IsFamily(inventory.FAMILY_UNIX):
		cmd, err = m.conn.RunCommand(fmt.Sprintf(loadbalancerMetadataScriptUnix, IMDSApiVersion))
	case m.platform.IsFamily(inventory.FAMILY_WINDOWS):
		cmd, err = m.conn.RunCommand(powershell.Encode(fmt.Sprintf(loadbalancerMetadataScriptWindows, IMDSApiVersion)))
	default:
		err = errors.New("your platform is not supported by azure metadata identifier resource")
	}

	if err != nil {
		return nil, err
	}
	return io.ReadAll(cmd.Stdout)
}
