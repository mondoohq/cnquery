package cpe

import (
	"testing"

	"github.com/facebookincubator/nvdtools/wfn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWfnParser(t *testing.T) {
	type testdata struct {
		uri string
		cpe *Cpe23
	}

	tests := []testdata{
		{
			uri: "cpe:2.3:part:vendor:product:version:update:edition:language:sw_edition:target_sw:target_hw:other",
			cpe: &Cpe23{
				Part:      "part",
				Vendor:    "vendor",
				Product:   "product",
				Version:   "version",
				Update:    "update",
				Edition:   "edition",
				Language:  "language",
				SWEdition: "sw_edition",
				TargetSW:  "target_sw",
				TargetHW:  "target_hw",
				Other:     "other",
			},
		},
		{
			uri: "cpe:2.3:a:static-resource-server_project:static-resource-server:1.7.2:*:*:*:*:node.js:*:*",
			cpe: &Cpe23{
				Part:      "a",
				Vendor:    "static\\-resource\\-server_project",
				Product:   "static\\-resource\\-server",
				Version:   "1\\.7\\.2",
				Update:    "",
				Edition:   "",
				Language:  "",
				SWEdition: "",
				TargetSW:  "node\\.js",
				TargetHW:  "",
				Other:     "",
			},
		},
		{
			uri: "cpe:2.3:a:microsoft:internet_explorer:8.0.6001:beta:*:*:*:*:*:*",
			cpe: &Cpe23{
				Part:      "a",
				Vendor:    "microsoft",
				Product:   "internet_explorer",
				Version:   "8\\.0\\.6001",
				Update:    "beta",
				Edition:   "",
				Language:  "",
				SWEdition: "",
				TargetSW:  "",
				TargetHW:  "",
				Other:     "",
			},
		},
		{
			// (Application) Microsoft Office 2007 Professional Service Pack 2
			uri: "cpe:2.3:a:microsoft:office:2007:sp2:-:*:professional:*:*:*",
			cpe: &Cpe23{
				Part:      "a",
				Vendor:    "microsoft",
				Product:   "office",
				Version:   "2007",
				Update:    "sp2",
				Edition:   "-",
				Language:  "",
				SWEdition: "professional",
				TargetSW:  "",
				TargetHW:  "",
				Other:     "",
			},
		},
		{
			//  (Operating System) Microsoft Windows 7 64-bit Service Pack 1
			uri: "cpe:2.3:o:microsoft:windows_7:-:sp1:-:*:*:*:x64:*",
			cpe: &Cpe23{
				Part:      "o",
				Vendor:    "microsoft",
				Product:   "windows_7",
				Version:   "-",
				Update:    "sp1",
				Edition:   "-",
				Language:  "",
				SWEdition: "",
				TargetSW:  "",
				TargetHW:  "x64",
				Other:     "",
			},
		},
		{
			uri: "cpe:2.3:o:linux:linux_kernel:2.6.27.51:*:*:*:*:*:*:*",
			cpe: &Cpe23{
				Part:      "o",
				Vendor:    "linux",
				Product:   "linux_kernel",
				Version:   "2\\.6\\.27\\.51",
				Update:    "",
				Edition:   "",
				Language:  "",
				SWEdition: "",
				TargetSW:  "",
				TargetHW:  "",
				Other:     "",
			},
		},
	}

	for i := range tests {
		// verify  that we can parse it
		cpe, err := wfn.Parse(tests[i].uri)
		if err != nil {
			t.Fatal(err)
		}
		assert.EqualValues(t, tests[i].cpe, cpe)

		// check that we have looseless conversion back
		relation, err := wfn.Compare(tests[i].cpe, cpe)
		require.NoError(t, err)
		assert.False(t, relation.IsDisjoint())

		// check that we can convert it back to string
		assert.Equal(t, tests[i].uri, cpe.BindToFmtString())
	}
}
