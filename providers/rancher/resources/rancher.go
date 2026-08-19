// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"strconv"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

// Setting names the server-wide configuration this provider reads directly.
// Settings are added and removed between Rancher releases, so every read of one
// has to survive its absence.
const (
	settingServerVersion       = "server-version"
	settingServerURL           = "server-url"
	settingTelemetryOpt        = "telemetry-opt"
	settingTemplateEnforcement = "cluster-template-enforcement"
	settingAuthTokenMaxTTL     = "auth-token-max-ttl-minutes"
	settingKubeconfigTokenTTL  = "kubeconfig-default-token-ttl-minutes"
	settingPasswordMinLength   = "password-min-length"
)

// localAuthConfigType is the type the built-in username and password store
// reports itself as in the authConfigs collection.
const localAuthConfigType = "localConfig"

func (r *mqlRancher) id() (string, error) {
	conn, err := rancherConnection(r.MqlRuntime)
	if err != nil {
		return "", err
	}
	return "rancher/" + conn.Host(), nil
}

// settingValue returns the value in force for a setting, and whether the server
// has the setting at all. A setting with no value falls back to the value
// Rancher ships, which is what the server itself would act on.
func (r *mqlRancher) settingValue(name string) (string, bool, error) {
	records, err := listRecords[settingRecord](r.MqlRuntime, pathSettings)
	if err != nil {
		return "", false, err
	}
	for i := range records {
		record := &records[i]
		key := record.Name
		if key == "" {
			key = record.ID
		}
		if key != name {
			continue
		}
		if record.Value != "" {
			return record.Value, true, nil
		}
		return record.Default, true, nil
	}
	return "", false, nil
}

func (r *mqlRancher) settings() ([]any, error) {
	records, err := listRecords[settingRecord](r.MqlRuntime, pathSettings)
	if err != nil {
		return nil, err
	}

	res := make([]any, 0, len(records))
	for i := range records {
		record := &records[i]
		name := record.Name
		if name == "" {
			name = record.ID
		}
		mqlSetting, err := CreateResource(r.MqlRuntime, "rancher.setting", map[string]*llx.RawData{
			"__id":         llx.StringData(name),
			"name":         llx.StringData(name),
			"value":        llx.StringData(record.Value),
			"defaultValue": llx.StringData(record.Default),
			"customized":   llx.BoolData(record.Customized),
			"source":       llx.StringData(record.Source),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlSetting)
	}
	return res, nil
}

func (r *mqlRancher) version() (string, error) {
	value, present, err := r.settingValue(settingServerVersion)
	if err != nil {
		return "", err
	}
	if !present {
		r.Version.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return value, nil
}

func (r *mqlRancher) serverUrl() (string, error) {
	value, present, err := r.settingValue(settingServerURL)
	if err != nil {
		return "", err
	}
	if !present {
		r.ServerUrl.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return value, nil
}

func (r *mqlRancher) telemetryOptIn() (string, error) {
	value, present, err := r.settingValue(settingTelemetryOpt)
	if err != nil {
		return "", err
	}
	if !present {
		// Rancher dropped the setting after 2.9. Reporting an empty choice
		// would read as "nobody decided" on a server where the question is no
		// longer asked.
		r.TelemetryOptIn.State = plugin.StateIsSet | plugin.StateIsNull
		return "", nil
	}
	return value, nil
}

func (r *mqlRancher) clusterTemplateEnforcement() (bool, error) {
	// The setting outlives the feature. Rancher 2.12 removed cluster templates
	// with RKE1 but still ships cluster-template-enforcement, and nothing reads
	// it any more; a fleet upgraded with the setting left at true would
	// otherwise be reported as enforcing a control that no longer exists. So
	// the endpoint decides whether there is a control to report at all, and the
	// setting only decides which way it is pointed.
	supported, err := endpointExists(r.MqlRuntime, pathClusterTemplates)
	if err != nil {
		return false, err
	}

	value, present, err := r.settingValue(settingTemplateEnforcement)
	if err != nil {
		return false, err
	}
	parsed, ok := parseBoolSetting(value)

	if !supported || !present || !ok {
		// Reporting false would say the control exists and is off, which is a
		// different and more reassuring claim than "there is no such control".
		// An unparseable value is treated the same way: we do not know which
		// way it went, so it must fail a check rather than pass one.
		r.ClusterTemplateEnforcement.State = plugin.StateIsSet | plugin.StateIsNull
		return false, nil
	}
	return parsed, nil
}

func (r *mqlRancher) authTokenMaxTtlMinutes() (int64, error) {
	return r.intSetting(settingAuthTokenMaxTTL, &r.AuthTokenMaxTtlMinutes)
}

func (r *mqlRancher) kubeconfigDefaultTokenTtlMinutes() (int64, error) {
	return r.intSetting(settingKubeconfigTokenTTL, &r.KubeconfigDefaultTokenTtlMinutes)
}

func (r *mqlRancher) passwordMinLength() (int64, error) {
	return r.intSetting(settingPasswordMinLength, &r.PasswordMinLength)
}

// intSetting reads a numeric setting, reporting an absent or unparseable one as
// null rather than as zero. Zero is a meaningful value for every one of these
// settings, so it must not double as "we could not read it".
func (r *mqlRancher) intSetting(name string, field *plugin.TValue[int64]) (int64, error) {
	value, present, err := r.settingValue(name)
	if err != nil {
		return 0, err
	}
	if !present {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return 0, nil
	}
	return parsed, nil
}

func (r *mqlRancher) localAuthEnabled() (bool, error) {
	records, err := listRecords[authConfigRecord](r.MqlRuntime, pathAuthConfigs)
	if err != nil {
		return false, err
	}
	for i := range records {
		if records[i].Type == localAuthConfigType {
			return records[i].Enabled, nil
		}
	}
	// The server did not report a local provider at all. Saying "local auth is
	// off" would be a claim we cannot support.
	r.LocalAuthEnabled.State = plugin.StateIsSet | plugin.StateIsNull
	return false, nil
}

func (r *mqlRancher) externalAuthEnabled() (bool, error) {
	records, err := listRecords[authConfigRecord](r.MqlRuntime, pathAuthConfigs)
	if err != nil {
		return false, err
	}
	for i := range records {
		if records[i].Type != localAuthConfigType && records[i].Enabled {
			return true, nil
		}
	}
	return false, nil
}

// parseBoolSetting reads a Rancher boolean setting, which is stored as text.
func parseBoolSetting(value string) (bool, bool) {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, false
	}
	return parsed, true
}
