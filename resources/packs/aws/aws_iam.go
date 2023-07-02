package aws

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"errors"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/smithy-go"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/aws/awsiam"
	"go.mondoo.com/cnquery/resources/packs/aws/awspolicy"
	"go.mondoo.com/cnquery/resources/packs/core"
)

func (p *mqlAwsIam) id() (string, error) {
	return "aws.iam", nil
}

func (c *mqlAwsIam) GetServerCertificates() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()
	var marker *string
	res := []interface{}{}

	for {
		certsResp, err := svc.ListServerCertificates(ctx, &iam.ListServerCertificatesInput{Marker: marker})
		if err != nil {
			return nil, err
		}
		if len(certsResp.ServerCertificateMetadataList) > 0 {
			certs, err := core.JsonToDictSlice(certsResp.ServerCertificateMetadataList)
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

func (c *mqlAwsIam) GetCredentialReport() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	var data []byte
	// try to fetch the credential report
	// https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetCredentialReport.html
	// 410 - ReportExpired
	// 404 - ReportInProgress
	// 410 - ReportNotPresent
	// 500 - ServiceFailure
	_, err = svc.GetCredentialReport(ctx, &iam.GetCredentialReportInput{})
	if err != nil {
		var awsFailErr *types.ServiceFailureException
		if errors.As(err, &awsFailErr) {
			return nil, errors.Join(err, errors.New("could not gather aws iam credential report"))
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

				if gresp.State == types.ReportStateTypeStarted || gresp.State == types.ReportStateTypeInprogress {
					// we need to wait
				} else if gresp.State == types.ReportStateTypeComplete {
					// we do not neet do do anything
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
					return nil, errors.Join(err, errors.New("could not gather aws iam credential report"))
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if rresp == nil {
		return nil, errors.Join(err, errors.New("could not gather aws iam credential report"))
	}

	data = rresp.Content

	// parse csv output
	entries, err := awsiam.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, errors.Join(err, errors.New("could not parse aws iam credential report"))
	}

	res := []interface{}{}
	for i := range entries {
		userEntry, err := c.MotorRuntime.CreateResource("aws.iam.usercredentialreportentry",
			"properties", entries[i],
		)
		if err != nil {
			return nil, err
		}
		res = append(res, userEntry)
	}
	return res, nil
}

func (c *mqlAwsIam) GetAccountPasswordPolicy() (map[string]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	resp, err := svc.GetAccountPasswordPolicy(ctx, &iam.GetAccountPasswordPolicyInput{})
	var notFoundErr *types.NoSuchEntityException
	if err != nil {
		if errors.As(err, &notFoundErr) {
			return nil, nil
		}
		return nil, errors.Join(err, errors.New("could not gather aws iam account-password-policy"))
	}

	res := ParsePasswordPolicy(resp.PasswordPolicy)

	return res, nil
}

func ParsePasswordPolicy(passwordPolicy *types.PasswordPolicy) map[string]interface{} {
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
		res["HardExpiry"] = core.ToBool(passwordPolicy.HardExpiry)
		res["RequireNumbers"] = passwordPolicy.RequireNumbers
		res["MinimumPasswordLength"] = strconv.FormatInt(mpl, 10)
	}
	return res
}

func (c *mqlAwsIam) GetAccountSummary() (map[string]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	resp, err := svc.GetAccountSummary(ctx, &iam.GetAccountSummaryInput{})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws iam account-summary"))
	}

	// convert result to MQL
	res := map[string]interface{}{}
	for k := range resp.SummaryMap {
		res[k] = int64(resp.SummaryMap[k])
	}

	return res, nil
}

func (c *mqlAwsIam) GetUsers() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	var marker *string
	res := []interface{}{}
	for {
		usersResp, err := svc.ListUsers(ctx, &iam.ListUsersInput{Marker: marker})
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather aws iam users"))
		}
		for i := range usersResp.Users {
			usr := usersResp.Users[i]

			mqlAwsIamUser, err := c.createIamUser(&usr)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamUser)
		}
		if usersResp.IsTruncated == false {
			break
		}
		marker = usersResp.Marker
	}
	return res, nil
}

func iamTagsToMap(tags []types.Tag) map[string]interface{} {
	var tagsMap map[string]interface{}

	if len(tags) > 0 {
		tagsMap := map[string]interface{}{}
		for i := range tags {
			tag := tags[i]
			tagsMap[core.ToString(tag.Key)] = core.ToString(tag.Value)
		}
	}

	return tagsMap
}

