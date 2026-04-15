// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"

	"github.com/oracle/oci-go-sdk/v65/audit"
	"github.com/oracle/oci-go-sdk/v65/common"
	"go.mondoo.com/mql/v13/providers/oci/connection"
)

func (o *mqlOciAudit) id() (string, error) {
	return "oci.audit", nil
}

func (o *mqlOciAudit) retentionPeriodDays() (int64, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	// Audit configuration is tenancy-level; use home region
	tenancy, err := conn.Tenant(context.Background())
	if err != nil {
		return 0, err
	}

	region := ""
	if tenancy.HomeRegionKey != nil {
		region = *tenancy.HomeRegionKey
	}

	client, err := conn.AuditClient(region)
	if err != nil {
		return 0, err
	}

	resp, err := client.GetConfiguration(context.Background(), audit.GetConfigurationRequest{
		CompartmentId: common.String(conn.TenantID()),
	})
	if err != nil {
		return 0, err
	}

	if resp.RetentionPeriodDays == nil {
		return 0, nil
	}
	return int64(*resp.RetentionPeriodDays), nil
}
