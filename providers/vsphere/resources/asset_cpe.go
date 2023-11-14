// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/facebookincubator/nvdtools/wfn"
	"go.mondoo.com/cnquery/v9/providers/vsphere/connection"
)

func (a *mqlAsset) cpes() ([]interface{}, error) {
	conn, ok := a.MqlRuntime.Connection.(*connection.VsphereConnection)
	if ok && conn.Asset() != nil && conn.Asset().Platform != nil {
		switch conn.Asset().Platform.Name {
		case connection.VspherePlatform:
			// follow the following format cpe:2.3:a:vmware:vcenter_server:7.0:update1d:*:*:*:*:*:*
			attr := wfn.Attributes{
				Part:    "a",
				Vendor:  "vmware",
				Product: "vcenter_server",
				Version: conn.Asset().Platform.Version,
				Update:  conn.Asset().Platform.Build,
			}
			return []interface{}{attr.BindToFmtString()}, nil

		case connection.EsxiPlatform:
			// follow the following format // cpe:2.3:o:vmware:esxi:7.0:update_3c:*:*:*:*:*:*
			attr := wfn.Attributes{
				Part:    "o",
				Vendor:  "vmware",
				Product: "esxi",
				Version: conn.Asset().Platform.Version,
				Update:  conn.Asset().Platform.Build,
			}
			return []interface{}{attr.BindToFmtString()}, nil
		}
	}
	return nil, nil
}
