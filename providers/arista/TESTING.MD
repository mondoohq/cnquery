# Arista EOS Provider Testing

## Overview

This document covers how to set up a vEOS lab device for mql provider development and validation. All commands are verified against vEOS. Some features are not available on vEOS — see [vEOS Limitations](#veos-limitations).

## Prerequisites

- Access to a vEOS instance — available via the [AWS Marketplace](https://aws.amazon.com/marketplace/search/results?searchTerms=arista+veos)
- SSH or console access

---

## System Basics

```
enable
configure terminal

! Hostname and domain
hostname arista-test-lab
ip domain-name lab.mondoo.com

! DNS
ip name-server 8.8.8.8
ip name-server 8.8.4.4

! Banners
banner motd
  ******************************************
  *   Mondoo mql Provider Test Lab         *
  *   Unauthorized access is prohibited    *
  ******************************************
EOF

banner login
  Authorized users only. All activity is monitored.
EOF

! Local users
username netadmin privilege 15 role network-admin secret netadmin123
username netops privilege 5 role network-operator secret netops123
username auditor privilege 1 secret auditor123

! SSH
management ssh
   idle-timeout 60
   authentication mode password
   no shutdown
!

! eAPI (used by mql HTTPS transport)
management api http-commands
   protocol https
   no shutdown
!

end
```

---

## Management Interface

```
configure terminal

interface Management1
   description MGMT
   ip address 192.168.100.10/24
!

ip route 0.0.0.0/0 192.168.100.1

end
```

---

## VRFs

```
configure terminal

vrf instance PROD
   description Production VRF
!
vrf instance DEV
   description Development VRF
!
vrf instance MGMT
   description Management VRF
!

end
```

---

## VLANs

```
configure terminal

! VLAN 101 - Basic, just a name
vlan 101
   name TEST-BASIC
!
! VLAN 102 - Suspended state
vlan 102
   name TEST-SUSPENDED
   state suspended
!
! VLAN 103 - Trunk group membership
vlan 103
   name TEST-TRUNK-GROUP
   trunk group CORE-PEERS
!
! VLAN 104 - Multiple trunk groups
vlan 104
   name TEST-MULTI-TRUNK
   trunk group CORE-PEERS
   trunk group MLAG-PEER
!
! VLAN 105 - PVLAN primary
vlan 105
   name TEST-PVLAN-PRIMARY
   private-vlan primary
!
! VLAN 106 - PVLAN community (associated to 105)
vlan 106
   name TEST-PVLAN-COMMUNITY
   private-vlan community
!
! VLAN 107 - PVLAN isolated (associated to 105)
vlan 107
   name TEST-PVLAN-ISOLATED
   private-vlan isolated
!
! VLAN 108 - No name (tests nil/default name handling)
vlan 108
!
! VLAN 109 - Name with underscores and hyphens
vlan 109
   name TEST_UNDERSCORE-AND-HYPHEN_99
!
! VLAN 110 - Suspended and in a trunk group (combined edge case)
vlan 110
   name TEST-SUSPENDED-TRUNK
   state suspended
   trunk group MLAG-PEER
!

end
```

| VLAN | Test case |
|------|-----------|
| 101  | Minimal config, name only |
| 102  | Suspended state |
| 103  | Single trunk group |
| 104  | Multiple trunk groups |
| 105  | PVLAN primary |
| 106  | PVLAN community |
| 107  | PVLAN isolated |
| 108  | No name (default/nil handling) |
| 109  | Name with special characters |
| 110  | Suspended + trunk group combined |

---

## Interfaces

### Loopbacks

```
configure terminal

interface Loopback0
   description ROUTER-ID
   ip address 10.0.0.1/32
!
interface Loopback1
   description BGP-ANYCAST
   ip address 10.0.0.2/32
!
interface Loopback100
   description VRF-PROD-LOOPBACK
   vrf PROD
   ip address 10.100.0.1/32
!

end
```

### SVIs

```
configure terminal

interface Vlan101
   description SVI-TEST-BASIC
   ip address 10.101.0.1/24
!
interface Vlan103
   description SVI-TRUNK-GROUP
   ip address 10.103.0.1/24
!
interface Vlan110
   description SVI-SUSPENDED-TRUNK
   ip address 10.110.0.1/24
   shutdown
!

end
```

---

## Routing

### Static Routes

```
configure terminal

ip route 192.168.0.0/16 10.1.1.2
ip route 172.16.0.0/12 10.1.1.2 name PRIVATE-RANGE
ip route 10.99.0.0/24 Null0 254 name BLACKHOLE

! VRF static route
ip route vrf PROD 0.0.0.0/0 10.100.1.2

end
```

### OSPF

```
configure terminal

router ospf 1
   router-id 10.0.0.1
   network 10.1.1.0/30 area 0.0.0.0
   network 10.0.0.0/32 area 0.0.0.0
   passive-interface default
   no passive-interface Ethernet1
!

end
```

### BGP

```
configure terminal

router bgp 65001
   router-id 10.0.0.1
   bgp log-neighbor-changes
   timers bgp 30 90

   ! eBGP peer
   neighbor 10.1.1.2 remote-as 65002
   neighbor 10.1.1.2 description EBGP-PEER-CORE
   neighbor 10.1.1.2 send-community

   ! iBGP peer
   neighbor 10.0.0.2 remote-as 65001
   neighbor 10.0.0.2 description IBGP-PEER-LOOPBACK
   neighbor 10.0.0.2 update-source Loopback0

   ! Shutdown peer (tests disabled neighbor fetching)
   neighbor 10.1.1.6 remote-as 65003
   neighbor 10.1.1.6 description EBGP-PEER-SHUTDOWN
   neighbor 10.1.1.6 shutdown

   ! Address family
   address-family ipv4
      neighbor 10.1.1.2 activate
      neighbor 10.0.0.2 activate
      network 10.0.0.1/32
      network 10.0.0.2/32
!

   ! VRF BGP
   vrf PROD
      router-id 10.100.0.1
      neighbor 10.100.1.2 remote-as 65099
      neighbor 10.100.1.2 description PROD-VRF-PEER
      address-family ipv4
         neighbor 10.100.1.2 activate
         network 10.100.0.1/32
!

end
```

---

## ACLs

```
configure terminal

! Standard ACL
ip access-list standard MGMT-ACCESS
   10 permit 192.168.100.0/24
   20 deny any log
!

! Extended ACL
ip access-list extended UPLINK-IN
   10 permit tcp 10.0.0.0/8 any eq 443
   20 permit tcp 10.0.0.0/8 any eq 22
   30 permit icmp any any
   40 deny ip any any log
!

! IPv6 ACL
ipv6 access-list V6-MGMT
   10 permit ipv6 fc00::/7 any
   20 deny ipv6 any any log
!

end
```

---

## AAA / TACACS+

```
configure terminal

! TACACS+ server (unreachable in lab, config present for fetch testing)
tacacs-server host 192.168.100.200 key lab-tacacs-key
tacacs-server timeout 5

! AAA
aaa group server tacacs+ TACACS-GROUP
   server 192.168.100.200
!
aaa authentication login default group TACACS-GROUP local
aaa authentication enable default group TACACS-GROUP local
aaa authorization exec default group TACACS-GROUP local
aaa accounting exec default start-stop group TACACS-GROUP

end
```

---

## Monitoring

### NTP

```
configure terminal

ntp server 0.pool.ntp.org prefer
ntp server 1.pool.ntp.org
ntp server 192.168.100.1 local-interface Management1

end
```

### Syslog

```
configure terminal

logging on
logging host 192.168.100.201
logging host 192.168.100.202 514
logging trap informational
logging source-interface Management1
logging facility local6

end
```

### SNMP

```
configure terminal

snmp-server community public ro
snmp-server community private rw
snmp-server location "Mondoo Lab - vEOS Test Instance"
snmp-server contact "lab@mondoo.com"
snmp-server host 192.168.100.203 version 2c public
snmp-server enable traps

end
```

---

## Verify

```
show running-config
show vlan brief
show interfaces status
show ip route summary
show bgp summary
show ip ospf neighbor
show ip access-lists
show ntp status
show snmp
show users
```

---

## Teardown

```
configure terminal

no vlan 101-110
no router bgp 65001
no router ospf 1
no ip access-list standard MGMT-ACCESS
no ip access-list extended UPLINK-IN
no ipv6 access-list V6-MGMT
no vrf instance PROD
no vrf instance DEV
no vrf instance MGMT
no username netops
no username auditor
no ntp server 0.pool.ntp.org
no ntp server 1.pool.ntp.org
no logging host 192.168.100.201
no logging host 192.168.100.202
no snmp-server community public
no snmp-server community private

end
```

---

## vEOS Limitations

The following are supported on physical EOS hardware but unavailable on vEOS:

| Feature | Command | Workaround |
|---------|---------|------------|
| Per-VLAN STP priority | `spanning-tree vlan X priority Y` | Not applicable |
| IGMP snooping | `ip igmp snooping vlan X` | Not applicable |
| Per-VLAN MAC aging | `mac address-table aging-time X vlan Y` | Not applicable |
