// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// resolveOciImage resolves a typed image resource from an image OCID. Returns
// (nil, nil) and marks the field as null when the OCID is empty.
func resolveOciImage(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciComputeImage]) (*mqlOciComputeImage, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "oci.compute.image", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciComputeImage), nil
}

// resolveOciVault resolves a typed vault resource from a vault OCID. Returns
// (nil, nil) and marks the field as null when the OCID is empty.
func resolveOciVault(runtime *plugin.Runtime, id string, field *plugin.TValue[*mqlOciKmsVault]) (*mqlOciKmsVault, error) {
	if id == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(runtime, "oci.kms.vault", map[string]*llx.RawData{
		"id": llx.StringData(id),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOciKmsVault), nil
}

// resolveOciSecurityGroups resolves a list of typed network security group
// resources from a list of NSG OCIDs. Empty list returns ([], nil).
func resolveOciSecurityGroups(runtime *plugin.Runtime, ids []any) ([]any, error) {
	out := make([]any, 0, len(ids))
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok || id == "" {
			continue
		}
		res, err := NewResource(runtime, "oci.network.networkSecurityGroup", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// resolveOciTopics resolves a list of typed ONS topic resources from a list of
// topic OCIDs. Non-topic OCIDs (alarms can target other destination types in
// the future) are skipped silently.
func resolveOciTopics(runtime *plugin.Runtime, ids []any) ([]any, error) {
	out := make([]any, 0, len(ids))
	for _, raw := range ids {
		id, ok := raw.(string)
		if !ok || id == "" {
			continue
		}
		res, err := NewResource(runtime, "oci.ons.topic", map[string]*llx.RawData{
			"id": llx.StringData(id),
		})
		if err != nil {
			return nil, err
		}
		out = append(out, res)
	}
	return out, nil
}

// ----- compute -----

func (o *mqlOciComputeInstance) image() (*mqlOciComputeImage, error) {
	return resolveOciImage(o.MqlRuntime, o.ImageId.Data, &o.Image)
}

func (o *mqlOciComputeBootVolume) image() (*mqlOciComputeImage, error) {
	return resolveOciImage(o.MqlRuntime, o.ImageId.Data, &o.Image)
}

func (o *mqlOciComputeBootVolume) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciComputeBlockVolume) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciComputeVnic) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciComputeVnic) securityGroups() ([]any, error) {
	if o.NsgIds.Error != nil {
		return nil, o.NsgIds.Error
	}
	return resolveOciSecurityGroups(o.MqlRuntime, o.NsgIds.Data)
}

// ----- kms -----

func (o *mqlOciKmsVault) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciKmsKey) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciKmsKey) vault() (*mqlOciKmsVault, error) {
	return resolveOciVault(o.MqlRuntime, o.VaultId.Data, &o.Vault)
}

func (o *mqlOciKmsKeyVersion) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciKmsKeyVersion) vault() (*mqlOciKmsVault, error) {
	return resolveOciVault(o.MqlRuntime, o.VaultId.Data, &o.Vault)
}

// ----- events / ons / monitoring -----

func (o *mqlOciEventsRule) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciOnsTopic) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciMonitoringAlarm) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciMonitoringAlarm) topics() ([]any, error) {
	if o.Destinations.Error != nil {
		return nil, o.Destinations.Error
	}
	return resolveOciTopics(o.MqlRuntime, o.Destinations.Data)
}

// ----- bastion / vault.secret -----

func (o *mqlOciBastionInstance) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciVaultSecret) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

// ----- database -----

func (o *mqlOciDatabaseDbSystem) securityGroups() ([]any, error) {
	if o.NsgIds.Error != nil {
		return nil, o.NsgIds.Error
	}
	return resolveOciSecurityGroups(o.MqlRuntime, o.NsgIds.Data)
}

func (o *mqlOciDatabaseDbSystem) backupSecurityGroups() ([]any, error) {
	if o.BackupNetworkNsgIds.Error != nil {
		return nil, o.BackupNetworkNsgIds.Error
	}
	return resolveOciSecurityGroups(o.MqlRuntime, o.BackupNetworkNsgIds.Data)
}

func (o *mqlOciDatabaseAutonomousDatabase) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.CompartmentID.Data, &o.Compartment)
}

func (o *mqlOciDatabaseAutonomousDatabase) securityGroups() ([]any, error) {
	if o.NsgIds.Error != nil {
		return nil, o.NsgIds.Error
	}
	return resolveOciSecurityGroups(o.MqlRuntime, o.NsgIds.Data)
}
