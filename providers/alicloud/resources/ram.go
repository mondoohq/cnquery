// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"fmt"
	"time"

	ramclient "github.com/alibabacloud-go/ram-20150501/v2/client"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
	"go.mondoo.com/mql/v13/types"
)

// ramParseTime parses an Alibaba Cloud RFC3339 timestamp string, returning nil when
// the pointer is nil or the value cannot be parsed.
func ramParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return nil
	}
	return &t
}

// ramStrVal safely dereferences a *string, returning "" for a nil pointer.
func ramStrVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ramBoolPtrAny returns the bool value, or nil when the pointer is nil, so dict
// fields represent an absent value as null rather than false.
func ramBoolPtrAny(b *bool) any {
	if b == nil {
		return nil
	}
	return *b
}

// ramInt32PtrAny returns the int value as int64, or nil when the pointer is nil.
func ramInt32PtrAny(i *int32) any {
	if i == nil {
		return nil
	}
	return int64(*i)
}

// ramStrPtrAny returns the string value, or nil when the pointer is nil.
func ramStrPtrAny(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// ramPolicyAttachmentDict builds a dict describing a policy attached to a user,
// group, or role. The three ...ForUser/ForGroup/ForRole responses share this
// field shape.
func ramPolicyAttachmentDict(policyName, policyType, defaultVersion, description, attachDate *string) map[string]any {
	return map[string]any{
		"policyName":     ramStrVal(policyName),
		"policyType":     ramStrVal(policyType),
		"defaultVersion": ramStrVal(defaultVersion),
		"description":    ramStrVal(description),
		"attachDate":     ramStrVal(attachDate),
	}
}

func (r *mqlAlicloudRam) id() (string, error) {
	return "alicloud.ram", nil
}

func (r *mqlAlicloudRam) users() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	var marker *string
	for {
		resp, err := client.ListUsers(&ramclient.ListUsersRequest{Marker: marker})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Users == nil {
			break
		}

		for _, u := range resp.Body.Users.User {
			if u == nil {
				continue
			}
			user, err := CreateResource(r.MqlRuntime, "alicloud.ram.user", map[string]*llx.RawData{
				"__id":        llx.StringData(ramStrVal(u.UserName)),
				"userId":      llx.StringDataPtr(u.UserId),
				"userName":    llx.StringDataPtr(u.UserName),
				"displayName": llx.StringDataPtr(u.DisplayName),
				"email":       llx.StringDataPtr(u.Email),
				"mobilePhone": llx.StringDataPtr(u.MobilePhone),
				"comments":    llx.StringDataPtr(u.Comments),
				"createDate":  llx.TimeDataPtr(ramParseTime(u.CreateDate)),
				"updateDate":  llx.TimeDataPtr(ramParseTime(u.UpdateDate)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, user)
		}

		if resp.Body.IsTruncated == nil || !*resp.Body.IsTruncated {
			break
		}
		marker = resp.Body.Marker
		if marker == nil || *marker == "" {
			break
		}
	}
	return res, nil
}

func (r *mqlAlicloudRam) groups() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	var marker *string
	for {
		resp, err := client.ListGroups(&ramclient.ListGroupsRequest{Marker: marker})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Groups == nil {
			break
		}

		for _, g := range resp.Body.Groups.Group {
			if g == nil {
				continue
			}
			group, err := CreateResource(r.MqlRuntime, "alicloud.ram.group", map[string]*llx.RawData{
				"__id":       llx.StringData(ramStrVal(g.GroupName)),
				"groupId":    llx.StringDataPtr(g.GroupId),
				"groupName":  llx.StringDataPtr(g.GroupName),
				"comments":   llx.StringDataPtr(g.Comments),
				"createDate": llx.TimeDataPtr(ramParseTime(g.CreateDate)),
				"updateDate": llx.TimeDataPtr(ramParseTime(g.UpdateDate)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, group)
		}

		if resp.Body.IsTruncated == nil || !*resp.Body.IsTruncated {
			break
		}
		marker = resp.Body.Marker
		if marker == nil || *marker == "" {
			break
		}
	}
	return res, nil
}

func (r *mqlAlicloudRam) roles() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	var marker *string
	for {
		resp, err := client.ListRoles(&ramclient.ListRolesRequest{Marker: marker})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Roles == nil {
			break
		}

		for _, role := range resp.Body.Roles.Role {
			if role == nil {
				continue
			}

			tags := map[string]any{}
			if role.Tags != nil {
				for _, t := range role.Tags.Tag {
					if t == nil || t.TagKey == nil {
						continue
					}
					tags[*t.TagKey] = ramStrVal(t.TagValue)
				}
			}

			mqlRole, err := CreateResource(r.MqlRuntime, "alicloud.ram.role", map[string]*llx.RawData{
				"__id":               llx.StringData(ramStrVal(role.RoleName)),
				"roleId":             llx.StringDataPtr(role.RoleId),
				"roleName":           llx.StringDataPtr(role.RoleName),
				"arn":                llx.StringDataPtr(role.Arn),
				"description":        llx.StringDataPtr(role.Description),
				"createDate":         llx.TimeDataPtr(ramParseTime(role.CreateDate)),
				"updateDate":         llx.TimeDataPtr(ramParseTime(role.UpdateDate)),
				"maxSessionDuration": llx.IntDataPtr(role.MaxSessionDuration),
				"tags":               llx.MapData(tags, types.String),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlRole)
		}

		if resp.Body.IsTruncated == nil || !*resp.Body.IsTruncated {
			break
		}
		marker = resp.Body.Marker
		if marker == nil || *marker == "" {
			break
		}
	}
	return res, nil
}

func (r *mqlAlicloudRam) policies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	var marker *string
	for {
		resp, err := client.ListPolicies(&ramclient.ListPoliciesRequest{Marker: marker})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Policies == nil {
			break
		}

		for _, p := range resp.Body.Policies.Policy {
			if p == nil {
				continue
			}

			tags := map[string]any{}
			if p.Tags != nil {
				for _, t := range p.Tags.Tag {
					if t == nil || t.TagKey == nil {
						continue
					}
					tags[*t.TagKey] = ramStrVal(t.TagValue)
				}
			}

			mqlPolicy, err := CreateResource(r.MqlRuntime, "alicloud.ram.policy", map[string]*llx.RawData{
				"__id":            llx.StringData(ramStrVal(p.PolicyType) + "/" + ramStrVal(p.PolicyName)),
				"policyName":      llx.StringDataPtr(p.PolicyName),
				"policyType":      llx.StringDataPtr(p.PolicyType),
				"description":     llx.StringDataPtr(p.Description),
				"defaultVersion":  llx.StringDataPtr(p.DefaultVersion),
				"attachmentCount": llx.IntDataPtr(p.AttachmentCount),
				"createDate":      llx.TimeDataPtr(ramParseTime(p.CreateDate)),
				"updateDate":      llx.TimeDataPtr(ramParseTime(p.UpdateDate)),
				"tags":            llx.MapData(tags, types.String),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, mqlPolicy)
		}

		if resp.Body.IsTruncated == nil || !*resp.Body.IsTruncated {
			break
		}
		marker = resp.Body.Marker
		if marker == nil || *marker == "" {
			break
		}
	}
	return res, nil
}

func (r *mqlAlicloudRam) passwordPolicy() (*mqlAlicloudRamPasswordPolicy, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetPasswordPolicy()
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.PasswordPolicy == nil {
		return nil, errors.New("alicloud: empty password policy returned by GetPasswordPolicy")
	}
	pp := resp.Body.PasswordPolicy

	res, err := CreateResource(r.MqlRuntime, "alicloud.ram.passwordPolicy", map[string]*llx.RawData{
		"__id":                       llx.StringData("alicloud.ram.passwordPolicy"),
		"minimumPasswordLength":      llx.IntDataPtr(pp.MinimumPasswordLength),
		"requireLowercaseCharacters": llx.BoolDataPtr(pp.RequireLowercaseCharacters),
		"requireUppercaseCharacters": llx.BoolDataPtr(pp.RequireUppercaseCharacters),
		"requireNumbers":             llx.BoolDataPtr(pp.RequireNumbers),
		"requireSymbols":             llx.BoolDataPtr(pp.RequireSymbols),
		"hardExpiry":                 llx.BoolDataPtr(pp.HardExpiry),
		"maxPasswordAge":             llx.IntDataPtr(pp.MaxPasswordAge),
		"passwordReusePrevention":    llx.IntDataPtr(pp.PasswordReusePrevention),
		"maxLoginAttempts":           llx.IntDataPtr(pp.MaxLoginAttemps),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlAlicloudRamPasswordPolicy), nil
}

func (r *mqlAlicloudRam) securityPreference() (any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	resp, err := client.GetSecurityPreference()
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.SecurityPreference == nil {
		return nil, nil
	}
	sp := resp.Body.SecurityPreference

	out := map[string]any{}
	if ak := sp.AccessKeyPreference; ak != nil {
		out["allowUserToManageAccessKeys"] = ramBoolPtrAny(ak.AllowUserToManageAccessKeys)
	}
	if mfa := sp.MFAPreference; mfa != nil {
		out["allowUserToManageMFADevices"] = ramBoolPtrAny(mfa.AllowUserToManageMFADevices)
	}
	if pk := sp.PublicKeyPreference; pk != nil {
		out["allowUserToManagePublicKeys"] = ramBoolPtrAny(pk.AllowUserToManagePublicKeys)
	}
	if lp := sp.LoginProfilePreference; lp != nil {
		out["allowUserToChangePassword"] = ramBoolPtrAny(lp.AllowUserToChangePassword)
		out["enableSaveMFATicket"] = ramBoolPtrAny(lp.EnableSaveMFATicket)
		out["loginNetworkMasks"] = ramStrPtrAny(lp.LoginNetworkMasks)
		out["loginSessionDuration"] = ramInt32PtrAny(lp.LoginSessionDuration)
	}
	return out, nil
}

// initAlicloudRamUser resolves a RAM user by userName via GetUser. It backs both
// direct lookups (alicloud.ram.user(userName: "x")) and cross-references from
// groups, reusing the cached instance when the user has already been listed.
func initAlicloudRamUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	rawName, ok := args["userName"]
	if !ok {
		return nil, nil, errors.New("alicloud.ram.user requires a userName to look up")
	}
	userName, ok := rawName.Value.(string)
	if !ok || userName == "" {
		return nil, nil, errors.New("alicloud.ram.user requires a userName to look up")
	}

	if x, ok := runtime.Resources.Get("alicloud.ram.user\x00" + userName); ok {
		return nil, x, nil
	}

	conn := runtime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetUser(&ramclient.GetUserRequest{UserName: &userName})
	if err != nil {
		return nil, nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.User == nil {
		return nil, nil, fmt.Errorf("alicloud.ram.user %q not found", userName)
	}
	u := resp.Body.User

	args = map[string]*llx.RawData{
		"__id":        llx.StringData(ramStrVal(u.UserName)),
		"userId":      llx.StringDataPtr(u.UserId),
		"userName":    llx.StringDataPtr(u.UserName),
		"displayName": llx.StringDataPtr(u.DisplayName),
		"email":       llx.StringDataPtr(u.Email),
		"mobilePhone": llx.StringDataPtr(u.MobilePhone),
		"comments":    llx.StringDataPtr(u.Comments),
		"createDate":  llx.TimeDataPtr(ramParseTime(u.CreateDate)),
		"updateDate":  llx.TimeDataPtr(ramParseTime(u.UpdateDate)),
	}
	return args, nil, nil
}

func (r *mqlAlicloudRamUser) id() (string, error) {
	return r.UserName.Data, nil
}

// getUser fetches the full user record via GetUser for fields the list response
// omits (lastLoginDate).
func (r *mqlAlicloudRamUser) getUser() (*ramclient.GetUserResponseBodyUser, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}
	userName := r.UserName.Data
	resp, err := client.GetUser(&ramclient.GetUserRequest{UserName: &userName})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.User == nil {
		return nil, fmt.Errorf("alicloud.ram.user %q not found", userName)
	}
	return resp.Body.User, nil
}

func (r *mqlAlicloudRamUser) lastLoginDate() (*time.Time, error) {
	u, err := r.getUser()
	if err != nil {
		log.Warn().Err(err).Str("user", r.UserName.Data).Msg("alicloud: could not fetch RAM user detail")
		return nil, nil
	}
	return ramParseTime(u.LastLoginDate), nil
}

func (r *mqlAlicloudRamUser) accessKeys() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	userName := r.UserName.Data
	resp, err := client.ListAccessKeys(&ramclient.ListAccessKeysRequest{UserName: &userName})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.AccessKeys == nil {
		return res, nil
	}
	for _, ak := range resp.Body.AccessKeys.AccessKey {
		if ak == nil {
			continue
		}
		mqlKey, err := CreateResource(r.MqlRuntime, "alicloud.ram.accessKey", map[string]*llx.RawData{
			"__id":        llx.StringData(userName + "/" + ramStrVal(ak.AccessKeyId)),
			"userName":    llx.StringData(userName),
			"accessKeyId": llx.StringDataPtr(ak.AccessKeyId),
			"status":      llx.StringDataPtr(ak.Status),
			"createDate":  llx.TimeDataPtr(ramParseTime(ak.CreateDate)),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlKey)
	}
	return res, nil
}

func (r *mqlAlicloudRamUser) groups() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	userName := r.UserName.Data
	resp, err := client.ListGroupsForUser(&ramclient.ListGroupsForUserRequest{UserName: &userName})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.Groups == nil {
		return res, nil
	}
	for _, g := range resp.Body.Groups.Group {
		if g == nil || g.GroupName == nil {
			continue
		}
		group, err := NewResource(r.MqlRuntime, "alicloud.ram.group", map[string]*llx.RawData{
			"groupName": llx.StringDataPtr(g.GroupName),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, group)
	}
	return res, nil
}

func (r *mqlAlicloudRamUser) policies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	userName := r.UserName.Data
	resp, err := client.ListPoliciesForUser(&ramclient.ListPoliciesForUserRequest{UserName: &userName})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.Policies == nil {
		return res, nil
	}
	for _, p := range resp.Body.Policies.Policy {
		if p == nil {
			continue
		}
		res = append(res, ramPolicyAttachmentDict(p.PolicyName, p.PolicyType, p.DefaultVersion, p.Description, p.AttachDate))
	}
	return res, nil
}

func (r *mqlAlicloudRamUser) mfaDevice() (any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	userName := r.UserName.Data
	resp, err := client.GetUserMFAInfo(&ramclient.GetUserMFAInfoRequest{UserName: &userName})
	if err != nil {
		// Users without a bound MFA device may surface an error here; treat as null.
		log.Debug().Err(err).Str("user", userName).Msg("alicloud: could not fetch RAM user MFA info")
		return nil, nil
	}
	if resp == nil || resp.Body == nil || resp.Body.MFADevice == nil || resp.Body.MFADevice.SerialNumber == nil {
		return nil, nil
	}
	dev := resp.Body.MFADevice
	return map[string]any{
		"serialNumber": ramStrVal(dev.SerialNumber),
		"type":         ramStrVal(dev.Type),
	}, nil
}

func (r *mqlAlicloudRamUser) loginProfile() (any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	userName := r.UserName.Data
	resp, err := client.GetLoginProfile(&ramclient.GetLoginProfileRequest{UserName: &userName})
	if err != nil {
		// Users with no console login profile return an EntityNotExist error; treat as null.
		log.Debug().Err(err).Str("user", userName).Msg("alicloud: could not fetch RAM user login profile")
		return nil, nil
	}
	if resp == nil || resp.Body == nil || resp.Body.LoginProfile == nil {
		return nil, nil
	}
	lp := resp.Body.LoginProfile
	out := map[string]any{
		"userName":              ramStrVal(lp.UserName),
		"mfaBindRequired":       ramBoolPtrAny(lp.MFABindRequired),
		"passwordResetRequired": ramBoolPtrAny(lp.PasswordResetRequired),
	}
	if t := ramParseTime(lp.CreateDate); t != nil {
		out["createDate"] = *t
	} else {
		out["createDate"] = nil
	}
	return out, nil
}

func (r *mqlAlicloudRamAccessKey) id() (string, error) {
	return r.UserName.Data + "/" + r.AccessKeyId.Data, nil
}

// initAlicloudRamGroup resolves a RAM group by groupName via GetGroup. It backs
// both direct lookups and cross-references from users, reusing the cached
// instance when the group has already been listed.
func initAlicloudRamGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	rawName, ok := args["groupName"]
	if !ok {
		return nil, nil, errors.New("alicloud.ram.group requires a groupName to look up")
	}
	groupName, ok := rawName.Value.(string)
	if !ok || groupName == "" {
		return nil, nil, errors.New("alicloud.ram.group requires a groupName to look up")
	}

	if x, ok := runtime.Resources.Get("alicloud.ram.group\x00" + groupName); ok {
		return nil, x, nil
	}

	conn := runtime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetGroup(&ramclient.GetGroupRequest{GroupName: &groupName})
	if err != nil {
		return nil, nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Group == nil {
		return nil, nil, fmt.Errorf("alicloud.ram.group %q not found", groupName)
	}
	g := resp.Body.Group

	args = map[string]*llx.RawData{
		"__id":       llx.StringData(ramStrVal(g.GroupName)),
		"groupId":    llx.StringDataPtr(g.GroupId),
		"groupName":  llx.StringDataPtr(g.GroupName),
		"comments":   llx.StringDataPtr(g.Comments),
		"createDate": llx.TimeDataPtr(ramParseTime(g.CreateDate)),
		"updateDate": llx.TimeDataPtr(ramParseTime(g.UpdateDate)),
	}
	return args, nil, nil
}

func (r *mqlAlicloudRamGroup) id() (string, error) {
	return r.GroupName.Data, nil
}

func (r *mqlAlicloudRamGroup) users() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	groupName := r.GroupName.Data
	res := []any{}
	var marker *string
	for {
		resp, err := client.ListUsersForGroup(&ramclient.ListUsersForGroupRequest{
			GroupName: &groupName,
			Marker:    marker,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Users == nil {
			break
		}

		for _, u := range resp.Body.Users.User {
			if u == nil || u.UserName == nil {
				continue
			}
			user, err := NewResource(r.MqlRuntime, "alicloud.ram.user", map[string]*llx.RawData{
				"userName": llx.StringDataPtr(u.UserName),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, user)
		}

		if resp.Body.IsTruncated == nil || !*resp.Body.IsTruncated {
			break
		}
		marker = resp.Body.Marker
		if marker == nil || *marker == "" {
			break
		}
	}
	return res, nil
}

func (r *mqlAlicloudRamGroup) policies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	groupName := r.GroupName.Data
	resp, err := client.ListPoliciesForGroup(&ramclient.ListPoliciesForGroupRequest{GroupName: &groupName})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.Policies == nil {
		return res, nil
	}
	for _, p := range resp.Body.Policies.Policy {
		if p == nil {
			continue
		}
		res = append(res, ramPolicyAttachmentDict(p.PolicyName, p.PolicyType, p.DefaultVersion, p.Description, p.AttachDate))
	}
	return res, nil
}

func (r *mqlAlicloudRamRole) id() (string, error) {
	return r.RoleName.Data, nil
}

func (r *mqlAlicloudRamRole) assumeRolePolicyDocument() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return "", err
	}

	roleName := r.RoleName.Data
	resp, err := client.GetRole(&ramclient.GetRoleRequest{RoleName: &roleName})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil || resp.Body.Role == nil {
		return "", nil
	}
	return ramStrVal(resp.Body.Role.AssumeRolePolicyDocument), nil
}

func (r *mqlAlicloudRamRole) policies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return nil, err
	}

	roleName := r.RoleName.Data
	resp, err := client.ListPoliciesForRole(&ramclient.ListPoliciesForRoleRequest{RoleName: &roleName})
	if err != nil {
		return nil, err
	}

	res := []any{}
	if resp == nil || resp.Body == nil || resp.Body.Policies == nil {
		return res, nil
	}
	for _, p := range resp.Body.Policies.Policy {
		if p == nil {
			continue
		}
		res = append(res, ramPolicyAttachmentDict(p.PolicyName, p.PolicyType, p.DefaultVersion, p.Description, p.AttachDate))
	}
	return res, nil
}

func (r *mqlAlicloudRamPolicy) id() (string, error) {
	return r.PolicyType.Data + "/" + r.PolicyName.Data, nil
}

func (r *mqlAlicloudRamPolicy) policyDocument() (string, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.RamClient()
	if err != nil {
		return "", err
	}

	policyName := r.PolicyName.Data
	policyType := r.PolicyType.Data
	resp, err := client.GetPolicy(&ramclient.GetPolicyRequest{
		PolicyName: &policyName,
		PolicyType: &policyType,
	})
	if err != nil {
		return "", err
	}
	if resp == nil || resp.Body == nil || resp.Body.DefaultPolicyVersion == nil {
		return "", nil
	}
	return ramStrVal(resp.Body.DefaultPolicyVersion.PolicyDocument), nil
}

func (r *mqlAlicloudRamPasswordPolicy) id() (string, error) {
	return "alicloud.ram.passwordPolicy", nil
}
