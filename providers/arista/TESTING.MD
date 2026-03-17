# Arista EOS Provider Testing

## Overview

This document covers how to set up a test EOS device for mql provider development and validation. Commands are verified against a vEOS instance in AWS. Some features are not available on vEOS hardware — see [Hardware Limitations](#hardware-limitations).

## Test VLAN Setup

The following VLANs cover the core attribute combinations used in provider fetch logic.

```
enable
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

### What each VLAN covers

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

### Verify

```
show vlan brief
```

## Hardware Limitations

The following commands are verified on physical EOS hardware but fail on cEOS/vEOS lab instances due to platform limitations. Include them when testing against real hardware.

### Spanning Tree (per-VLAN priority)

Only applies when STP mode is PVST or RSTP. Verify mode first:

```
show spanning-tree summary
```

If supported:

```
spanning-tree vlan 101 priority 4096
spanning-tree vlan 103 priority 8192
spanning-tree vlan 104 priority 32768
```

Not applicable in MSTP mode — configure per-instance instead:

```
spanning-tree mst 0 priority 4096
```

### IGMP Snooping

```
no ip igmp snooping vlan 102
ip igmp snooping vlan 103 querier
ip igmp snooping vlan 103 querier address 10.0.103.1
```

### MAC Address Table Aging

```
mac address-table aging-time 60 vlan 109
mac address-table aging-time 0 vlan 110
```
