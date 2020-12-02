package vmwareguestapi

// Guest Operations API
// https://docs.vmware.com/en/VMware-Cloud-on-AWS/services/com.vmware.vsphere.vmc-aws-manage-data-center-vms.doc/GUID-FE3B00A4-83F5-45AF-9B16-40008BC39E6F.html
// https://github.com/vmware/vsphere-guest-run/blob/master/vsphere_guest_run/vsphere.py
//
// Install vmware tools
// - [Installing VMware Tools in a Linux virtual machine using Red Hat Package Manager](https://kb.vmware.com/s/article/1018392)
// - [Installing VMware Tools in a Linux virtual machine using a Compiler](https://kb.vmware.com/s/article/1018414)
// - [Installing open-vm-tools](https://docs.vmware.com/en/VMware-Tools/11.0.0/com.vmware.vsphere.vmwaretools.doc/GUID-C48E1F14-240D-4DD1-8D4C-25B6EBE4BB0F.html)
// - [Using Open VM Tools](https://docs.vmware.com/en/VMware-Tools/11.1.0/com.vmware.vsphere.vmwaretools.doc/GUID-8B6EA5B7-453B-48AA-92E5-DB7F061341D1.html)
//
// ```powershell
// Set-PowerCLIConfiguration -InvalidCertificateAction:Ignore
// Connect-VIServer -Server 127.0.0.1 -Port 8990 -User user -Password pass
// $vm = Get-VM example-centos
// ```

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/guest"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"go.mondoo.io/mondoo/motor/transports/vmwareguestapi/toolbox"
	"go.mondoo.io/mondoo/motor/transports/vsphere"

	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor/transports"
	"go.mondoo.io/mondoo/motor/transports/ssh/cat"
)

func New(endpoint *transports.TransportConfig) (*Transport, error) {
	if endpoint.Backend != transports.TransportBackend_CONNECTION_VSPHERE_VM {
		return nil, errors.New("backend is not supported for vSphere transport")
	}

	// derive vsphere connection url from Transport Config
	vsphereUrl, err := vsphere.VSphereConnectionURL(endpoint.Host, endpoint.Port, endpoint.User, endpoint.Password)
	if err != nil {
		return nil, err
	}

	inventoryPath := endpoint.Options["inventoryPath"]
	guestuser := endpoint.Options["guestUser"]
	guestpassword := endpoint.Options["guestPassword"]

	// establish vsphere connection
	ctx := context.Background()
	client, err := govmomi.NewClient(ctx, vsphereUrl, true)
	if err != nil {
		return nil, err
	}

	// get vm via inventory path
	var vm *object.VirtualMachine
	finder := find.NewFinder(client.Client, true)
	vm, err = finder.VirtualMachine(context.Background(), inventoryPath)
	if err != nil {
		return nil, err
	}

	// initialize manager for processes and file
	o := guest.NewOperationsManager(client.Client, vm.Reference())
	pm, err := o.ProcessManager(ctx)
	if err != nil {
		return nil, err
	}

	fm, err := o.FileManager(ctx)
	if err != nil {
		return nil, err
	}

	// initialize vm authentication via password auth
	auth := &types.NamePasswordAuthentication{}
	auth.Username = guestuser
	auth.Password = guestpassword

	family := ""
	var props mo.VirtualMachine
	err = vm.Properties(context.Background(), vm.Reference(), []string{"guest.guestFamily"}, &props)
	if err != nil {
		return nil, err
	}

	if props.Guest != nil {
		family = props.Guest.GuestFamily
	}

	tb := &toolbox.Client{
		ProcessManager: pm,
		FileManager:    fm,
		Authentication: auth,
		GuestFamily:    types.VirtualMachineGuestOsFamily(family),
	}

	return &Transport{
		client: client,
		pm:     pm,
		fm:     fm,
		tb:     tb,
		auth:   auth,
		family: family,
	}, nil
}

type Transport struct {
	client *govmomi.Client
	pm     *guest.ProcessManager
	fm     *guest.FileManager
	tb     *toolbox.Client
	auth   types.BaseGuestAuthentication
	family string
	fs     afero.Fs
}

func (t *Transport) RunCommand(command string) (*transports.Command, error) {
	log.Debug().Str("command", command).Str("transport", "vmwareguest").Msg("run command")
	c := &Command{tb: t.tb}

	cmd, err := c.Exec(command)
	log.Debug().Err(err).Int("exit", cmd.ExitStatus).Msg("completed command")
	return cmd, err
}

func (t *Transport) FileInfo(path string) (transports.FileInfoDetails, error) {
	fs := t.FS()
	afs := &afero.Afero{Fs: fs}
	stat, err := afs.Stat(path)
	if err != nil {
		return transports.FileInfoDetails{}, err
	}

	uid := int64(-1)
	gid := int64(-1)

	// if t.Sudo != nil || t.UseScpFilesystem {
	// 	if stat, ok := stat.Sys().(*transports.FileInfo); ok {
	// 		uid = int64(stat.Uid)
	// 		gid = int64(stat.Gid)
	// 	}
	// } else {
	// 	if stat, ok := stat.Sys().(*rawsftp.FileStat); ok {
	// 		uid = int64(stat.UID)
	// 		gid = int64(stat.GID)
	// 	}
	// }
	mode := stat.Mode()

	return transports.FileInfoDetails{
		Mode: transports.FileModeDetails{mode},
		Size: stat.Size(),
		Uid:  uid,
		Gid:  gid,
	}, nil
}

func (t *Transport) FS() afero.Fs {
	// if we cached an instance already, return it
	if t.fs != nil {
		return t.fs
	}

	// even with PowerCli this is not working therefore we stick to catfs for now
	// Copy-VMGuestFile -VM $vm -GuestToLocal /etc/os-release -GuestUser root -GuestPassword vagrant -Destination os-release
	// Copy-VMGuestFile: 11/05/2020 18:38:57	Copy-VMGuestFile		A specified parameter was not correct:
	// t.fs = &VmwareGuestFs{
	// 	tb:            t.tb,
	// 	commandRunner: t,
	// }
	t.fs = cat.New(t)
	return t.fs
}

func (t *Transport) Close() {}

func (t *Transport) Capabilities() transports.Capabilities {
	return transports.Capabilities{
		transports.Capability_File,
		transports.Capability_RunCommand,
	}
}

func (t *Transport) Kind() transports.Kind {
	return transports.Kind_KIND_VIRTUAL_MACHINE
}

func (t *Transport) Runtime() string {
	return transports.RUNTIME_VSPHERE_VM
}
