# vSphere Test Setup

# Access direct ESXi CLI

If you have direct access to the host, press Alt+F1 to open the log in page on the machine's physical console.
Provide credentials when prompted.

> Note: To return to the Direct Console User Interface press Alt-F2.

see https://kb.vmware.com/s/article/2004746

# VCSIM simulator

```bash
vcsim -l "127.0.0.1:8990"
```

```bash
export GOVC_INSECURE=true
export GOVC_URL=https://user:pass@127.0.0.1:8990/sdk GOVC_SIM_PID=51361

govc about               
Name:         VMware vCenter Server (govmomi simulator)
Vendor:       VMware, Inc.
Version:      6.5.0
Build:        5973321
OS type:      darwin-amd64
API type:     VirtualCenter
API version:  6.5
Product ID:   vpx
UUID:         dbed6e0c-bd88-4ef6-b594-21283e1c677f
```

**Use Simulator to test HA and DRS resources:**

```bash
govc cluster.change -drs-enabled -vsan-enabled -vsan-autoclaim DC0_C0 
```


you can use govc with an ESXi server by setting the credentials properly:

```bash
export GOVC_URL='https://root:password1!@192.168.56.102/sdk'
```

# Use PowerCLI with simulator

Install [PowerCLI for macOS](https://blogs.vmware.com/PowerCLI/2018/03/installing-powercli-10-0-0-macos.html)

```bash
brew cask install powershell
```

Connect ot the simulator:

```powershell
Set-PowerCLIConfiguration -InvalidCertificateAction:Ignore
Connect-VIServer -Server 127.0.0.1 -Port 8990 -User user -Password pass
```

Run individual commands:

```powershell
Foreach ($VMHost in Get-VMHost ) {
$ESXCli = Get-EsxCli -VMHost $VMHost; 
(Get-ESXCli).software.vib.list() | Select-Object @{N="VMHost";E={$VMHost}}, Name, AcceptanceLevel, CreationDate, ID, InstallDate, Status, Vendor, Version;}

# open esxi cli
$host = Get-VMHost | Select-Object -Index 0 
Get-EsxCli -VMHost $host

# display host services
Get-VMHost

Name                 ConnectionState PowerState NumCpu CpuUsageMhz CpuTotalMhz   MemoryUsageGB   MemoryTotalGB Version
----                 --------------- ---------- ------ ----------- -----------   -------------   ------------- -------
DC0_H0               Connected       PoweredOn       2          67        7182           1.371           4.000   6.5.0
DC0_C0_H0            Connected       PoweredOn       2          67        7182           1.371           4.000   6.5.0
DC0_C0_H1            Connected       PoweredOn       2          67        7182           1.371           4.000   6.5.0
DC0_C0_H2            Connected       PoweredOn       2          67        7182           1.371           4.000   6.5.0
192.168.56.102       Connected       PoweredOn       2         447        8016           1.054           3.973   6.7.0

# get host services
Get-VMHost DC0_H0 | Get-VMHostService

# start esxi host service
Get-VMHost DC0_H0 | Get-VMHostService | Where { $_.Key -eq "TSM-SSH" } | Start-VMHostService

# display packages for ESXi Host
Set-PowerCLIConfiguration -InvalidCertificateAction:Ignore
Connect-VIServer -Server 192.168.56.102 -Protocol https -User root -Password password1!
$ESXCli = Get-EsxCli -VMHost 192.168.56.102; 
($ESXCli).software.vib.list() | Select-Object @{N="VMHost";E={$VMHost}}, Name, AcceptanceLevel, CreationDate, ID, InstallDate, Status, Vendor, Version;}

# get esxi version
(Get-EsxCli).system.version.get()

Build   : Releasebuild-8169922
Patch   : 0
Product : VMware ESXi
Update  : 0
Version : 6.7.0

# get esxi modules
(Get-EsxCli).system.module.list()
IsEnabled IsLoaded Name
--------- -------- ----
true      true     vmkernel
true      true     chardevs
true      true     user
true      true     procfs

# display installed vibs
(Get-EsxCli).software.vib.list() 

AcceptanceLevel : VMwareCertified
CreationDate    : 2018-04-03
ID              : VMW_bootbank_ata-libata-92_3.00.9.2-16vmw.670.0.0.8169922
InstallDate     : 2020-07-16
Name            : ata-libata-92
Status          : 
Vendor          : VMW
Version         : 3.00.9.2-16vmw.670.0.0.8169922

AcceptanceLevel : VMwareCertified
CreationDate    : 2018-04-03
ID              : VMW_bootbank_ata-pata-amd_0.3.10-3vmw.670.0.0.8169922
InstallDate     : 2020-07-16
Name            : ata-pata-amd
Status          : 
Vendor          : VMW
Version         : 0.3.10-3vmw.670.0.0.8169922

# display network adapter
Get-VMHostNetworkAdapter

Name       Mac               DhcpEnabled IP              SubnetMask      DeviceName
----       ---               ----------- --              ----------      ----------
vmnic0     08:00:27:60:45:9d False                                           vmnic0
vmk0       08:00:27:60:45:9d True        192.168.56.102  255.255.255.0         vmk0

# display virtual switches
Get-VirtualSwitch
WARNING: The output of the command produced distributed virtual switch objects. This behavior is obsolete and may change in the future. To retrieve distributed switches, use Get-VDSwitch cmdlet in the VDS component. To retrieve standard switches, use -Standard.

Name                           NumPorts   Mtu   Notes
----                           --------   ---   -----
DVS0                           0          0     
vSwitch0                       1536       1500  
vSwitch0                       1536       1500  
vSwitch0                       1536       1500  
vSwitch0                       1536       1500  
vSwitch0                       2560       1500 
```

**To verify the network config**

Read [Check MTU size](https://www.sysadminstories.com/2019/12/check-esxi-mtu-settings-with-powercli.html). At ESXi level we need to check 3 settings: distributed virtual switch, physical network interfaces (vmnic used for uplinks) and vmkernel portgroups. To accomplish this we make use of two different PowerCLI cmdlets: Get-EsxCli and Get-VMHostNetworkAdapter

```powershell
# distributed virtual switch
(Get-EsxCli).network.vswitch.dvs.list()

# standard virtual switch
(Get-EsxCli).network.vswitch.standard.list()

BeaconEnabled    : false
BeaconInterval   : 1
BeaconRequiredBy : 
BeaconThreshold  : 3
CDPStatus        : listen
Class            : cswitch
ConfiguredPorts  : 128
MTU              : 1500
Name             : vSwitch0
NumPorts         : 2560
Portgroups       : {VM Network, Management Network}
Uplinks          : {vmnic0}
UsedPorts        : 4


# vmnics
(Get-EsxCli).network.nic.list.Invoke()        

AdminStatus : Up
Description : Intel Corporation 82540EM Gigabit Ethernet Controller
Driver      : e1000
Duplex      : Full
Link        : Up
LinkStatus  : Up
MACAddress  : 08:00:27:60:45:9d
MTU         : 1500
Name        : vmnic0
PCIDevice   : 0000:00:03.0
Speed       : 1000

# get network adapter
Get-VMHostNetworkAdapter | ConvertTo-JSON
```

