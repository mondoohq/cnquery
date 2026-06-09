// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"regexp"
	"strings"
)

// Compiled once for the two fields the metric-filter accessors select on,
// rather than recompiling on every call.
var (
	monitoredEventNameRe   = filterPatternValueRe("eventName")
	monitoredEventSourceRe = filterPatternValueRe("eventSource")
)

// filterPatternValueRe matches an equality selector in a CloudWatch Logs metric
// filter pattern — `$.<field> = <value>` — capturing the value whether it is
// quoted or bare. The `=` is not preceded by `!`, so `!=` selectors are
// excluded.
func filterPatternValueRe(field string) *regexp.Regexp {
	return regexp.MustCompile(`\$\.` + regexp.QuoteMeta(field) + `\s*=\s*(?:"([^"]*)"|([^\s)|&"]+))`)
}

// extractFilterPatternValues returns the distinct values a metric filter pattern
// selects via the `$.<field> = <value>` terms matched by re, preserving the
// order they appear. Inequality (`!=`) terms are ignored.
func extractFilterPatternValues(re *regexp.Regexp, pattern string) []string {
	matches := re.FindAllStringSubmatch(pattern, -1)
	res := []string{}
	seen := map[string]struct{}{}
	for _, m := range matches {
		v := m[1]
		if v == "" {
			v = m[2]
		}
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		res = append(res, v)
	}
	return res
}

func (a *mqlAwsCloudwatchLoggroupMetricsfilter) monitoredEventNames() ([]any, error) {
	pattern := a.GetFilterPattern()
	if pattern.Error != nil {
		return nil, pattern.Error
	}
	return toAnySlice(extractFilterPatternValues(monitoredEventNameRe, pattern.Data)), nil
}

func (a *mqlAwsCloudwatchLoggroupMetricsfilter) monitoredEventSources() ([]any, error) {
	pattern := a.GetFilterPattern()
	if pattern.Error != nil {
		return nil, pattern.Error
	}
	return toAnySlice(extractFilterPatternValues(monitoredEventSourceRe, pattern.Data)), nil
}

// isActiveSubscriptionArn reports whether an SNS subscription ARN belongs to a
// confirmed subscriber. SNS reports "PendingConfirmation" or "Deleted" in place
// of a real ARN for subscriptions that cannot receive notifications.
func isActiveSubscriptionArn(arn string) bool {
	return arn != "" && arn != "PendingConfirmation" && arn != "Deleted"
}

// hasActiveAlarm reports whether any of the filter's metrics has an alarm that
// notifies a confirmed SNS subscriber. It walks metrics -> alarms -> SNS topic
// actions -> subscriptions, the same chain CIS monitoring controls assert.
func (a *mqlAwsCloudwatchLoggroupMetricsfilter) hasActiveAlarm() (bool, error) {
	metrics := a.GetMetrics()
	if metrics.Error != nil {
		return false, metrics.Error
	}
	for _, m := range metrics.Data {
		metric := m.(*mqlAwsCloudwatchMetric)
		alarms := metric.GetAlarms()
		if alarms.Error != nil {
			return false, alarms.Error
		}
		for _, al := range alarms.Data {
			alarm := al.(*mqlAwsCloudwatchMetricsalarm)
			actions := alarm.GetActions()
			if actions.Error != nil {
				return false, actions.Error
			}
			for _, ac := range actions.Data {
				topic := ac.(*mqlAwsSnsTopic)
				arn := topic.GetArn()
				if arn.Error != nil {
					return false, arn.Error
				}
				if !strings.HasPrefix(arn.Data, "arn:aws:sns:") {
					continue
				}
				subs := topic.GetSubscriptions()
				if subs.Error != nil {
					return false, subs.Error
				}
				for _, s := range subs.Data {
					sub := s.(*mqlAwsSnsSubscription)
					subArn := sub.GetArn()
					if subArn.Error != nil {
						return false, subArn.Error
					}
					if isActiveSubscriptionArn(subArn.Data) {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}
