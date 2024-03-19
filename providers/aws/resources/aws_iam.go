// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/util/convert"
	"go.mondoo.com/cnquery/v10/providers/aws/connection"
	"go.mondoo.com/cnquery/v10/providers/aws/resources/awsiam"
	"go.mondoo.com/cnquery/v10/providers/aws/resources/awspolicy"
	"go.mondoo.com/cnquery/v10/types"
)

func (a *mqlAwsIam) id() (string, error) {
	return "aws.iam", nil
}

func (a *mqlAwsIam) serverCertificates() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()
	var marker *string
	res := []interface{}{}

	for {
		certsResp, err := svc.ListServerCertificates(ctx, &iam.ListServerCertificatesInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		if len(certsResp.ServerCertificateMetadataList) > 0 {
			certs, err := convert.JsonToDictSlice(certsResp.ServerCertificateMetadataList)
			if err != nil {
				return nil, err
			}
			res = append(res, certs)
		}
		if !certsResp.IsTruncated {
			break
		}
		marker = certsResp.Marker
	}
	return res, nil
}

func (a *mqlAwsIam) credentialReport() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	var data []byte
	// try to fetch the credential report
	// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetCredentialReport.html
	// 410 - ReportExpired
	// 404 - ReportInProgress
	// 410 - ReportNotPresent
	// 500 - ServiceFailure
	_, err := svc.GetCredentialReport(ctx, &iam.GetCredentialReportInput{})
	if err != nil {
		var awsFailErr *iamtypes.ServiceFailureException
		if errors.As(err, &awsFailErr) {
			return nil, errors.Wrap(err, "could not gather aws iam credential report")
		}

		// if we have an error and it is not 500 we generate a report
		var ae smithy.APIError
		if errors.As(err, &ae) {
			if ae.ErrorCode() == "ReportNotPresent" {
				// generate a new report
				gresp, err := svc.GenerateCredentialReport(ctx, &iam.GenerateCredentialReportInput{})
				if err != nil {
					return nil, err
				}

				if gresp.State == iamtypes.ReportStateTypeStarted || gresp.State == iamtypes.ReportStateTypeInprogress {
					// we need to wait
				} else if gresp.State == iamtypes.ReportStateTypeComplete {
					// we do not need do do anything
				} else {
					// unsupported report state
					return nil, fmt.Errorf("aws iam credential report state is not supported: %s", gresp.State)
				}
			}
		}
	}

	// loop as long as the response is 404 since this means the report is still in progress
	rresp, err := svc.GetCredentialReport(ctx, &iam.GetCredentialReportInput{})
	var ae smithy.APIError
	if errors.As(err, &ae) {
		for ae.ErrorCode() == "NoSuchEntity" || ae.ErrorCode() == "ReportInProgress" {
			rresp, err = svc.GetCredentialReport(ctx, &iam.GetCredentialReportInput{})
			if err == nil {
				break
			}

			log.Error().Err(err).Msgf("resp %v, err: %v", rresp, err)

			if errors.As(err, &ae) {
				if ae.ErrorCode() != "NoSuchEntity" && ae.ErrorCode() != "ReportInProgress" {
					return nil, errors.Wrap(err, "could not gather aws iam credential report")
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if rresp == nil {
		return nil, errors.Wrap(err, "could not gather aws iam credential report")
	}

	data = rresp.Content

	// parse csv output
	entries, err := awsiam.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Wrap(err, "could not parse aws iam credential report")
	}

	res := []interface{}{}
	for i := range entries {
		userEntry, err := CreateResource(a.MqlRuntime, "aws.iam.usercredentialreportentry",
			map[string]*llx.RawData{"properties": llx.MapData(entries[i], types.String)},
		)
		if err != nil {
			return nil, err
		}
		res = append(res, userEntry)
	}
	return res, nil
}

func (a *mqlAwsIam) accountPasswordPolicy() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	resp, err := svc.GetAccountPasswordPolicy(ctx, &iam.GetAccountPasswordPolicyInput{})
	var notFoundErr *iamtypes.NoSuchEntityException
	if err != nil {
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "could not gather aws iam account-password-policy")
	}

	res := ParsePasswordPolicy(resp.PasswordPolicy)

	return res, nil
}