func (c *mqlAwsIam) createIamUser(usr *types.User) (resources.ResourceType, error) {
	if usr == nil {
		return nil, errors.New("no iam user provided")
	}

	return c.MotorRuntime.CreateResource("aws.iam.user",
		"arn", core.ToString(usr.Arn),
		"id", core.ToString(usr.UserId),
		"name", core.ToString(usr.UserName),
		"createDate", usr.CreateDate,
		"passwordLastUsed", usr.PasswordLastUsed,
		"tags", iamTagsToMap(usr.Tags),
	)
}

func (c *mqlAwsIam) GetVirtualMfaDevices() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	devicesResp, err := svc.ListVirtualMFADevices(ctx, &iam.ListVirtualMFADevicesInput{})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not gather aws iam virtual-mfa-devices"))
	}

	// note: adding pagination to this call results in Throttling: Rate exceeded error
	res := []interface{}{}
	for i := range devicesResp.VirtualMFADevices {
		device := devicesResp.VirtualMFADevices[i]

		var mqlAwsIamUser resources.ResourceType
		usr := device.User
		if usr != nil {
			mqlAwsIamUser, err = c.createIamUser(usr)
			if err != nil {
				return nil, err
			}
		}

		mqlAwsIamMfaDevice, err := c.MotorRuntime.CreateResource("aws.iam.virtualmfadevice",
			"serialNumber", core.ToString(device.SerialNumber),
			"enableDate", device.EnableDate,
			"user", mqlAwsIamUser,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlAwsIamMfaDevice)
	}

	return res, nil
}

func (c *mqlAwsIam) mqlPolicies(policies []types.Policy) ([]interface{}, error) {
	res := []interface{}{}
	for i := range policies {
		policy := policies[i]
		// NOTE: here we have all the information about the policy already
		// therefore we pass the information in, so that MQL does not have to resolve it again
		mqlAwsIamPolicy, err := c.MotorRuntime.CreateResource("aws.iam.policy",
			"arn", core.ToString(policy.Arn),
			"id", core.ToString(policy.PolicyId),
			"name", core.ToString(policy.PolicyName),
			"description", core.ToString(policy.Description),
			"isAttachable", policy.IsAttachable,
			"attachmentCount", core.ToInt64From32(policy.AttachmentCount),
			"createDate", policy.CreateDate,
			"updateDate", policy.UpdateDate,
		)
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAwsIamPolicy)
	}
	return res, nil
}

func (c *mqlAwsIam) GetAttachedPolicies() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
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
			return nil, errors.Join(err, errors.New("could not gather aws iam policies"))
		}

		policies, err := c.mqlPolicies(policiesResp.Policies)
		if err != nil {
			return nil, err
		}
		res = append(res, policies...)

		if policiesResp.IsTruncated == false {
			break
		}
		marker = policiesResp.Marker
	}

	return res, nil
}

func (c *mqlAwsIam) GetPolicies() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	res := []interface{}{}
	var marker *string
	for {
		policiesResp, err := svc.ListPolicies(ctx, &iam.ListPoliciesInput{
			Marker: marker,
		})
		if err != nil {
			return nil, errors.Join(err, errors.New("could not gather aws iam policies"))
		}

		policies, err := c.mqlPolicies(policiesResp.Policies)
		if err != nil {
			return nil, err
		}
		res = append(res, policies...)

		if policiesResp.IsTruncated == false {
			break
		}
		marker = policiesResp.Marker
	}

	return res, nil
}

func (c *mqlAwsIam) GetRoles() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
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

		for i := range rolesResp.Roles {
			role := rolesResp.Roles[i]

			mqlAwsIamRole, err := c.MotorRuntime.CreateResource("aws.iam.role",
				"arn", core.ToString(role.Arn),
				"id", core.ToString(role.RoleId),
				"name", core.ToString(role.RoleName),
				"description", core.ToString(role.Description),
				"tags", iamTagsToMap(role.Tags),
				"createDate", role.CreateDate,
			)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamRole)
		}

		if rolesResp.IsTruncated == false {
			break
		}
		marker = rolesResp.Marker
	}

	return res, nil
}

