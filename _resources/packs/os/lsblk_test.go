package os

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBlockEntries(t *testing.T) {
	data := `{"blockdevices": [
			 {
					"name": "loop0",
					"fstype": "squashfs",
					"fsver": "4.0",
					"label": null,
					"uuid": null,
					"fsavail": "0",
					"fsuse%": "100%",
					"mountpoints": [
							"/var/lib/snapd/snap/core/10577"
					]
			 },{
					"name": "sda",
					"fstype": "btrfs",
					"fsver": null,
					"label": "storage01",
					"uuid": "6060df9a-7e53-439c-9189-ba9657161fd4",
					"fsavail": "764.8G",
					"fsuse%": "80%",
					"mountpoints": [
							"/data"
					]
			 },{
					"name": "sdb",
					"fstype": "btrfs",
					"fsver": null,
					"label": "storage01",
					"uuid": "6060df9a-7e53-439c-9189-ba9657161fd4",
					"fsavail": null,
					"fsuse%": null,
					"mountpoints": [
							null
					]
			 },{
					"name": "sdc",
					"fstype": null,
					"fsver": null,
					"label": null,
					"uuid": null,
					"fsavail": null,
					"fsuse%": null,
					"mountpoints": [
							null
					],
					"children": [
						 {
								"name": "sdc1",
								"fstype": "vfat",
								"fsver": "FAT32",
								"label": null,
								"uuid": "0EC7-F4C1",
								"fsavail": "193.5M",
								"fsuse%": "62%",
								"mountpoints": [
										"/boot"
								]
						 },{
								"name": "sdc2",
								"fstype": "ext4",
								"fsver": "1.0",
								"label": null,
								"uuid": "6c44ec5a-4727-47d4-b485-81cff72b207e",
								"fsavail": "80.2G",
								"fsuse%": "77%",
								"mountpoints": [
										"/"
								]
						 }
					]
			 },{
					"name": "sdd",
					"fstype": "btrfs",
					"fsver": null,
					"label": "storage01",
					"uuid": "6060df9a-7e53-439c-9189-ba9657161fd4",
					"fsavail": null,
					"fsuse%": null,
					"mountpoints": [
							null
					]
			 }
		]
 }`
	devices, err := parseBlockEntries([]byte(data))
	assert.Nil(t, err)
	assert.Equal(t, len(devices.Blockdevices), 5)
	assert.Equal(t, devices.Blockdevices, []blockdevice{{
		Name:        "loop0",
		Fstype:      "squashfs",
		Label:       "",
		Uuid:        "",
		Mountpoints: []interface{}{"/var/lib/snapd/snap/core/10577"},
	}, {
		Name:        "sda",
		Fstype:      "btrfs",
		Label:       "storage01",
		Uuid:        "6060df9a-7e53-439c-9189-ba9657161fd4",
		Mountpoints: []interface{}{"/data"},
	}, {
		Name:        "sdb",
		Fstype:      "btrfs",
		Label:       "storage01",
		Uuid:        "6060df9a-7e53-439c-9189-ba9657161fd4",
		Mountpoints: []interface{}{nil},
	}, {
		Name:        "sdc",
		Fstype:      "",
		Label:       "",
		Uuid:        "",
		Mountpoints: []interface{}{nil},
		Children: []blockdevice{{
			Name:        "sdc1",
			Fstype:      "vfat",
			Label:       "",
			Uuid:        "0EC7-F4C1",
			Mountpoints: []interface{}{"/boot"},
		}, {
			Name:        "sdc2",
			Fstype:      "ext4",
			Label:       "",
			Uuid:        "6c44ec5a-4727-47d4-b485-81cff72b207e",
			Mountpoints: []interface{}{"/"},
		}},
	}, {
		Name:        "sdd",
		Fstype:      "btrfs",
		Label:       "storage01",
		Uuid:        "6060df9a-7e53-439c-9189-ba9657161fd4",
		Mountpoints: []interface{}{nil},
	}})

	data = `{
		"blockdevices": [
			 {"name": "xvda", "fstype": null, "label": null, "uuid": null, "mountpoint": null,
					"children": [
						 {"name": "xvda1", "fstype": "xfs", "label": "/", "uuid": "e6c06bf4-70a3-4524-84fa-35484afc0d19", "mountpoint": "/"}
					]
			 }
		]
 }`
	devices, err = parseBlockEntries([]byte(data))
	assert.Nil(t, err)
	assert.Equal(t, len(devices.Blockdevices), 1)
	assert.Equal(t, devices.Blockdevices, []blockdevice{{
		Name:       "xvda",
		Mountpoint: "",
		Children: []blockdevice{{
			Name:       "xvda1",
			Fstype:     "xfs",
			Label:      "/",
			Uuid:       "e6c06bf4-70a3-4524-84fa-35484afc0d19",
			Mountpoint: "/",
		}},
	}})
}