func ParsePasswordPolicy(passwordPolicy *iamtypes.PasswordPolicy) map[string]interface{} {
	res := map[string]interface{}{}

	if passwordPolicy != nil {
		prp := int64(0)
		if passwordPolicy.PasswordReusePrevention != nil {
			prp = int64(*passwordPolicy.PasswordReusePrevention)
		}
		mpa := int64(0)
		if passwordPolicy.MaxPasswordAge != nil {
			mpa = int64(*passwordPolicy.MaxPasswordAge)
		}
		mpl := int64(0)
		if passwordPolicy.MinimumPasswordLength != nil {
			mpl = int64(*passwordPolicy.MinimumPasswordLength)
		}

		res["AllowUsersToChangePassword"] = passwordPolicy.AllowUsersToChangePassword
		res["RequireUppercaseCharacters"] = passwordPolicy.RequireUppercaseCharacters
		res["RequireSymbols"] = passwordPolicy.RequireSymbols
		res["ExpirePasswords"] = passwordPolicy.ExpirePasswords
		res["PasswordReusePrevention"] = strconv.FormatInt(prp, 10)
		res["RequireLowercaseCharacters"] = passwordPolicy.RequireLowercaseCharacters
		res["MaxPasswordAge"] = strconv.FormatInt(mpa, 10)
		res["HardExpiry"] = convert.ToBool(passwordPolicy.HardExpiry)
		res["RequireNumbers"] = passwordPolicy.RequireNumbers
		res["MinimumPasswordLength"] = strconv.FormatInt(mpl, 10)
	}
	return res
}

func (a *mqlAwsIam) accountSummary() (map[string]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	resp, err := svc.GetAccountSummary(ctx, &iam.GetAccountSummaryInput{})
	if err != nil {
		return nil, errors.Wrap(err, "could not gather aws iam account-summary")
	}

	// convert result to MQL
	res := map[string]interface{}{}
	for k := range resp.SummaryMap {
		res[k] = int64(resp.SummaryMap[k])
	}

	return res, nil
}

func (a *mqlAwsIam) users() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	var marker *string
	res := []interface{}{}
	for {
		usersResp, err := svc.ListUsers(ctx, &iam.ListUsersInput{Marker: marker})
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws iam users")
		}
		for i := range usersResp.Users {
			usr := usersResp.Users[i]

			mqlAwsIamUser, err := a.createIamUser(&usr)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamUser)
		}
		if !usersResp.IsTruncated {
			break
		}
		marker = usersResp.Marker
	}
	return res, nil
}

func iamTagsToMap(tags []iamtypes.Tag) map[string]interface{} {
	var tagsMap map[string]interface{}

	if len(tags) > 0 {
		tagsMap := map[string]interface{}{}
		for i := range tags {
			tag := tags[i]
			tagsMap[convert.ToString(tag.Key)] = convert.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (a *mqlAwsIam) createIamUser(usr *iamtypes.User) (plugin.Resource, error) {
	if usr == nil {
		return nil, errors.New("no iam user provided")
	}

	return CreateResource(a.MqlRuntime, "aws.iam.user",
		map[string]*llx.RawData{
			"arn":              llx.StringDataPtr(usr.Arn),
			"id":               llx.StringDataPtr(usr.UserId),
			"name":             llx.StringDataPtr(usr.UserName),
			"createDate":       llx.TimeDataPtr(usr.CreateDate),
			"passwordLastUsed": llx.TimeDataPtr(usr.PasswordLastUsed),
			"tags":             llx.MapData(iamTagsToMap(usr.Tags), types.String),
		},
	)
}

func (a *mqlAwsIam) virtualMfaDevices() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	devicesResp, err := svc.ListVirtualMFADevices(ctx, &iam.ListVirtualMFADevicesInput{})
	if err != nil {
		log.Error().Err(err).Msg("cannot gather virtual mfa devices info")
		a.VirtualMfaDevices = plugin.TValue[[]interface{}]{Error: err, State: plugin.StateIsSet}
		return nil, nil
	}

	// note: adding pagination to this call results in Throttling: Rate exceeded error
	res := []interface{}{}
	for i := range devicesResp.VirtualMFADevices {
		device := devicesResp.VirtualMFADevices[i]

		var mqlAwsIamUser plugin.Resource
		args := map[string]*llx.RawData{
			"serialNumber": llx.StringDataPtr(device.SerialNumber),
			"enableDate":   llx.TimeDataPtr(device.EnableDate),
		}

		usr := device.User
		if usr != nil {
			mqlAwsIamUser, err = NewResource(a.MqlRuntime, "aws.iam.user", map[string]*llx.RawData{
				"arn":  llx.StringDataPtr(usr.Arn),
				"name": llx.StringDataPtr(usr.UserName),
			})
			if err == nil {
				args["user"] = llx.ResourceData(mqlAwsIamUser, "aws.iam.user")
			}
		}

		if usr == nil || err != nil {
			args["user"] = llx.NilData
		}

		mqlAwsIamMfaDevice, err := CreateResource(a.MqlRuntime, "aws.iam.virtualmfadevice", args)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlAwsIamMfaDevice)
	}

	return res, nil
}

func (a *mqlAwsIam) mqlPolicies(policies []iamtypes.Policy) ([]interface{}, error) {
	res := []interface{}{}
	for i := range policies {
		policy := policies[i]
		// NOTE: here we have all the information about the policy already
		// therefore we pass the information in, so that MQL does not have to resolve it again
		mqlAwsIamPolicy, err := CreateResource(a.MqlRuntime, "aws.iam.policy",
			map[string]*llx.RawData{
				"arn":             llx.StringDataPtr(policy.Arn),
				"id":              llx.StringDataPtr(policy.PolicyId),
				"name":            llx.StringDataPtr(policy.PolicyName),
				"description":     llx.StringDataPtr(policy.Description),
				"isAttachable":    llx.BoolData(policy.IsAttachable),
				"attachmentCount": llx.IntDataDefault(policy.AttachmentCount, 0),
				"createDate":      llx.TimeDataPtr(policy.CreateDate),
				"updateDate":      llx.TimeDataPtr(policy.UpdateDate),
			})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAwsIamPolicy)
	}
	return res, nil
}