func (c *mqlAwsIam) GetGroups() ([]interface{}, error) {
	provider, err := awsProvider(c.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
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

			mqlAwsIamGroup, err := c.MotorRuntime.CreateResource("aws.iam.group",
				"arn", core.ToString(grp.Arn),
				"name", core.ToString(grp.GroupName),
			)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamGroup)
		}

		if groupsResp.IsTruncated == false {
			break
		}
		marker = groupsResp.Marker
	}

	return res, nil
}

func (p *mqlAwsIamUsercredentialreportentry) id() (string, error) {
	props, err := p.Properties()
	if err != nil {
		return "", err
	}

	userid := props["arn"].(string)

	return "aws/iam/credentialreport/" + userid, nil
}

func (p *mqlAwsIamUsercredentialreportentry) GetArn() (string, error) {
	props, err := p.Properties()
	if err != nil {
		return "", err
	}

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
	props, err := p.Properties()
	if err != nil {
		return false, err
	}

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
	props, err := p.Properties()
	if err != nil {
		return "", err
	}

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
	props, err := p.Properties()
	if err != nil {
		return nil, err
	}

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

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey1Active() (bool, error) {
	return p.getBoolValue("access_key_1_active")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey1LastRotated() (*time.Time, error) {
	return p.getTimeValue("access_key_1_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey1LastUsedDate() (*time.Time, error) {
	return p.getTimeValue("access_key_1_last_used_date")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey1LastUsedRegion() (string, error) {
	return p.getStringValue("access_key_1_last_used_region")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey1LastUsedService() (string, error) {
	return p.getStringValue("access_key_1_last_used_service")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey2Active() (bool, error) {
	return p.getBoolValue("access_key_2_active")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey2LastRotated() (*time.Time, error) {
	return p.getTimeValue("access_key_2_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey2LastUsedDate() (*time.Time, error) {
	return p.getTimeValue("access_key_2_last_used_date")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey2LastUsedRegion() (string, error) {
	return p.getStringValue("access_key_2_last_used_region")
}

func (p *mqlAwsIamUsercredentialreportentry) GetAccessKey2LastUsedService() (string, error) {
	return p.getStringValue("access_key_2_last_used_service")
}

// TODO: update keys

func (p *mqlAwsIamUsercredentialreportentry) GetCert1Active() (bool, error) {
	return p.getBoolValue("cert_1_active")
}

func (p *mqlAwsIamUsercredentialreportentry) GetCert1LastRotated() (*time.Time, error) {
	return p.getTimeValue("cert_1_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) GetCert2Active() (bool, error) {
	return p.getBoolValue("cert_2_active")
}

func (p *mqlAwsIamUsercredentialreportentry) GetCert2LastRotated() (*time.Time, error) {
	return p.getTimeValue("cert_2_last_rotated")
}

func (p *mqlAwsIamUsercredentialreportentry) GetMfaActive() (bool, error) {
	return p.getBoolValue("mfa_active")
}

func (p *mqlAwsIamUsercredentialreportentry) GetPasswordEnabled() (bool, error) {
	return p.getBoolValue("password_enabled")
}

func (p *mqlAwsIamUsercredentialreportentry) GetPasswordLastChanged() (*time.Time, error) {
	return p.getTimeValue("password_last_changed")
}

func (p *mqlAwsIamUsercredentialreportentry) GetPasswordLastUsed() (*time.Time, error) {
	return p.getTimeValue("password_last_used")
}

func (p *mqlAwsIamUsercredentialreportentry) GetPasswordNextRotation() (*time.Time, error) {
	return p.getTimeValue("password_next_rotation")
}

func (p *mqlAwsIamUsercredentialreportentry) GetUser() (interface{}, error) {
	props, err := p.Properties()
	if err != nil {
		return nil, err
	}

	if props == nil {
		log.Info().Msgf("could not retrieve key")
		return nil, errors.New("could not read the credentials report")
	}

	// handle special case for the root account since that user does not exist
	if props["user"] == "<root_account>" {
		return nil, nil
	}

	mqlUser, err := p.MotorRuntime.CreateResource("aws.iam.user",
		"name", props["user"],
	)
	if err != nil {
		return nil, err
	}

	return mqlUser, nil
}

func (p *mqlAwsIamUsercredentialreportentry) GetUserCreationTime() (*time.Time, error) {
	return p.getTimeValue("user_creation_time")
}

func (u *mqlAwsIamVirtualmfadevice) id() (string, error) {
	return u.SerialNumber()
}

func (p *mqlAwsIamUser) init(args *resources.Args) (*resources.Args, AwsIamUser, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}
	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}

	if (*args)["arn"] == nil && (*args)["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws iam user")
	}

	// TODO: avoid reloading if all groups have been loaded already
	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	// TODO: handle arn and extract the name
	// if (*args)["arn"] != nil { }

	if (*args)["name"] != nil {
		username := (*args)["name"].(string)
		resp, err := svc.GetUser(ctx, &iam.GetUserInput{
			UserName: &username,
		})
		if err != nil {
			return nil, nil, err
		}

		usr := resp.User
		(*args)["arn"] = core.ToString(usr.Arn)
		(*args)["id"] = core.ToString(usr.UserId)
		(*args)["name"] = core.ToString(usr.UserName)
		(*args)["createDate"] = usr.CreateDate
		(*args)["passwordLastUsed"] = usr.PasswordLastUsed
		(*args)["tags"] = iamTagsToMap(usr.Tags)

		return args, nil, nil
	}

	// if the package cannot be found, we init it as an empty package
	(*args)["arn"] = ""
	(*args)["id"] = ""
	(*args)["name"] = ""
	(*args)["createDate"] = &time.Time{}
	(*args)["passwordLastUsed"] = &time.Time{}
	(*args)["tags"] = make(map[string]interface{})

	return args, nil, nil
}

func (u *mqlAwsIamUser) id() (string, error) {
	return u.Arn()
}

func (u *mqlAwsIamUser) GetAccessKeys() ([]interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	username, err := u.Name()
	if err != nil {
		return nil, err
	}

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
		metadata, err := core.JsonToDictSlice(keysResp.AccessKeyMetadata)
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

func (u *mqlAwsIamUser) GetPolicies() ([]interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	username, err := u.Name()
	if err != nil {
		return nil, err
	}

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
		if userPolicies.IsTruncated == false {
			break
		}
		marker = userPolicies.Marker
	}

	return res, nil
}

func (u *mqlAwsIamUser) GetAttachedPolicies() ([]interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	username, err := u.Name()
	if err != nil {
		return nil, err
	}

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

			mqlAwsIamPolicy, err := u.MotorRuntime.CreateResource("aws.iam.policy",
				"arn", core.ToString(attachedPolicy.PolicyArn),
			)
			if err != nil {
				return nil, err
			}

			res = append(res, mqlAwsIamPolicy)
		}
		if userAttachedPolicies.IsTruncated == false {
			break
		}
		marker = userAttachedPolicies.Marker
	}

	return res, nil
}

func (u *mqlAwsIamPolicy) id() (string, error) {
	return u.Arn()
}

func (u *mqlAwsIamPolicy) loadPolicy(arn string) (*types.Policy, error) {
	c, ok := u.Cache.Load("_policy")
	if ok {
		log.Info().Msg("use policy from cache")
		return c.Data.(*types.Policy), nil
	}

	// if its not in the cache, fetch it
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	policy, err := svc.GetPolicy(ctx, &iam.GetPolicyInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	// cache the data
	u.Cache.Store("_policy", &resources.CacheEntry{Data: policy.Policy})
	return policy.Policy, nil
}

func (u *mqlAwsIamPolicy) GetId() (string, error) {
	arn, err := u.Arn()
	if err != nil {
		return "", err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return "", err
	}
	return core.ToString(policy.PolicyId), nil
}

func (u *mqlAwsIamPolicy) GetName() (string, error) {
	arn, err := u.Arn()
	if err != nil {
		return "", err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return "", err
	}
	return core.ToString(policy.PolicyName), nil
}

func (u *mqlAwsIamPolicy) GetDescription() (string, error) {
	arn, err := u.Arn()
	if err != nil {
		return "", err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return "", err
	}
	return core.ToString(policy.Description), nil
}

func (u *mqlAwsIamPolicy) GetIsAttachable() (bool, error) {
	arn, err := u.Arn()
	if err != nil {
		return false, err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return false, err
	}
	return policy.IsAttachable, nil
}

func (u *mqlAwsIamPolicy) GetAttachmentCount() (int64, error) {
	arn, err := u.Arn()
	if err != nil {
		return int64(0), err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return int64(0), err
	}
	return core.ToInt64From32(policy.AttachmentCount), nil
}

func (u *mqlAwsIamPolicy) GetCreateDate() (*time.Time, error) {
	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return nil, err
	}
	return policy.CreateDate, nil
}

func (u *mqlAwsIamPolicy) GetUpdateDate() (*time.Time, error) {
	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}

	policy, err := u.loadPolicy(arn)
	if err != nil {
		return nil, err
	}
	return policy.UpdateDate, nil
}

func (u *mqlAwsIamPolicy) GetScope() (string, error) {
	arnValue, err := u.Arn()
	if err != nil {
		return "", err
	}

	parsed, err := arn.Parse(arnValue)
	if err != nil {
		return "", err
	}

	if parsed.AccountID == "aws" {
		return "aws", nil
	}

	return "local", nil
}

type attachedEntities struct {
	PolicyGroups []types.PolicyGroup
	PolicyRoles  []types.PolicyRole
	PolicyUsers  []types.PolicyUser
}

func (u *mqlAwsIamPolicy) listAttachedEntities(arn string) (attachedEntities, error) {
	c, ok := u.Cache.Load("_attachedentities")
	if ok {
		log.Debug().Msg("use attached entities from cache")
		return c.Data.(attachedEntities), nil
	}
	var res attachedEntities

	// if its not in the cache, fetch it
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return res, err
	}

	svc := provider.Iam("")
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
	u.Cache.Store("_attachedentities", &resources.CacheEntry{Data: res})
	return res, nil
}

func (u *mqlAwsIamPolicy) GetAttachedUsers() ([]interface{}, error) {
	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}
	entities, err := u.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range entities.PolicyUsers {
		usr := entities.PolicyUsers[i]

		mqlUser, err := u.MotorRuntime.CreateResource("aws.iam.user",
			"name", core.ToString(usr.UserName),
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (u *mqlAwsIamPolicy) GetAttachedRoles() ([]interface{}, error) {
	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}
	entities, err := u.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range entities.PolicyRoles {
		role := entities.PolicyRoles[i]

		mqlUser, err := u.MotorRuntime.CreateResource("aws.iam.role",
			"name", core.ToString(role.RoleName),
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (u *mqlAwsIamPolicy) GetAttachedGroups() ([]interface{}, error) {
	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}
	entities, err := u.listAttachedEntities(arn)
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range entities.PolicyGroups {
		group := entities.PolicyGroups[i]

		mqlUser, err := u.MotorRuntime.CreateResource("aws.iam.group",
			"name", core.ToString(group.GroupName),
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlUser)
	}
	return res, nil
}

func (u *mqlAwsIamPolicy) GetDefaultVersion() (interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}

	policyVersions, err := svc.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	for i := range policyVersions.Versions {
		policyversion := policyVersions.Versions[i]
		if policyversion.IsDefaultVersion == true {
			mqlAwsIamPolicyVersion, err := u.MotorRuntime.CreateResource("aws.iam.policyversion",
				"arn", arn,
				"versionId", core.ToString(policyversion.VersionId),
				"isDefaultVersion", policyversion.IsDefaultVersion,
				"createDate", policyversion.CreateDate,
			)
			if err != nil {
				return nil, err
			}
			return mqlAwsIamPolicyVersion, nil
		}
	}
	return nil, errors.New("unable to find default policy version")
}

func (u *mqlAwsIamPolicy) GetVersions() ([]interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	arn, err := u.Arn()
	if err != nil {
		return nil, err
	}

	policyVersions, err := svc.ListPolicyVersions(ctx, &iam.ListPolicyVersionsInput{PolicyArn: &arn})
	if err != nil {
		return nil, err
	}

	res := []interface{}{}
	for i := range policyVersions.Versions {
		policyversion := policyVersions.Versions[i]

		mqlAwsIamPolicyVersion, err := u.MotorRuntime.CreateResource("aws.iam.policyversion",
			"arn", arn,
			"versionId", core.ToString(policyversion.VersionId),
			"isDefaultVersion", policyversion.IsDefaultVersion,
			"createDate", policyversion.CreateDate,
		)
		if err != nil {
			return nil, err
		}

		res = append(res, mqlAwsIamPolicyVersion)
	}

	return res, nil
}

func (u *mqlAwsIamPolicyversion) id() (string, error) {
	arn, err := u.Arn()
	if err != nil {
		return "", err
	}

	versionid, err := u.VersionId()
	if err != nil {
		return "", err
	}

	return arn + "/" + versionid, nil
}

func (u *mqlAwsIamPolicyversion) GetDocument() (interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return "", err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	arn, err := u.Arn()
	if err != nil {
		return "", err
	}

	versionid, err := u.VersionId()
	if err != nil {
		return "", err
	}

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
	dict, err := core.JsonToDict(policyDoc)
	if err != nil {
		return "", err
	}
	return dict, nil
}

func (p *mqlAwsIamRole) init(args *resources.Args) (*resources.Args, AwsIamRole, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	if (*args)["arn"] == nil && (*args)["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws iam role")
	}

	// TODO: avoid reloading if all groups have been loaded already
	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	// TODO: handle arn and extract the name
	// if (*args)["arn"] != nil { }

	if (*args)["name"] != nil {
		rolename := (*args)["name"].(string)
		resp, err := svc.GetRole(ctx, &iam.GetRoleInput{
			RoleName: &rolename,
		})
		if err != nil {
			return nil, nil, err
		}

		role := resp.Role
		(*args)["arn"] = core.ToString(role.Arn)
		(*args)["id"] = core.ToString(role.RoleId)
		(*args)["name"] = core.ToString(role.RoleName)
		(*args)["description"] = core.ToString(role.Description)
		(*args)["tags"] = iamTagsToMap(role.Tags)
		(*args)["createDate"] = role.CreateDate

		return args, nil, nil
	}

	// if the package cannot be found, we init it as an empty package
	(*args)["arn"] = ""
	(*args)["id"] = ""
	(*args)["name"] = ""
	(*args)["description"] = ""
	(*args)["tags"] = make(map[string]interface{})
	(*args)["createDate"] = &time.Time{}

	return args, nil, nil
}

func (u *mqlAwsIamRole) id() (string, error) {
	return u.Arn()
}

func (p *mqlAwsIamGroup) init(args *resources.Args) (*resources.Args, AwsIamGroup, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}
	if len(*args) == 0 {
		if ids := getAssetIdentifier(p.MqlResource().MotorRuntime); ids != nil {
			(*args)["name"] = ids.name
			(*args)["arn"] = ids.arn
		}
	}
	if (*args)["arn"] == nil && (*args)["name"] == nil {
		return nil, nil, errors.New("arn or name required to fetch aws iam group")
	}

	// TODO: avoid reloading if all groups have been loaded already
	provider, err := awsProvider(p.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	// TODO: handle arn and extract the name
	// if (*args)["arn"] != nil { }

	if (*args)["name"] != nil {
		groupname := (*args)["name"].(string)
		resp, err := svc.GetGroup(ctx, &iam.GetGroupInput{
			GroupName: &groupname,
		})
		if err != nil {
			return nil, nil, err
		}
		usernames := []interface{}{}
		for _, user := range resp.Users {
			usernames = append(usernames, core.ToString(user.UserName))
		}

		grp := resp.Group
		(*args)["arn"] = core.ToString(grp.Arn)
		(*args)["id"] = core.ToString(grp.GroupId)
		(*args)["name"] = core.ToString(grp.GroupName)
		(*args)["createDate"] = grp.CreateDate
		(*args)["usernames"] = usernames

		return args, nil, nil
	}

	// if the package cannot be found, we init it as an empty package
	(*args)["arn"] = ""
	(*args)["id"] = ""
	(*args)["name"] = ""
	(*args)["createDate"] = &time.Time{}
	(*args)["usernames"] = []interface{}{}

	return args, nil, nil
}

func (u *mqlAwsIamGroup) id() (string, error) {
	return u.Arn()
}

func (u *mqlAwsIamUser) GetGroups() ([]interface{}, error) {
	provider, err := awsProvider(u.MotorRuntime.Motor.Provider)
	if err != nil {
		return nil, err
	}

	svc := provider.Iam("")
	ctx := context.Background()

	username, err := u.Name()
	if err != nil {
		return nil, err
	}

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
			res = append(res, core.ToString(userGroups.Groups[i].GroupName))
		}
		if userGroups.IsTruncated == false {
			break
		}
		marker = userGroups.Marker
	}

	return res, nil
}
