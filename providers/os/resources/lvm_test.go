// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLvmPVs(t *testing.T) {
	// Output of: pvs --reportformat json --units b --nosuffix -o pv_name,pv_uuid,vg_name,pv_fmt,pv_attr,pv_size,pv_free
	input := `{
      "report": [
          {
              "pv": [
                  {"pv_name":"/dev/sda1", "pv_uuid":"abc-123", "vg_name":"vg0",    "pv_fmt":"lvm2", "pv_attr":"a--", "pv_size":"10737418240", "pv_free":"2147483648"},
                  {"pv_name":"/dev/sdb1", "pv_uuid":"def-456", "vg_name":"",       "pv_fmt":"lvm2", "pv_attr":"---", "pv_size":"5368709120",  "pv_free":"5368709120"}
              ]
          }
      ]
  }
`
	pvs, err := parseLvmPVs(input)
	require.NoError(t, err)
	require.Len(t, pvs, 2)

	assert.Equal(t, "/dev/sda1", pvs[0].Name)
	assert.Equal(t, "abc-123", pvs[0].UUID)
	assert.Equal(t, "vg0", pvs[0].VGName)
	assert.Equal(t, "lvm2", pvs[0].Format)
	assert.Equal(t, "a--", pvs[0].Attributes)
	assert.Equal(t, int64(10737418240), pvs[0].SizeBytes)
	assert.Equal(t, int64(2147483648), pvs[0].FreeBytes)

	// Unassigned PV — vg_name empty
	assert.Equal(t, "/dev/sdb1", pvs[1].Name)
	assert.Equal(t, "", pvs[1].VGName)
}

func TestParseLvmVGs(t *testing.T) {
	input := `{
      "report": [
          {
              "vg": [
                  {"vg_name":"vg0", "vg_uuid":"vg-uuid-1", "vg_attr":"wz--n-", "vg_size":"21474836480", "vg_free":"5368709120", "pv_count":"2", "lv_count":"3", "snap_count":"1"}
              ]
          }
      ]
  }
`
	vgs, err := parseLvmVGs(input)
	require.NoError(t, err)
	require.Len(t, vgs, 1)

	v := vgs[0]
	assert.Equal(t, "vg0", v.Name)
	assert.Equal(t, "vg-uuid-1", v.UUID)
	assert.Equal(t, "wz--n-", v.Attributes)
	assert.Equal(t, int64(21474836480), v.SizeBytes)
	assert.Equal(t, int64(5368709120), v.FreeBytes)
	assert.Equal(t, int64(2), v.PVCount)
	assert.Equal(t, int64(3), v.LVCount)
	assert.Equal(t, int64(1), v.SnapshotCount)
}

func TestParseLvmLVs(t *testing.T) {
	// Mix of: regular LV, snapshot (origin set), thin pool (data_percent set, pool_lv empty),
	// thin LV (pool_lv set).
	input := `{
      "report": [
          {
              "lv": [
                  {"lv_name":"root",    "lv_path":"/dev/vg0/root",    "lv_uuid":"lv-1", "vg_name":"vg0", "lv_attr":"-wi-ao----", "lv_size":"10737418240", "origin":"",     "data_percent":"",      "pool_lv":""},
                  {"lv_name":"snap",    "lv_path":"/dev/vg0/snap",    "lv_uuid":"lv-2", "vg_name":"vg0", "lv_attr":"swi-a-s---", "lv_size":"1073741824",  "origin":"root", "data_percent":"12.50", "pool_lv":""},
                  {"lv_name":"pool",    "lv_path":"",                  "lv_uuid":"lv-3", "vg_name":"vg0", "lv_attr":"twi-aotz--", "lv_size":"5368709120",  "origin":"",     "data_percent":"45.00", "pool_lv":""},
                  {"lv_name":"thindata","lv_path":"/dev/vg0/thindata", "lv_uuid":"lv-4", "vg_name":"vg0", "lv_attr":"Vwi-a-tz--", "lv_size":"2147483648",  "origin":"",     "data_percent":"30.25", "pool_lv":"pool"}
              ]
          }
      ]
  }
`
	lvs, err := parseLvmLVs(input)
	require.NoError(t, err)
	require.Len(t, lvs, 4)

	// Regular LV — data_percent is empty -> nil
	assert.Equal(t, "root", lvs[0].Name)
	assert.Equal(t, "/dev/vg0/root", lvs[0].Path)
	assert.Equal(t, "-wi-ao----", lvs[0].Attributes)
	assert.Equal(t, int64(10737418240), lvs[0].SizeBytes)
	assert.Equal(t, "", lvs[0].Origin)
	assert.Nil(t, lvs[0].DataPercent)
	assert.Equal(t, "", lvs[0].PoolName)

	// Snapshot
	assert.Equal(t, "snap", lvs[1].Name)
	assert.Equal(t, "root", lvs[1].Origin)
	require.NotNil(t, lvs[1].DataPercent)
	assert.Equal(t, 12.5, *lvs[1].DataPercent)

	// Thin pool
	assert.Equal(t, "pool", lvs[2].Name)
	require.NotNil(t, lvs[2].DataPercent)
	assert.Equal(t, 45.0, *lvs[2].DataPercent)

	// Thin volume
	assert.Equal(t, "thindata", lvs[3].Name)
	assert.Equal(t, "pool", lvs[3].PoolName)
	require.NotNil(t, lvs[3].DataPercent)
	assert.Equal(t, 30.25, *lvs[3].DataPercent)
}

func TestParseLvmEmptyReport(t *testing.T) {
	// LVM emits an empty array when no objects exist.
	input := `{"report": [{"pv": []}]}`
	pvs, err := parseLvmPVs(input)
	require.NoError(t, err)
	assert.Empty(t, pvs)
}

func TestParseLvmInt(t *testing.T) {
	v, err := parseLvmInt("pv_size", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), v)

	v, err = parseLvmInt("pv_size", "   ")
	require.NoError(t, err)
	assert.Equal(t, int64(0), v)

	v, err = parseLvmInt("pv_size", "12345")
	require.NoError(t, err)
	assert.Equal(t, int64(12345), v)

	v, err = parseLvmInt("pv_size", "  12345 ")
	require.NoError(t, err)
	assert.Equal(t, int64(12345), v)

	// Garbage values are surfaced as errors so callers don't silently
	// substitute 0 for an unparseable column.
	_, err = parseLvmInt("pv_size", "not a number")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pv_size")
}

func TestParseLvmFloat(t *testing.T) {
	// Empty is "not applicable" — nil pointer.
	v, err := parseLvmFloat("data_percent", "")
	require.NoError(t, err)
	assert.Nil(t, v)

	v, err = parseLvmFloat("data_percent", "12.5")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, 12.5, *v)

	v, err = parseLvmFloat("data_percent", "100.00")
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, 100.0, *v)

	// Garbage values are surfaced as errors.
	_, err = parseLvmFloat("data_percent", "nope")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data_percent")
}
