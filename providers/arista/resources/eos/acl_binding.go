// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package eos

import (
	"strings"
)

// AclBinding records one place an access-list is applied.
//
//	interface Ethernet1
//	   ip access-group UPLINK-IN in
//	!
//	management ssh
//	   ip access-group MGMT-ACCESS in
//
// Knowing which access-lists exist says nothing about what they protect. A
// list that is written but never applied filters nothing, and "is management
// access restricted to the admin network?" cannot be answered from the rules
// alone. The binding is the missing half.
type AclBinding struct {
	// Target is the kind of thing the list is applied to: "interface",
	// "managementSsh", "managementTelnet", "managementApi", or
	// "controlPlane".
	Target string
	// TargetName is the interface name for interface bindings, and empty
	// for the management services and the control plane, which are
	// singletons.
	TargetName string
	// Direction is "in" or "out". EOS defaults an omitted direction to
	// inbound.
	Direction string
	// Family is "ipv4" or "ipv6".
	Family string
	// AclName is the name of the applied access-list.
	AclName string
}

// aclBindingTargets maps a top-level configuration header to the binding
// target it represents. Interface blocks are handled separately since their
// header carries the interface name.
var aclBindingTargets = map[string]string{
	"management ssh":               "managementSsh",
	"management telnet":            "managementTelnet",
	"management api http-commands": "managementApi",
	"control-plane":                "controlPlane",
}

// ParseAclBindings finds every place an access-list is applied: on interfaces,
// on the management services, and on the control plane.
func ParseAclBindings(runningConfig string) []AclBinding {
	res := []AclBinding{}

	EachTopLevelBlock(runningConfig, func(header, body string) {
		target := ""
		targetName := ""

		if name, ok := strings.CutPrefix(header, "interface "); ok {
			target = "interface"
			targetName = strings.TrimSpace(name)
		} else if t, ok := aclBindingTargets[header]; ok {
			target = t
		} else {
			return
		}

		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// A negated binding removes the one above it, so it must not
			// register as a binding of its own.
			if strings.HasPrefix(line, "no ") {
				continue
			}

			family := ""
			var rest string
			if cut, ok := strings.CutPrefix(line, "ip access-group "); ok {
				family, rest = "ipv4", cut
			} else if cut, ok := strings.CutPrefix(line, "ipv6 access-group "); ok {
				family, rest = "ipv6", cut
			} else {
				continue
			}

			fields := strings.Fields(rest)
			if len(fields) == 0 {
				continue
			}
			binding := AclBinding{
				Target:     target,
				TargetName: targetName,
				Family:     family,
				AclName:    fields[0],
				// EOS applies an access-group inbound unless told otherwise.
				Direction: "in",
			}
			if len(fields) > 1 && (fields[1] == "in" || fields[1] == "out") {
				binding.Direction = fields[1]
			}
			res = append(res, binding)
		}
	})

	return res
}