func (a *mqlAwsIam) attachedPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	res := []interface{}{}
	var marker *string
	for {
		policiesResp, err := svc.ListPolicies(ctx, &iam.ListPoliciesInput{
			// setting only attached ensures we only fetch policies attached to a user, group, or role
			OnlyAttached: true,
			Marker:       marker,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws iam policies")
		}

		policies, err := a.mqlPolicies(policiesResp.Policies)
		if err != nil {
			return nil, err
		}
		res = append(res, policies...)

		if !policiesResp.IsTruncated {
			break
		}
		marker = policiesResp.Marker
	}

	return res, nil
}

func (a *mqlAwsIam) policies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	res := []interface{}{}
	var marker *string
	for {
		policiesResp, err := svc.ListPolicies(ctx, &iam.ListPoliciesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, errors.Wrap(err, "could not gather aws iam policies")
		}

		policies, err := a.mqlPolicies(policiesResp.Policies)
		if err != nil {
			return nil, err
		}
		res = append(res, policies...)

		if !policiesResp.IsTruncated {
			break
		}
		marker = policiesResp.Marker
	}

	return res, nil
}

func (a *mqlAwsIam) roles() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)
	svc := conn.Iam("")
	ctx := context.Background()

	res := []interface{}{}
	var marker *string
	for {
		rolesResp, err := svc.ListRoles(ctx, &iam.ListRolesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		// Added Trust relationship policy attached to each role
		for _, role := range rolesResp.Roles {
			policyOutput, err := svc.GetRole(ctx, &iam.GetRoleInput{RoleName: role.RoleName})
			var policyDocumentMap map[string]interface{}
			if err == nil && policyOutput.Role != nil && policyOutput.Role.AssumeRolePolicyDocument != nil {
				policyDocument := *policyOutput.Role.AssumeRolePolicyDocument
				decodedPolicyDocument, decodeErr := url.QueryUnescape(policyDocument)
				if decodeErr == nil {
					json.Unmarshal([]byte(decodedPolicyDocument), &policyDocumentMap)
				}
			}

			mqlAwsIamRole, err := CreateResource(a.MqlRuntime, "aws.iam.role",
				map[string]*llx.RawData{
					"arn":                      llx.StringDataPtr(role.Arn),
					"id":                       llx.StringDataPtr(role.RoleId),
					"name":                     llx.StringDataPtr(role.RoleName),
					"description":              llx.StringDataPtr(role.Description),
					"tags":                     llx.MapData(iamTagsToMap(role.Tags), types.String),
					"createDate":               llx.TimeDataPtr(role.CreateDate),
					"assumeRolePolicyDocument": llx.MapData(policyDocumentMap, types.Any),
				})
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamRole)
		}

		if !rolesResp.IsTruncated {
			break
		}
		marker = rolesResp.Marker
	}

	return res, nil
}

