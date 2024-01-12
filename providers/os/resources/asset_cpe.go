// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/facebookincubator/nvdtools/wfn"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"go.mondoo.com/cnquery/v10/providers/os/resources/cpe"
	"strings"
)

func (a *mqlAsset) cpes() ([]interface{}, error) {
	// 1 - try to read the cpe from the file
	lf, err := CreateResource(a.MqlRuntime, "file", map[string]*llx.RawData{
		"path": llx.StringData("/etc/system-release-cpe"),
	})
	if err != nil {
		return nil, err
	}
	file := lf.(*mqlFile)
	data := file.GetContent()
	if data.Error == nil {
		// cpe:2.3:o:amazon:amazon_linux:2023 is not complete
		attr, err := wfn.Parse(strings.TrimSpace(data.Data))
		if err == nil {
			cpe, err := a.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
				"uri": llx.StringData(attr.BindToFmtString()),
			})
			if err != nil {
				return nil, err
			}
			return []interface{}{cpe}, nil
		}
	}

	// 2 - use platform and version to generate the cpe
	conn, ok := a.MqlRuntime.Connection.(shared.Connection)
	if ok && conn.Asset() != nil && conn.Asset().Platform != nil {
		// on windows, we need to determine if we are on a workstation
		workstation := false
		if conn.Asset().Platform.Labels["windows.mondoo.com/product-type"] == "1" {
			workstation = true
		}

		cpe, ok := cpe.PlatformCPE(conn.Asset().Platform.Name, conn.Asset().Platform.Version, workstation)
		if ok {
			cpe, err := a.MqlRuntime.CreateSharedResource("cpe", map[string]*llx.RawData{
				"uri": llx.StringData(cpe),
			})
			if err != nil {
				return nil, err
			}
			return []interface{}{cpe}, nil
		}
	}

	return nil, nil
}
