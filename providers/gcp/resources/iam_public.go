// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

const (
	publicMemberAllUsers              = "allUsers"
	publicMemberAllAuthenticatedUsers = "allAuthenticatedUsers"
)

func isPublicMember(s string) bool {
	return s == publicMemberAllUsers || s == publicMemberAllAuthenticatedUsers
}

// iamPolicyHasPublicMember returns true when any binding member is allUsers
// or allAuthenticatedUsers. The input is the .Data of an iamPolicy() TValue —
// each element is expected to be a *mqlGcpResourcemanagerBinding.
func iamPolicyHasPublicMember(bindings []any) (bool, error) {
	for _, raw := range bindings {
		binding, ok := raw.(*mqlGcpResourcemanagerBinding)
		if !ok || binding == nil {
			continue
		}
		members := binding.GetMembers()
		if members.Error != nil {
			return false, members.Error
		}
		for _, m := range members.Data {
			if s, ok := m.(string); ok && isPublicMember(s) {
				return true, nil
			}
		}
	}
	return false, nil
}