func (a *mqlAwsIam) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	res := []interface{}{}
	var marker *string
	for {
		groupsResp, err := svc.ListGroups(ctx, &iam.ListGroupsInput{
			Marker: marker,
		})
		if err != nil {
			return nil, err
		}

		for i := range groupsResp.Groups {
			grp := groupsResp.Groups[i]

			mqlAwsIamGroup, err := NewResource(a.MqlRuntime, "aws.iam.group",
				map[string]*llx.RawData{
					"arn":  llx.StringDataPtr(grp.Arn),
					"name": llx.StringDataPtr(grp.GroupName),
				})
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamGroup)
		}

		if !groupsResp.IsTruncated {
			break
		}
		marker = groupsResp.Marker
	}

	return res, nil
}

func (p *mqlAwsIamUsercredentialreportentry) id() (string, error) {
	props := p.Properties.Data

	userid := props["arn"].(string)

	return "aws/iam/credentialreport/" + userid, nil
}

func (p *mqlAwsIamUsercredentialreportentry) arn() (string, error) {
	props := p.Properties.Data

	if props == nil {
		return "", errors.New("could not read the credentials report")
	}

	val, ok := props["arn"].(string)
	if !ok {
		return "", errors.New("arn is not a string value")
	}

	return val, nil
}

func (p *mqlAwsIamUsercredentialreportentry) getBoolValue(key string) (bool, error) {
	props := p.Properties.Data

	if props == nil {
		return false, errors.New("could not read the credentials report")
	}

	val, ok := props[key].(string)
	if !ok {
		return false, errors.New(key + " is not a string value")
	}

	// handle "N/A" and "not_supported" value
	// some accounts do not support specific values eg. root_account does not support password_enabled
	if val == "not_supported" {
		return false, nil
	}

	return strconv.ParseBool(val)
}

func (p *mqlAwsIamUsercredentialreportentry) getStringValue(key string) (string, error) {
	props := p.Properties.Data

	if props == nil {
		return "", errors.New("could not read the credentials report")
	}

	val, ok := props[key].(string)
	if !ok {
		return "", errors.New(key + " is not a string value")
	}

	return val, nil
}

