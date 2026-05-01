// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers/datadog/connection"
)

type mqlDatadogInternal struct {
	ipAllowlistOnce  sync.Once
	ipAllowlistAttrs datadogV2.IPAllowlistAttributes
	ipAllowlistErr   error

	usersOnce sync.Once
	usersList []datadogV2.User
	usersErr  error
}

// isForbidden checks if an HTTP response indicates a 403 Forbidden error,
// which typically means the Datadog plan does not include the required feature.
func isForbidden(resp *http.Response) bool {
	return resp != nil && resp.StatusCode == http.StatusForbidden
}

func (r *mqlDatadog) id() (string, error) {
	return "datadog", nil
}

func toAnyStrings(s []string) []interface{} {
	r := make([]interface{}, len(s))
	for i, v := range s {
		r[i] = v
	}
	return r
}

func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func parseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}
	return &t
}

// fetchUsers fetches all users once and caches the result, shared by users() and serviceAccounts().
func (r *mqlDatadog) fetchUsers() ([]datadogV2.User, error) {
	r.usersOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
		api := datadogV2.NewUsersApi(conn.ApiClient())

		pageSize := int64(100)
		items, cancel := api.ListUsersWithPagination(conn.AuthCtx(),
			*datadogV2.NewListUsersOptionalParameters().WithPageSize(pageSize))
		defer cancel()

		for item := range items {
			if item.Error != nil {
				r.usersErr = item.Error
				return
			}
			r.usersList = append(r.usersList, item.Item)
		}
	})
	return r.usersList, r.usersErr
}

// --- Users ---

