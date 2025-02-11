package linux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mondoo.com/cnquery/v11/providers/os/connection/snapshot"
	"go.mondoo.com/cnquery/v11/providers/os/resources"
	"go.uber.org/mock/gomock"
	"k8s.io/utils/ptr"
)

type deviceManagerTestFixture struct {
	dmgr          *LinuxDeviceManager
	volumeMounter *snapshot.MockVolumeMounter

	mockCtrl *gomock.Controller
}

func newFixture(t *testing.T) *deviceManagerTestFixture {
	ctrl := gomock.NewController(t)
	volumeMounter := snapshot.NewMockVolumeMounter(ctrl)

	return &deviceManagerTestFixture{
		dmgr: &LinuxDeviceManager{
			volumeMounter: volumeMounter,
			opts:          make(map[string]string),
		},
		volumeMounter: volumeMounter,

		mockCtrl: ctrl,
	}
}

func TestMountWithFstab(t *testing.T) {
	f := newFixture(t)

	t.Run("happy path", func(t *testing.T) {
		partitions := []*snapshot.PartitionInfo{
			{
				Name:   "/dev/sdf1",
				FsType: "ext4",
				Uuid:   "sdf1-uuid",
			},
			{
				Name:     "/dev/sdf3",
				FsType:   "fat32",
				PartUuid: "sdf3-uuid",
			},
			{
				Name:   "/dev/sdg1",
				FsType: "ext4",
				Label:  "data-label",
			},
		}
		entries := []resources.FstabEntry{
			{
				Device:     "UUID=sdf1-uuid",
				Mountpoint: "/",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
			{
				Device:     "PARTUUID=sdf3-uuid",
				Mountpoint: "/boot/efi",
				Fstype:     "fat32",
				Options:    []string{"defaults"},
			},
			{
				Device:     "LABEL=data-label",
				Mountpoint: "/data",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
		}

		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[0], ScanDir: nil,
		})).Return("/tmp/scandir", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[1], ScanDir: ptr.To("/tmp/scandir/boot/efi"),
		})).Return("/tmp/scandir/boot/efi", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[2], ScanDir: ptr.To("/tmp/scandir/data"),
		})).Return("/tmp/scandir/data", nil).Times(1)

		result, err := f.dmgr.mountWithFstab(partitions, entries)
		assert.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Equal(t, "/dev/sdf1", result[0].Name)
		assert.Equal(t, "/tmp/scandir", result[0].MountPoint)

		assert.Equal(t, "/dev/sdf3", result[1].Name)
		assert.Equal(t, "/tmp/scandir/boot/efi", result[1].MountPoint)

		assert.Equal(t, "/dev/sdg1", result[2].Name)
		assert.Equal(t, "/tmp/scandir/data", result[2].MountPoint)
	})

	t.Run("no entries matched", func(t *testing.T) {
		partitions := []*snapshot.PartitionInfo{
			{
				Name:   "/dev/sdf1",
				FsType: "ext4",
				Uuid:   "sdf1-uuid",
			},
			{
				Name:     "/dev/sdf3",
				FsType:   "fat32",
				PartUuid: "sdf3-uuid",
			},
			{
				Name:   "/dev/sdg1",
				FsType: "ext4",
				Label:  "data-label",
			},
		}
		entries := []resources.FstabEntry{
			{
				Device:     "UUID=sdf1-wrong-uuid",
				Mountpoint: "/",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
			{
				Device:     "PARTUUID=sdf3-wrong-uuid",
				Mountpoint: "/boot/efi",
				Fstype:     "fat32",
				Options:    []string{"defaults"},
			},
			{
				Device:     "LABEL=data-wrong-label",
				Mountpoint: "/data",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
		}

		result, err := f.dmgr.mountWithFstab(partitions, entries)
		assert.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Equal(t, "/dev/sdf1", result[0].Name)
		assert.Equal(t, "", result[0].MountPoint)

		assert.Equal(t, "/dev/sdf3", result[1].Name)
		assert.Equal(t, "", result[1].MountPoint)

		assert.Equal(t, "/dev/sdg1", result[2].Name)
		assert.Equal(t, "", result[2].MountPoint)
	})

	t.Run("root not found", func(t *testing.T) {
		partitions := []*snapshot.PartitionInfo{
			{
				Name:   "/dev/sdf1",
				FsType: "ext4",
				Uuid:   "sdf1-uuid",
			},
			{
				Name:     "/dev/sdf3",
				FsType:   "fat32",
				PartUuid: "sdf3-uuid",
			},
			{
				Name:   "/dev/sdg1",
				FsType: "ext4",
				Label:  "data-label",
			},
		}
		entries := []resources.FstabEntry{
			{
				Device:     "UUID=sdf1-wrong-uuid",
				Mountpoint: "/",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
			{
				Device:     "PARTUUID=sdf3-uuid",
				Mountpoint: "/boot/efi",
				Fstype:     "fat32",
				Options:    []string{"defaults"},
			},
			{
				Device:     "LABEL=data-label",
				Mountpoint: "/data",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
		}

		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[1], ScanDir: nil,
		})).Return("/tmp/scandir1", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[2], ScanDir: nil,
		})).Return("/tmp/scandir2", nil).Times(1)

		result, err := f.dmgr.mountWithFstab(partitions, entries)
		assert.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Equal(t, "/dev/sdf1", result[0].Name)
		assert.Equal(t, "", result[0].MountPoint)

		assert.Equal(t, "/dev/sdf3", result[1].Name)
		assert.Equal(t, "/tmp/scandir1", result[1].MountPoint)

		assert.Equal(t, "/dev/sdg1", result[2].Name)
		assert.Equal(t, "/tmp/scandir2", result[2].MountPoint)
	})

	t.Run("one not found", func(t *testing.T) {
		partitions := []*snapshot.PartitionInfo{
			{
				Name:   "/dev/sdf1",
				FsType: "ext4",
				Uuid:   "sdf1-uuid",
			},
			{
				Name:     "/dev/sdf3",
				FsType:   "fat32",
				PartUuid: "sdf3-uuid",
			},
			{
				Name:   "/dev/sdg1",
				FsType: "ext4",
				Label:  "data-label",
			},
		}
		entries := []resources.FstabEntry{
			{
				Device:     "UUID=sdf1-uuid",
				Mountpoint: "/",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
			{
				Device:     "PARTUUID=sdf3-uuid",
				Mountpoint: "/boot/efi",
				Fstype:     "fat32",
				Options:    []string{"defaults"},
			},
			{
				Device:     "LABEL=data-wrong-label",
				Mountpoint: "/data",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
		}

		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[0], ScanDir: nil,
		})).Return("/tmp/scandir", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[1], ScanDir: ptr.To("/tmp/scandir/boot/efi"),
		})).Return("/tmp/scandir/boot/efi", nil).Times(1)

		result, err := f.dmgr.mountWithFstab(partitions, entries)
		assert.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Equal(t, "/dev/sdf1", result[0].Name)
		assert.Equal(t, "/tmp/scandir", result[0].MountPoint)

		assert.Equal(t, "/dev/sdf3", result[1].Name)
		assert.Equal(t, "/tmp/scandir/boot/efi", result[1].MountPoint)

		assert.Equal(t, "/dev/sdg1", result[2].Name)
		assert.Equal(t, "", result[2].MountPoint)
	})

	t.Run("test pre-mounted", func(t *testing.T) {
		partitions := []*snapshot.PartitionInfo{
			{
				Name:       "/dev/sdf1",
				FsType:     "ext4",
				Uuid:       "sdf1-uuid",
				MountPoint: "/tmp/prescandir",
			},
			{
				Name:     "/dev/sdf3",
				FsType:   "fat32",
				PartUuid: "sdf3-uuid",
			},
			{
				Name:   "/dev/sdg1",
				FsType: "ext4",
				Label:  "data-label",
			},
		}
		entries := []resources.FstabEntry{
			{
				Device:     "UUID=sdf1-uuid",
				Mountpoint: "/",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
			{
				Device:     "PARTUUID=sdf3-uuid",
				Mountpoint: "/boot/efi",
				Fstype:     "fat32",
				Options:    []string{"defaults"},
			},
			{
				Device:     "LABEL=data-label",
				Mountpoint: "/data",
				Fstype:     "ext4",
				Options:    []string{"defaults"},
			},
		}

		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[0], ScanDir: nil,
		})).Return("/tmp/scandir", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[1], ScanDir: ptr.To("/tmp/scandir/boot/efi"),
		})).Return("/tmp/scandir/boot/efi", nil).Times(1)
		f.volumeMounter.EXPECT().MountP(gomock.Eq(&snapshot.MountPartitionDto{
			PartitionInfo: partitions[2], ScanDir: ptr.To("/tmp/scandir/data"),
		})).Return("/tmp/scandir/data", nil).Times(1)

		f.volumeMounter.EXPECT().UmountP(partitions[0]).Return(nil).Times(1)

		result, err := f.dmgr.mountWithFstab(partitions, entries)
		assert.NoError(t, err)
		assert.Len(t, result, 3)

		assert.Equal(t, "/dev/sdf1", result[0].Name)
		assert.Equal(t, "/tmp/scandir", result[0].MountPoint)

		assert.Equal(t, "/dev/sdf3", result[1].Name)
		assert.Equal(t, "/tmp/scandir/boot/efi", result[1].MountPoint)

		assert.Equal(t, "/dev/sdg1", result[2].Name)
		assert.Equal(t, "/tmp/scandir/data", result[2].MountPoint)
	})
}