func (p *mqlAwsIamUsercredentialreportentry) getTimeValue(key string) (*time.Time, error) {
	props := p.Properties.Data

	if props == nil {
		log.Info().Msgf("could not retrieve key")
		return nil, errors.New("could not read the credentials report")
	}

	val, ok := props[key].(string)
	if !ok {
		log.Info().Msgf("key is not a string")
		return nil, errors.New(key + " is not a valid string value")
	}

	// handle "N/A" and "not_supported" value
	// some accounts do not support specific values eg. root_account does not support password_last_changed or password_next_rotation
	if val == "N/A" || val == "not_supported" || val == "no_information" {
		return &llx.NeverFutureTime, nil
	}

	// parse iso 8601  "2020-07-15T14:52:00+00:00"
	format := "2006-01-02T15:04:05-07:00"
	parsed, err := time.Parse(format, val)
	if err != nil {
		log.Error().Err(err).Msg("could not parse the time")
		return nil, errors.New("failed to parse time: " + err.Error())
	}

	return &parsed, nil
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey1Active() (bool, error) {
	return p.getBoolValue("access_key_1_active")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey1LastRotated() (*time.Time, error) {
	return p.getTimeValue("access_key_1_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey1LastUsedDate() (*time.Time, error) {
	return p.getTimeValue("access_key_1_last_used_date")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey1LastUsedRegion() (string, error) {
	return p.getStringValue("access_key_1_last_used_region")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey1LastUsedService() (string, error) {
	return p.getStringValue("access_key_1_last_used_service")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey2Active() (bool, error) {
	return p.getBoolValue("access_key_2_active")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey2LastRotated() (*time.Time, error) {
	return p.getTimeValue("access_key_2_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey2LastUsedDate() (*time.Time, error) {
	return p.getTimeValue("access_key_2_last_used_date")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey2LastUsedRegion() (string, error) {
	return p.getStringValue("access_key_2_last_used_region")
}

func (p *mqlAwsIamUsercredentialreportentry) accessKey2LastUsedService() (string, error) {
	return p.getStringValue("access_key_2_last_used_service")
}

// TODO: update keys

func (p *mqlAwsIamUsercredentialreportentry) cert1Active() (bool, error) {
	return p.getBoolValue("cert_1_active")
}

func (p *mqlAwsIamUsercredentialreportentry) cert1LastRotated() (*time.Time, error) {
	return p.getTimeValue("cert_1_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) cert2Active() (bool, error) {
	return p.getBoolValue("cert_2_active")
}

func (p *mqlAwsIamUsercredentialreportentry) cert2LastRotated() (*time.Time, error) {
	return p.getTimeValue("cert_2_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) mfaActive() (bool, error) {
	return p.getBoolValue("mfa_active")
}

func (p *mqlAwsIamUsercredentialreportentry) passwordEnabled() (bool, error) {
	return p.getBoolValue("password_enabled")
}

func (p *mqlAwsIamUsercredentialreportentry) passwordLastChanged() (*time.Time, error) {
	return p.getTimeValue("password_last_changed")
}

func (p *mqlAwsIamUsercredentialreportentry) passwordLastUsed() (*time.Time, error) {
	return p.getTimeValue("password_last_used")
}

func (p *mqlAwsIamUsercredentialreportentry) passwordNextRotation() (*time.Time, error) {
	return p.getTimeValue("password_next_rotation")
}

func (a *mqlAwsIamUsercredentialreportentry) user() (*mqlAwsIamUser, error) {
	props := a.Properties.Data

	if props == nil {
		log.Info().Msgf("could not retrieve key")
		return nil, errors.New("could not read the credentials report")
	}
	// handle special case for the root account since that user does not exist
	if props["user"] == "<root_account>" {
		return nil, errors.New("root user does not exist")
	}

	mqlUser, err := NewResource(a.MqlRuntime, "aws.iam.user",
		map[string]*llx.RawData{
			"name": llx.StringData(props["user"].(string)),
		},
	)
	if err != nil {
		return nil, err
	}

	return mqlUser.(*mqlAwsIamUser), nil
}

func (a *mqlAwsIamUsercredentialreportentry) userCreationTime() (*time.Time, error) {
	return a.getTimeValue("user_creation_time")
}

func (a *mqlAwsIamVirtualmfadevice) id() (string, error) {
	return a.SerialNumber.Data, nil
}

func initAwsIamUser(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}

	if args["name"] == nil {
		return nil, nil, errors.New("name required to fetch aws iam user")
	}
	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	if args["name"] != nil {
		if usr, ok := args["name"].Value.(string); ok {
			username := usr
			resp, err := svc.GetUser(ctx, &iam.GetUserInput{
				UserName: &username,
			})
			if err != nil {
				return nil, nil, err
			}

			usr := resp.User
			args["arn"] = llx.StringDataPtr(usr.Arn)
			args["id"] = llx.StringDataPtr(usr.UserId)
			args["name"] = llx.StringDataPtr(usr.UserName)
			args["createDate"] = llx.TimeDataPtr(usr.CreateDate)
			args["passwordLastUsed"] = llx.TimeDataPtr(usr.PasswordLastUsed)
			args["tags"] = llx.MapData(iamTagsToMap(usr.Tags), types.String)

			return args, nil, nil
		}
	}

	return args, nil, nil
}

func (a *mqlAwsIamUser) id() (string, error) {
	if a.Arn.Error != nil {
		return "", a.Arn.Error
	}
	return a.Arn.Data, nil
}

func (a *mqlAwsIamUser) accessKeys() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	username := a.Name.Data

	var marker *string
	res := []interface{}{}
	for {
		keysResp, err := svc.ListAccessKeys(ctx, &iam.ListAccessKeysInput{
			UserName: &username,
			Marker:   marker,
		})
		if err != nil {
			return nil, err
		}
		metadata, err := convert.JsonToDictSlice(keysResp.AccessKeyMetadata)
		if err != nil {
			return nil, err
		}
		res = append(res, metadata)
		if !keysResp.IsTruncated {
			break
		}
		marker = keysResp.Marker
	}

	return res, nil
}

func (a *mqlAwsIamUser) policies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	username := a.Name.Data

	var marker *string
	res := []interface{}{}
	for {
		userPolicies, err := svc.ListUserPolicies(ctx, &iam.ListUserPoliciesInput{
			UserName: &username,
			Marker:   marker,
		})
		if err != nil {
			return nil, err
		}

		for i := range userPolicies.PolicyNames {
			res = append(res, userPolicies.PolicyNames[i])
		}
		if !userPolicies.IsTruncated {
			break
		}
		marker = userPolicies.Marker
	}

	return res, nil
}

func (a *mqlAwsIamUser) attachedPolicies() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	username := a.Name.Data

	var marker *string
	res := []interface{}{}
	for {
		userAttachedPolicies, err := svc.ListAttachedUserPolicies(ctx, &iam.ListAttachedUserPoliciesInput{
			Marker:   marker,
			UserName: &username,
		})
		if err != nil {
			return nil, err
		}

		for i := range userAttachedPolicies.AttachedPolicies {
			attachedPolicy := userAttachedPolicies.AttachedPolicies[i]

			mqlAwsIamPolicy, err := CreateResource(a.MqlRuntime, "aws.iam.policy",
				map[string]*llx.RawData{"arn": llx.StringDataPtr(attachedPolicy.PolicyArn)},
			)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamPolicy)
		}
		if !userAttachedPolicies.IsTruncated {
			break
		}
		marker = userAttachedPolicies.Marker
	}

	return res, nil
}

func (a *mqlAwsIamPolicy) id() (string, error) {
	if a == nil {
		return "", nil
	}
	return a.Arn.Data, nil
}

func (a *mqlAwsIamPolicy) loadPolicy(arn string) (*iamtypes.Policy, error) {
	// c, ok := a.Cache.Load("_policy")
	// if ok {
	// 	log.Info().Msg("use policy from cache")
	// 	return c.Data.(*types.Policy), nil
	// }

	// if its not in the cache, fetch it
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	policy, err := svc.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	// cache the data
	// a.Cache.Store("_policy", &resources.CacheEntry{Data: policy.Policy})
	return policy.Policy, nil
}

func (a *mqlAwsIamPolicy) name() (string, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil {
		return "", err
	}
	return convert.ToString(policy.PolicyName), nil
}

func (a *mqlAwsIamPolicy) description() (string, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil {
		return "", err
	}
	return convert.ToString(policy.Description), nil
}

func (a *mqlAwsIamPolicy) isAttachable() (bool, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil {
		return false, err
	}
	return policy.IsAttachable, nil
}

func (a *mqlAwsIamPolicy) attachmentCount() (int64, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil || policy.AttachmentCount == nil {
		return int64(0), err
	}
	return int64(*policy.AttachmentCount), nil
}

func (a *mqlAwsIamPolicy) createDate() (*time.Time, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil {
		return nil, err
	}
	return policy.CreateDate, nil
}

func (a *mqlAwsIamPolicy) updateDate() (*time.Time, error) {
	arn := a.Arn.Data

	policy, err := a.loadPolicy(arn)
	if err != nil {
		return nil, err
	}
	return policy.UpdateDate, nil
}

func (a *mqlAwsIamPolicy) scope() (string, error) {
	arnVal := a.Arn.Data

	parsed, err := arn.Parse(arnVal)
	if err != nil {
		return "", err
	}

	if parsed.AccountID == "aws" {
		return "aws", nil
	}

	return "local", nil
}

type attachedEntities struct {
	PolicyGroups []iamtypes.PolicyGroup
	PolicyRoles  []iamtypes.PolicyRole
	PolicyUsers  []iamtypes.PolicyUser
}

func (a *mqlAwsIamPolicy) listAttachedEntities(arn string) (attachedEntities, error) {
	// c, ok := a.Cache.Load("_attachedentities")
	// if ok {
	// 	log.Debug().Msg("use attached entities from cache")
	// 	return c.Data.(attachedEntities), nil
	// }
	var res attachedEntities

	// if its not in the cache, fetch it
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	var marker *string
	for {
		entities, err := svc.ListEntitiesForPolicy(ctx, &iam.ListEntitiesForPolicyInput{
			Marker:    marker,
			PolicyArn: &arn,
		})
		if err != nil {
			return res, err
		}

		if len(entities.PolicyGroups) > 0 {
			res.PolicyGroups = append(res.PolicyGroups, entities.PolicyGroups...)
		}

		if len(entities.PolicyRoles) > 0 {
			res.PolicyRoles = append(res.PolicyRoles, entities.PolicyRoles...)
		}

		if len(entities.PolicyUsers) > 0 {
			res.PolicyUsers = append(res.PolicyUsers, entities.PolicyUsers...)
		}

		if entities.IsTruncated == false {
			break
		}
		marker = entities.Marker
	}

	// cache the data
	// a.Cache.Store("_attachedentities", &resources.CacheEntry{Data: res})
	return res, nil
}

func (a *mqlAwsIamPolicy) attachedUsers() ([]interface{}, error) {
	arn := a.Arn.Data

	entities, err := a.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}
	res := []interface{}{}
	for i := range entities.PolicyUsers {
		usr := entities.PolicyUsers[i]
		mqlUser, err := NewResource(a.MqlRuntime, "aws.iam.user",
			map[string]*llx.RawData{
				"name": llx.StringDataPtr(usr.UserName),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (a *mqlAwsIamPolicy) attachedRoles() ([]interface{}, error) {
	arn := a.Arn.Data
	entities, err := a.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range entities.PolicyRoles {
		role := entities.PolicyRoles[i]

		mqlUser, err := NewResource(a.MqlRuntime, "aws.iam.role",
			map[string]*llx.RawData{"name": llx.StringDataPtr(role.RoleName)},
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (a *mqlAwsIamPolicy) attachedGroups() ([]interface{}, error) {
	arn := a.Arn.Data

	entities, err := a.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range entities.PolicyGroups {
		group := entities.PolicyGroups[i]

		mqlUser, err := NewResource(a.MqlRuntime, "aws.iam.group",
			map[string]*llx.RawData{
				"name": llx.StringDataPtr(group.GroupName),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (a *mqlAwsIamPolicy) defaultVersion() (*mqlAwsIamPolicyversion, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	arn := a.Arn.Data

	policyVersions, err := svc.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	for i := range policyVersions.Versions {
		policyversion := policyVersions.Versions[i]
		if policyversion.IsDefaultVersion {
			mqlAwsIamPolicyVersion, err := CreateResource(a.MqlRuntime, "aws.iam.policyversion",
				map[string]*llx.RawData{
					"arn":              llx.StringData(arn),
					"versionId":        llx.StringDataPtr(policyversion.VersionId),
					"isDefaultVersion": llx.BoolData(policyversion.IsDefaultVersion),
					"createDate":       llx.TimeDataPtr(policyversion.CreateDate),
				})
			if err != nil {
				return nil, err
			}
			return mqlAwsIamPolicyVersion.(*mqlAwsIamPolicyversion), nil
		}
	}
	return nil, errors.New("unable to find default policy version")
}

func (a *mqlAwsIamPolicy) versions() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	arn := a.Arn.Data

	policyVersions, err := svc.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policyVersions.Versions {
		policyversion := policyVersions.Versions[i]

		mqlAwsIamPolicyVersion, err := CreateResource(a.MqlRuntime, "aws.iam.policyversion",
			map[string]*llx.RawData{
				"arn":              llx.StringData(arn),
				"versionId":        llx.StringDataPtr(policyversion.VersionId),
				"isDefaultVersion": llx.BoolData(policyversion.IsDefaultVersion),
				"createDate":       llx.TimeDataPtr(policyversion.CreateDate),
			})
		if err != nil {
			return nil, err
		}

		res = append(res, mqlAwsIamPolicyVersion)
	}

	return res, nil
}

func (a *mqlAwsIamPolicyversion) id() (string, error) {
	arn := a.Arn.Data

	versionid := a.VersionId.Data

	return arn + "/" + versionid, nil
}

func (a *mqlAwsIamPolicyversion) document() (interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	arn := a.Arn.Data
	versionid := a.VersionId.Data

	policyVersion, err := svc.GetPolicyVersion(ctx, &iam.GetPolicyVersionInput{
		PolicyArn: &arn,
		VersionId: &versionid,
	})
	if err != nil {
		return "", err
	}

	if policyVersion.PolicyVersion.Document == nil {
		return "", errors.New("could not retrieve the policy document")
	}
	decodedValue, err := url.QueryUnescape(*policyVersion.PolicyVersion.Document)
	if err != nil {
		return "", err
	}
	policyDoc := awspolicy.IamPolicyDocument{}
	err = json.Unmarshal([]byte(decodedValue), &policyDoc)
	if err != nil {
		return "", err
	}
	dict, err := convert.JsonToDict(policyDoc)
	if err != nil {
		return "", err
	}
	return dict, nil
}

func initAwsIamRole(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}

	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws iam role")
	}

	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	if args["name"] != nil {
		rolename := args["name"].Value.(string)
		resp, err := svc.GetRole(ctx, &iam.GetRoleInput{
			RoleName: &rolename,
		})
		if err != nil {
			return nil, nil, err
		}

		role := resp.Role

		var policyDocumentMap map[string]interface{}
		if role != nil && role.AssumeRolePolicyDocument != nil {
			policyDocument := *role.AssumeRolePolicyDocument
			decodedPolicyDocument, decodeErr := url.QueryUnescape(policyDocument)
			if decodeErr == nil {
				json.Unmarshal([]byte(decodedPolicyDocument), &policyDocumentMap)
			}
		}

		args["arn"] = llx.StringDataPtr(role.Arn)
		args["id"] = llx.StringDataPtr(role.RoleId)
		args["name"] = llx.StringDataPtr(role.RoleName)
		args["description"] = llx.StringDataPtr(role.Description)
		args["tags"] = llx.MapData(iamTagsToMap(role.Tags), types.String)
		args["createDate"] = llx.TimeDataPtr(role.CreateDate)
		args["assumeRolePolicyDocument"] = llx.MapData(policyDocumentMap, types.Any)
		return args, nil, nil
	}

	return args, nil, nil
}

func (a *mqlAwsIamRole) id() (string, error) {
	return a.Arn.Data, nil
}

func initAwsIamGroup(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 2 {
		return args, nil, nil
	}
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["arn"] = llx.StringData(ids.arn)
		}
	}
	if args["arn"] == nil && args["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws iam group")
	}

	conn := runtime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	if args["name"] != nil {
		groupname := args["name"].Value.(string)
		resp, err := svc.GetGroup(ctx, &iam.GetGroupInput{
			GroupName: &groupname,
		})
		if err != nil {
			return nil, nil, err
		}
		usernames := []interface{}{}
		for _, user := range resp.Users {
			usernames = append(usernames, convert.ToString(user.UserName))
		}

		grp := resp.Group
		args["arn"] = llx.StringDataPtr(grp.Arn)
		args["id"] = llx.StringDataPtr(grp.GroupId)
		args["name"] = llx.StringDataPtr(grp.GroupName)
		args["createDate"] = llx.TimeDataPtr(grp.CreateDate)
		args["usernames"] = llx.ArrayData(usernames, types.String)
		return args, nil, nil
	}

	return args, nil, nil
}

func (a *mqlAwsIamGroup) id() (string, error) {
	return a.Arn.Data, nil
}

func (a *mqlAwsIamUser) groups() ([]interface{}, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()

	username := a.Name.Data

	var marker *string
	res := []interface{}{}
	for {
		userGroups, err := svc.ListGroupsForUser(ctx, &iam.ListGroupsForUserInput{
			UserName: &username,
			Marker:   marker,
		})
		if err != nil {
			return nil, err
		}

		for i := range userGroups.Groups {
			res = append(res, convert.ToString(userGroups.Groups[i].GroupName))
		}
		if !userGroups.IsTruncated {
			break
		}
		marker = userGroups.Marker
	}

	return res, nil
}

func (a *mqlAwsIamUser) loginProfile() (*mqlAwsIamLoginProfile, error) {
	conn := a.MqlRuntime.Connection.(*connection.AwsConnection)

	svc := conn.Iam("")
	ctx := context.Background()
	name := a.Name.Data

	profile, err := svc.GetLoginProfile(ctx, &iam.GetLoginProfileInput{
		UserName: &name,
	})

	var ae smithy.APIError
	if errors.As(err, &ae) {
		if ae.ErrorCode() == "NoSuchEntity" {
			a.LoginProfile.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}

	date := profile.LoginProfile.CreateDate
	if date == nil {
		return nil, errors.New("login profile doesn't have a createDate")
	}

	o, err := CreateResource(a.MqlRuntime, "aws.iam.loginProfile", map[string]*llx.RawData{
		"createdAt": llx.TimeData(*date),
	})
	if err != nil {
		return nil, err
	}
	return o.(*mqlAwsIamLoginProfile), nil
}

func (a *mqlAwsIamLoginProfile) init() (string, error) {
	date := a.CreatedAt.Data
	if date == nil {
		return "", nil
	}
	// Note: the precision of AWS logins is in seconds. Current AWS docs don't
	// specify a precision. Using seconds is reasonable.
	return strconv.FormatInt(date.Unix(), 10), nil
}