func (r *mqlDatadog) users() ([]interface{}, error) {
	usersList, err := r.fetchUsers()
	if err != nil {
		return nil, err
	}

	var all []interface{}
	for _, u := range usersList {
		attrs := u.GetAttributes()
		res, err := CreateResource(r.MqlRuntime, "datadog.user", map[string]*llx.RawData{
			"id":             llx.StringData(u.GetId()),
			"email":          llx.StringData(attrs.GetEmail()),
			"name":           llx.StringData(attrs.GetName()),
			"handle":         llx.StringData(attrs.GetHandle()),
			"status":         llx.StringData(attrs.GetStatus()),
			"title":          llx.StringData(attrs.GetTitle()),
			"serviceAccount": llx.BoolData(attrs.GetServiceAccount()),
			"verified":       llx.BoolData(attrs.GetVerified()),
			"disabled":       llx.BoolData(attrs.GetDisabled()),
			"createdAt":      llx.TimeDataPtr(timePtr(attrs.GetCreatedAt())),
			"icon":           llx.StringData(attrs.GetIcon()),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogUser) id() (string, error) {
	return "datadog.user/" + r.Id.Data, nil
}

// --- Roles ---

func (r *mqlDatadog) roles() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewRolesApi(conn.ApiClient())

	resp, httpResp, err := api.ListRoles(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> roles not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, role := range resp.GetData() {
		attrs := role.GetAttributes()
		res, err := CreateResource(r.MqlRuntime, "datadog.role", map[string]*llx.RawData{
			"id":         llx.StringData(role.GetId()),
			"name":       llx.StringData(attrs.GetName()),
			"userCount":  llx.IntData(int64(attrs.GetUserCount())),
			"createdAt":  llx.TimeDataPtr(timePtr(attrs.GetCreatedAt())),
			"modifiedAt": llx.TimeDataPtr(timePtr(attrs.GetModifiedAt())),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogRole) id() (string, error) {
	return "datadog.role/" + r.Id.Data, nil
}

// --- Monitors ---

func (r *mqlDatadog) monitors() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV1.NewMonitorsApi(conn.ApiClient())

	var all []interface{}
	pageSize := int32(100)
	page := int64(0)

	for {
		monitors, httpResp, err := api.ListMonitors(conn.AuthCtx(),
			*datadogV1.NewListMonitorsOptionalParameters().WithPageSize(pageSize).WithPage(page))
		if err != nil {
			if isForbidden(httpResp) {
				log.Warn().Msg("datadog> monitors not available (403 Forbidden)")
				return nil, nil
			}
			return nil, err
		}

		for _, m := range monitors {
			tags := toAnyStrings(m.GetTags())

			creator := ""
			if c, ok := m.GetCreatorOk(); ok && c != nil {
				creator = c.GetEmail()
			}

			priority := int64(0)
			if p, ok := m.GetPriorityOk(); ok && p != nil {
				priority = *p
			}

			notifyNoData := false
			if opts, ok := m.GetOptionsOk(); ok && opts != nil {
				notifyNoData = opts.GetNotifyNoData()
			}

			options := map[string]interface{}{}
			if opts, ok := m.GetOptionsOk(); ok && opts != nil {
				if v, ok := opts.GetRenotifyIntervalOk(); ok && v != nil {
					options["renotifyInterval"] = float64(*v)
				}
				if v, ok := opts.GetTimeoutHOk(); ok && v != nil {
					options["timeoutH"] = float64(*v)
				}
				if v, ok := opts.GetEvaluationDelayOk(); ok && v != nil {
					options["evaluationDelay"] = float64(*v)
				}
				if v, ok := opts.GetNotifyAuditOk(); ok && v != nil {
					options["notifyAudit"] = *v
				}
			}

			res, err := CreateResource(r.MqlRuntime, "datadog.monitor", map[string]*llx.RawData{
				"id":           llx.IntData(m.GetId()),
				"name":         llx.StringData(m.GetName()),
				"type":         llx.StringData(string(m.GetType())),
				"query":        llx.StringData(m.GetQuery()),
				"message":      llx.StringData(m.GetMessage()),
				"overallState": llx.StringData(string(m.GetOverallState())),
				"tags":         llx.ArrayData(tags, "\x02"),
				"priority":     llx.IntData(priority),
				"created":      llx.TimeDataPtr(timePtr(m.GetCreated())),
				"modified":     llx.TimeDataPtr(timePtr(m.GetModified())),
				"creator":      llx.StringData(creator),
				"notifyNoData": llx.BoolData(notifyNoData),
				"options":      llx.DictData(options),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}

		if len(monitors) < int(pageSize) {
			break
		}
		page++
	}
	return all, nil
}

func (r *mqlDatadogMonitor) id() (string, error) {
	return "datadog.monitor/" + strconv.FormatInt(r.Id.Data, 10), nil
}

// --- Dashboards ---

func (r *mqlDatadog) dashboards() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV1.NewDashboardsApi(conn.ApiClient())

	resp, httpResp, err := api.ListDashboards(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> dashboards not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, d := range resp.GetDashboards() {
		res, err := CreateResource(r.MqlRuntime, "datadog.dashboard", map[string]*llx.RawData{
			"id":           llx.StringData(d.GetId()),
			"title":        llx.StringData(d.GetTitle()),
			"description":  llx.StringData(d.GetDescription()),
			"layoutType":   llx.StringData(string(d.GetLayoutType())),
			"url":          llx.StringData(d.GetUrl()),
			"createdAt":    llx.TimeDataPtr(timePtr(d.GetCreatedAt())),
			"modifiedAt":   llx.TimeDataPtr(timePtr(d.GetModifiedAt())),
			"authorHandle": llx.StringData(d.GetAuthorHandle()),
			"isReadOnly":   llx.BoolData(d.GetIsReadOnly()),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogDashboard) id() (string, error) {
	return "datadog.dashboard/" + r.Id.Data, nil
}

// --- Synthetics Tests ---

func (r *mqlDatadog) syntheticsTests() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV1.NewSyntheticsApi(conn.ApiClient())

	resp, httpResp, err := api.ListTests(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> synthetics tests not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, t := range resp.GetTests() {
		tags := toAnyStrings(t.GetTags())
		locations := toAnyStrings(t.GetLocations())

		creator := ""
		if c, ok := t.GetCreatorOk(); ok && c != nil {
			creator = c.GetEmail()
		}

		config := map[string]interface{}{}
		if cfg, ok := t.GetConfigOk(); ok && cfg != nil {
			if req, ok := cfg.GetRequestOk(); ok && req != nil {
				config["method"] = req.GetMethod()
				config["url"] = req.GetUrl()
			}
		}

		options := map[string]interface{}{}
		if opts, ok := t.GetOptionsOk(); ok && opts != nil {
			if v, ok := opts.GetTickEveryOk(); ok {
				options["tickEvery"] = float64(*v)
			}
			if v, ok := opts.GetFollowRedirectsOk(); ok {
				options["followRedirects"] = *v
			}
			if v, ok := opts.GetMinFailureDurationOk(); ok {
				options["minFailureDuration"] = float64(*v)
			}
		}

		res, err := CreateResource(r.MqlRuntime, "datadog.syntheticsTest", map[string]*llx.RawData{
			"publicId":  llx.StringData(t.GetPublicId()),
			"name":      llx.StringData(t.GetName()),
			"type":      llx.StringData(string(t.GetType())),
			"subtype":   llx.StringData(string(t.GetSubtype())),
			"status":    llx.StringData(string(t.GetStatus())),
			"message":   llx.StringData(t.GetMessage()),
			"tags":      llx.ArrayData(tags, "\x02"),
			"locations": llx.ArrayData(locations, "\x02"),
			"monitorId": llx.IntData(t.GetMonitorId()),
			"creator":   llx.StringData(creator),
			"config":    llx.DictData(config),
			"options":   llx.DictData(options),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogSyntheticsTest) id() (string, error) {
	return "datadog.syntheticsTest/" + r.PublicId.Data, nil
}

// --- SLOs ---

func (r *mqlDatadog) slos() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV1.NewServiceLevelObjectivesApi(conn.ApiClient())

	resp, httpResp, err := api.ListSLOs(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> SLOs not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, s := range resp.GetData() {
		tags := toAnyStrings(s.GetTags())

		creator := ""
		if c, ok := s.GetCreatorOk(); ok && c != nil {
			creator = c.GetEmail()
		}

		targetThreshold := float64(0)
		warningThreshold := float64(0)
		timeframe := ""
		if thresholds := s.GetThresholds(); len(thresholds) > 0 {
			targetThreshold = thresholds[0].GetTarget()
			if w, ok := thresholds[0].GetWarningOk(); ok && w != nil {
				warningThreshold = *w
			}
			timeframe = string(thresholds[0].GetTimeframe())
		}

		monitorIds := make([]interface{}, len(s.GetMonitorIds()))
		for i, id := range s.GetMonitorIds() {
			monitorIds[i] = id
		}

		res, err := CreateResource(r.MqlRuntime, "datadog.slo", map[string]*llx.RawData{
			"id":               llx.StringData(s.GetId()),
			"name":             llx.StringData(s.GetName()),
			"type":             llx.StringData(string(s.GetType())),
			"description":      llx.StringData(s.GetDescription()),
			"tags":             llx.ArrayData(tags, "\x02"),
			"targetThreshold":  llx.FloatData(targetThreshold),
			"warningThreshold": llx.FloatData(warningThreshold),
			"timeframe":        llx.StringData(timeframe),
			"creator":          llx.StringData(creator),
			"createdAt":        llx.TimeDataPtr(timePtr(time.Unix(s.GetCreatedAt(), 0))),
			"modifiedAt":       llx.TimeDataPtr(timePtr(time.Unix(s.GetModifiedAt(), 0))),
			"monitorIds":       llx.ArrayData(monitorIds, "\x05"),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogSlo) id() (string, error) {
	return "datadog.slo/" + r.Id.Data, nil
}

// --- Log Indexes ---

func (r *mqlDatadog) logIndexes() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV1.NewLogsIndexesApi(conn.ApiClient())

	resp, httpResp, err := api.ListLogIndexes(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> log indexes not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, idx := range resp.GetIndexes() {
		filterQuery := ""
		if f, ok := idx.GetFilterOk(); ok && f != nil {
			filterQuery = f.GetQuery()
		}

		exclusionFilters := make([]interface{}, len(idx.GetExclusionFilters()))
		for i, ef := range idx.GetExclusionFilters() {
			efMap := map[string]interface{}{
				"name":      ef.GetName(),
				"isEnabled": ef.GetIsEnabled(),
			}
			if f, ok := ef.GetFilterOk(); ok && f != nil {
				efMap["query"] = f.GetQuery()
				efMap["sampleRate"] = f.GetSampleRate()
			}
			exclusionFilters[i] = efMap
		}

		dailyLimit := int64(0)
		if v, ok := idx.GetDailyLimitOk(); ok && v != nil {
			dailyLimit = *v
		}

		warnPct := float64(0)
		if v, ok := idx.GetDailyLimitWarningThresholdPercentageOk(); ok && v != nil {
			warnPct = *v
		}

		numRetention := int64(0)
		if v, ok := idx.GetNumRetentionDaysOk(); ok && v != nil {
			numRetention = *v
		}

		numFlex := int64(0)
		if v, ok := idx.GetNumFlexLogsRetentionDaysOk(); ok && v != nil {
			numFlex = *v
		}

		res, err := CreateResource(r.MqlRuntime, "datadog.logIndex", map[string]*llx.RawData{
			"name":                                 llx.StringData(idx.GetName()),
			"filter":                               llx.StringData(filterQuery),
			"numRetentionDays":                     llx.IntData(numRetention),
			"dailyLimit":                           llx.IntData(dailyLimit),
			"dailyLimitWarningThresholdPercentage": llx.FloatData(warnPct),
			"isRateLimited":                        llx.BoolData(idx.GetIsRateLimited()),
			"numFlexLogsRetentionDays":             llx.IntData(numFlex),
			"exclusionFilters":                     llx.ArrayData(exclusionFilters, "\x13"),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogLogIndex) id() (string, error) {
	return "datadog.logIndex/" + r.Name.Data, nil
}

// --- Security Rules ---

func (r *mqlDatadog) securityRules() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewSecurityMonitoringApi(conn.ApiClient())

	var all []interface{}
	pageSize := int64(100)
	page := int64(0)

	for {
		resp, httpResp, err := api.ListSecurityMonitoringRules(conn.AuthCtx(),
			*datadogV2.NewListSecurityMonitoringRulesOptionalParameters().WithPageSize(pageSize).WithPageNumber(page))
		if err != nil {
			if isForbidden(httpResp) {
				log.Warn().Msg("datadog> security monitoring rules not available (403 Forbidden). Your Datadog plan may not include Cloud SIEM")
				return nil, nil
			}
			return nil, err
		}

		for _, ruleWrapper := range resp.GetData() {
			// The response is a union type; extract the standard rule if present
			rule := ruleWrapper.SecurityMonitoringStandardRuleResponse
			if rule == nil {
				continue
			}

			tags := toAnyStrings(rule.GetTags())

			cases := make([]interface{}, len(rule.GetCases()))
			for i, c := range rule.GetCases() {
				cases[i] = map[string]interface{}{
					"name":      c.GetName(),
					"status":    string(c.GetStatus()),
					"condition": c.GetCondition(),
				}
			}

			filters := make([]interface{}, len(rule.GetFilters()))
			for i, f := range rule.GetFilters() {
				filters[i] = map[string]interface{}{
					"query":  f.GetQuery(),
					"action": string(f.GetAction()),
				}
			}

			options := map[string]interface{}{}
			if opts, ok := rule.GetOptionsOk(); ok && opts != nil {
				options["detectionMethod"] = string(opts.GetDetectionMethod())
				if v, ok := opts.GetEvaluationWindowOk(); ok {
					options["evaluationWindow"] = float64(*v)
				}
				if v, ok := opts.GetKeepAliveOk(); ok {
					options["keepAlive"] = float64(*v)
				}
				if v, ok := opts.GetMaxSignalDurationOk(); ok {
					options["maxSignalDuration"] = float64(*v)
				}
			}

			res, err := CreateResource(r.MqlRuntime, "datadog.securityRule", map[string]*llx.RawData{
				"id":               llx.StringData(rule.GetId()),
				"name":             llx.StringData(rule.GetName()),
				"type":             llx.StringData(string(rule.GetType())),
				"message":          llx.StringData(rule.GetMessage()),
				"isEnabled":        llx.BoolData(rule.GetIsEnabled()),
				"hasExtendedTitle": llx.BoolData(rule.GetHasExtendedTitle()),
				"tags":             llx.ArrayData(tags, "\x02"),
				"isDefault":        llx.BoolData(rule.GetIsDefault()),
				"isDeleted":        llx.BoolData(rule.GetIsDeleted()),
				"createdAt":        llx.TimeDataPtr(timePtr(time.UnixMilli(rule.GetCreatedAt()))),
				"updatedAt":        llx.TimeDataPtr(timePtr(time.UnixMilli(rule.GetUpdatedAt()))),
				"cases":            llx.ArrayData(cases, "\x13"),
				"filters":          llx.ArrayData(filters, "\x13"),
				"options":          llx.DictData(options),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}

		if int64(len(resp.GetData())) < pageSize {
			break
		}
		page++
	}
	return all, nil
}

func (r *mqlDatadogSecurityRule) id() (string, error) {
	return "datadog.securityRule/" + r.Id.Data, nil
}

// --- Downtimes ---

func (r *mqlDatadog) downtimes() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewDowntimesApi(conn.ApiClient())

	var all []interface{}
	items, cancel := api.ListDowntimesWithPagination(conn.AuthCtx())
	defer cancel()

	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		d := item.Item
		attrs := d.GetAttributes()

		notifyEndStates := make([]interface{}, 0)
		for _, v := range attrs.GetNotifyEndStates() {
			notifyEndStates = append(notifyEndStates, string(v))
		}

		notifyEndTypes := make([]interface{}, 0)
		for _, v := range attrs.GetNotifyEndTypes() {
			notifyEndTypes = append(notifyEndTypes, string(v))
		}

		monitorId := map[string]interface{}{}
		if mi, ok := attrs.GetMonitorIdentifierOk(); ok && mi != nil {
			if mid := mi.DowntimeMonitorIdentifierId; mid != nil {
				monitorId["monitorId"] = float64(mid.GetMonitorId())
			}
			if mtags := mi.DowntimeMonitorIdentifierTags; mtags != nil {
				monitorId["monitorTags"] = mtags.GetMonitorTags()
			}
		}

		schedule := map[string]interface{}{}
		if sched, ok := attrs.GetScheduleOk(); ok && sched != nil {
			if rrule := sched.DowntimeScheduleRecurrencesResponse; rrule != nil {
				schedule["timezone"] = rrule.GetTimezone()
			}
			if oneTime := sched.DowntimeScheduleOneTimeResponse; oneTime != nil {
				if s, ok := oneTime.GetStartOk(); ok && s != nil {
					schedule["start"] = s.Format(time.RFC3339)
				}
				if e, ok := oneTime.GetEndOk(); ok && e != nil {
					schedule["end"] = (*e).Format(time.RFC3339)
				}
			}
		}

		scope := attrs.GetScope()

		var canceledAt *time.Time
		if t, ok := attrs.GetCanceledOk(); ok && t != nil {
			v := *t
			if !v.IsZero() {
				canceledAt = &v
			}
		}

		res, err := CreateResource(r.MqlRuntime, "datadog.downtime", map[string]*llx.RawData{
			"id":                            llx.StringData(d.GetId()),
			"displayTimezone":               llx.StringData(attrs.GetDisplayTimezone()),
			"message":                       llx.StringData(attrs.GetMessage()),
			"muteFirstRecoveryNotification": llx.BoolData(attrs.GetMuteFirstRecoveryNotification()),
			"notifyEndStates":               llx.ArrayData(notifyEndStates, "\x02"),
			"notifyEndTypes":                llx.ArrayData(notifyEndTypes, "\x02"),
			"status":                        llx.StringData(string(attrs.GetStatus())),
			"monitorIdentifier":             llx.DictData(monitorId),
			"schedule":                      llx.DictData(schedule),
			"scope":                         llx.StringData(scope),
			"createdAt":                     llx.TimeDataPtr(timePtr(attrs.GetCreated())),
			"modifiedAt":                    llx.TimeDataPtr(timePtr(attrs.GetModified())),
			"canceledAt":                    llx.TimeDataPtr(canceledAt),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogDowntime) id() (string, error) {
	return "datadog.downtime/" + r.Id.Data, nil
}

// --- API Keys ---

func (r *mqlDatadog) apiKeys() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewKeyManagementApi(conn.ApiClient())

	var all []interface{}
	pageSize := int64(100)
	page := int64(0)

	for {
		resp, httpResp, err := api.ListAPIKeys(conn.AuthCtx(),
			*datadogV2.NewListAPIKeysOptionalParameters().WithPageSize(pageSize).WithPageNumber(page))
		if err != nil {
			if isForbidden(httpResp) {
				log.Warn().Msg("datadog> API keys not available (403 Forbidden)")
				return nil, nil
			}
			return nil, err
		}

		for _, k := range resp.GetData() {
			attrs := k.GetAttributes()
			res, err := CreateResource(r.MqlRuntime, "datadog.apiKey", map[string]*llx.RawData{
				"id":         llx.StringData(k.GetId()),
				"name":       llx.StringData(attrs.GetName()),
				"createdAt":  llx.TimeDataPtr(parseTime(attrs.GetCreatedAt())),
				"modifiedAt": llx.TimeDataPtr(parseTime(attrs.GetModifiedAt())),
				"last4":      llx.StringData(attrs.GetLast4()),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}

		if int64(len(resp.GetData())) < pageSize {
			break
		}
		page++
	}
	return all, nil
}

func (r *mqlDatadogApiKey) id() (string, error) {
	return "datadog.apiKey/" + r.Id.Data, nil
}

// --- Application Keys ---

func (r *mqlDatadog) applicationKeys() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewKeyManagementApi(conn.ApiClient())

	var all []interface{}
	pageSize := int64(100)
	page := int64(0)

	for {
		resp, httpResp, err := api.ListApplicationKeys(conn.AuthCtx(),
			*datadogV2.NewListApplicationKeysOptionalParameters().WithPageSize(pageSize).WithPageNumber(page))
		if err != nil {
			if isForbidden(httpResp) {
				log.Warn().Msg("datadog> application keys not available (403 Forbidden)")
				return nil, nil
			}
			return nil, err
		}

		for _, k := range resp.GetData() {
			attrs := k.GetAttributes()
			scopes := toAnyStrings(attrs.GetScopes())

			res, err := CreateResource(r.MqlRuntime, "datadog.applicationKey", map[string]*llx.RawData{
				"id":        llx.StringData(k.GetId()),
				"name":      llx.StringData(attrs.GetName()),
				"createdAt": llx.TimeDataPtr(parseTime(attrs.GetCreatedAt())),
				"last4":     llx.StringData(attrs.GetLast4()),
				"scopes":    llx.ArrayData(scopes, "\x02"),
			})
			if err != nil {
				return nil, err
			}
			all = append(all, res)
		}

		if int64(len(resp.GetData())) < pageSize {
			break
		}
		page++
	}
	return all, nil
}

func (r *mqlDatadogApplicationKey) id() (string, error) {
	return "datadog.applicationKey/" + r.Id.Data, nil
}

// --- IP Allowlist ---

func (r *mqlDatadog) fetchIPAllowlist() (datadogV2.IPAllowlistAttributes, error) {
	r.ipAllowlistOnce.Do(func() {
		conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
		api := datadogV2.NewIPAllowlistApi(conn.ApiClient())

		resp, httpResp, err := api.GetIPAllowlist(conn.AuthCtx())
		if err != nil {
			if isForbidden(httpResp) {
				log.Warn().Msg("datadog> IP allowlist not available (403 Forbidden)")
				return
			}
			r.ipAllowlistErr = err
			return
		}

		data := resp.GetData()
		r.ipAllowlistAttrs = data.GetAttributes()
	})
	return r.ipAllowlistAttrs, r.ipAllowlistErr
}

func (r *mqlDatadog) ipAllowlistEnabled() (bool, error) {
	attrs, err := r.fetchIPAllowlist()
	if err != nil {
		return false, err
	}
	return attrs.GetEnabled(), nil
}

func (r *mqlDatadog) ipAllowlistEntries() ([]interface{}, error) {
	attrs, err := r.fetchIPAllowlist()
	if err != nil {
		return nil, err
	}

	var all []interface{}
	for _, entry := range attrs.GetEntries() {
		entryData := entry.GetData()
		entryAttrs := entryData.GetAttributes()
		all = append(all, map[string]interface{}{
			"cidrBlock":  entryAttrs.GetCidrBlock(),
			"note":       entryAttrs.GetNote(),
			"createdAt":  entryAttrs.GetCreatedAt().Format(time.RFC3339),
			"modifiedAt": entryAttrs.GetModifiedAt().Format(time.RFC3339),
		})
	}
	return all, nil
}

// --- AWS Integrations ---

func (r *mqlDatadog) integrationAwsAccounts() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewAWSIntegrationApi(conn.ApiClient())

	resp, httpResp, err := api.ListAWSAccounts(conn.AuthCtx())
	if err != nil {
		if isForbidden(httpResp) {
			log.Warn().Msg("datadog> AWS integrations not available (403 Forbidden)")
			return nil, nil
		}
		return nil, err
	}

	var all []interface{}
	for _, acct := range resp.GetData() {
		attrs := acct.GetAttributes()

		accountTags := toAnyStrings(attrs.GetAccountTags())

		metricsEnabled := false
		if mc, ok := attrs.GetMetricsConfigOk(); ok && mc != nil {
			metricsEnabled = mc.GetEnabled()
		}

		resourceCollectionEnabled := false
		if rc, ok := attrs.GetResourcesConfigOk(); ok && rc != nil {
			resourceCollectionEnabled = rc.GetCloudSecurityPostureManagementCollection()
		}

		logsEnabled := false
		if lc, ok := attrs.GetLogsConfigOk(); ok && lc != nil {
			if lf, ok := lc.GetLambdaForwarderOk(); ok && lf != nil {
				logsEnabled = len(lf.GetLambdas()) > 0
			}
		}

		// Extract auth config role name
		roleName := ""
		if ac, ok := attrs.GetAuthConfigOk(); ok && ac != nil {
			if arn := ac.AWSAuthConfigRole; arn != nil {
				roleName = arn.GetRoleName()
			}
		}

		// Extract metrics tag filters as filter tags
		var filterTags []interface{}
		if mc, ok := attrs.GetMetricsConfigOk(); ok && mc != nil {
			for _, tf := range mc.GetTagFilters() {
				for _, tag := range tf.GetTags() {
					filterTags = append(filterTags, tag)
				}
			}
		}
		if filterTags == nil {
			filterTags = []interface{}{}
		}

		res, err := CreateResource(r.MqlRuntime, "datadog.integration.aws", map[string]*llx.RawData{
			"accountId":                 llx.StringData(attrs.GetAwsAccountId()),
			"roleName":                  llx.StringData(roleName),
			"metricsEnabled":            llx.BoolData(metricsEnabled),
			"resourceCollectionEnabled": llx.BoolData(resourceCollectionEnabled),
			"logsEnabled":               llx.BoolData(logsEnabled),
			"filterTags":                llx.ArrayData(filterTags, "\x02"),
			"hostTags":                  llx.ArrayData([]interface{}{}, "\x02"),
			"accountTags":               llx.ArrayData(accountTags, "\x02"),
			"excludedRegions":           llx.ArrayData([]interface{}{}, "\x02"),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogIntegrationAws) id() (string, error) {
	return "datadog.integration.aws/" + r.AccountId.Data, nil
}

// --- Teams ---

func (r *mqlDatadog) teams() ([]interface{}, error) {
	conn := r.MqlRuntime.Connection.(*connection.DatadogConnection)
	api := datadogV2.NewTeamsApi(conn.ApiClient())

	var all []interface{}
	pageSize := int64(100)
	items, cancel := api.ListTeamsWithPagination(conn.AuthCtx(),
		*datadogV2.NewListTeamsOptionalParameters().WithPageSize(pageSize))
	defer cancel()

	for item := range items {
		if item.Error != nil {
			return nil, item.Error
		}
		t := item.Item
		attrs := t.GetAttributes()
		res, err := CreateResource(r.MqlRuntime, "datadog.team", map[string]*llx.RawData{
			"id":          llx.StringData(t.GetId()),
			"name":        llx.StringData(attrs.GetName()),
			"handle":      llx.StringData(attrs.GetHandle()),
			"description": llx.StringData(attrs.GetDescription()),
			"userCount":   llx.IntData(int64(attrs.GetUserCount())),
			"createdAt":   llx.TimeDataPtr(timePtr(attrs.GetCreatedAt())),
			"modifiedAt":  llx.TimeDataPtr(timePtr(attrs.GetModifiedAt())),
		})
		if err != nil {
			return nil, err
		}
		all = append(all, res)
	}
	return all, nil
}

func (r *mqlDatadogTeam) id() (string, error) {
	return "datadog.team/" + r.Id.Data, nil
}
