// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"errors"
	"strconv"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/os/connection/shared"
	"go.mondoo.com/mql/v13/providers/os/resources/windows"
	"go.mondoo.com/mql/v13/types"
)

// enabled reports whether the Defender service and its real-time antivirus and
// antispyware protection are all active. It resolves to false (not an error)
// when Defender is unavailable on the target.
func (d *mqlWindowsDefender) enabled() (bool, error) {
	statusRes := d.GetStatus()
	if statusRes.Error != nil {
		if errors.Is(statusRes.Error, windows.ErrDefenderUnavailable) {
			return false, nil
		}
		return false, statusRes.Error
	}
	s := statusRes.Data
	if s == nil {
		return false, nil
	}
	return s.AmServiceEnabled.Data && s.AntivirusEnabled.Data && s.AntispywareEnabled.Data, nil
}

func (d *mqlWindowsDefender) status() (*mqlWindowsDefenderStatus, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)

	status, err := windows.GetDefenderComputerStatus(conn)
	if err != nil {
		if errors.Is(err, windows.ErrDefenderUnavailable) {
			d.Status.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(d.MqlRuntime, "windows.defender.status", map[string]*llx.RawData{
		"__id":                             llx.StringData("windows.defender.status"),
		"amEngineVersion":                  llx.StringData(status.AMEngineVersion),
		"amProductVersion":                 llx.StringData(status.AMProductVersion),
		"amServiceEnabled":                 llx.BoolData(status.AMServiceEnabled),
		"amServiceVersion":                 llx.StringData(status.AMServiceVersion),
		"antispywareEnabled":               llx.BoolData(status.AntispywareEnabled),
		"antispywareSignatureAge":          llx.IntData(status.AntispywareSignatureAge),
		"antispywareSignatureLastUpdated":  llx.TimeDataPtr(windows.DefenderTime(status.AntispywareSignatureLastUpdated)),
		"antispywareSignatureVersion":      llx.StringData(status.AntispywareSignatureVersion),
		"antivirusEnabled":                 llx.BoolData(status.AntivirusEnabled),
		"antivirusSignatureAge":            llx.IntData(status.AntivirusSignatureAge),
		"antivirusSignatureLastUpdated":    llx.TimeDataPtr(windows.DefenderTime(status.AntivirusSignatureLastUpdated)),
		"antivirusSignatureVersion":        llx.StringData(status.AntivirusSignatureVersion),
		"behaviorMonitorEnabled":           llx.BoolData(status.BehaviorMonitorEnabled),
		"computerID":                       llx.StringData(status.ComputerID),
		"computerState":                    llx.IntData(status.ComputerState),
		"defenderSignaturesOutOfDate":      llx.BoolData(status.DefenderSignaturesOutOfDate),
		"deviceControlDefaultEnforcement":  llx.IntData(status.DeviceControlDefaultEnforcement),
		"deviceControlPoliciesLastUpdated": llx.TimeDataPtr(windows.DefenderTime(status.DeviceControlPoliciesLastUpdated)),
		"deviceControlState":               llx.IntData(status.DeviceControlState),
		"fullScanAge":                      llx.IntData(status.FullScanAge),
		"fullScanStartTime":                llx.TimeDataPtr(windows.DefenderTime(status.FullScanStartTime)),
		"fullScanEndTime":                  llx.TimeDataPtr(windows.DefenderTime(status.FullScanEndTime)),
		"fullScanOverdue":                  llx.BoolData(status.FullScanOverdue),
		"fullScanRequired":                 llx.BoolData(status.FullScanRequired),
		"fullScanSignatureVersion":         llx.StringData(status.FullScanSignatureVersion),
		"ioavProtectionEnabled":            llx.BoolData(status.IoavProtectionEnabled),
		"isTamperProtected":                llx.BoolData(status.IsTamperProtected),
		"isVirtualMachine":                 llx.BoolData(status.IsVirtualMachine),
		"lastFullScanSource":               llx.IntData(status.LastFullScanSource),
		"lastQuickScanSource":              llx.IntData(status.LastQuickScanSource),
		"nisEnabled":                       llx.BoolData(status.NISEnabled),
		"nisEngineVersion":                 llx.StringData(status.NISEngineVersion),
		"nisSignatureAge":                  llx.IntData(status.NISSignatureAge),
		"nisSignatureLastUpdated":          llx.TimeDataPtr(windows.DefenderTime(status.NISSignatureLastUpdated)),
		"nisSignatureVersion":              llx.StringData(status.NISSignatureVersion),
		"onAccessProtectionEnabled":        llx.BoolData(status.OnAccessProtectionEnabled),
		"productStatus":                    llx.IntData(status.ProductStatus),
		"quickScanAge":                     llx.IntData(status.QuickScanAge),
		"quickScanStartTime":               llx.TimeDataPtr(windows.DefenderTime(status.QuickScanStartTime)),
		"quickScanEndTime":                 llx.TimeDataPtr(windows.DefenderTime(status.QuickScanEndTime)),
		"quickScanOverdue":                 llx.BoolData(status.QuickScanOverdue),
		"quickScanSignatureVersion":        llx.StringData(status.QuickScanSignatureVersion),
		"realTimeProtectionEnabled":        llx.BoolData(status.RealTimeProtectionEnabled),
		"realTimeScanDirection":            llx.IntData(status.RealTimeScanDirection),
		"rebootRequired":                   llx.BoolData(status.RebootRequired),
		"smartAppControlState":             llx.StringData(status.SmartAppControlState),
		"smartAppControlExpiration":        llx.TimeDataPtr(windows.DefenderTime(status.SmartAppControlExpiration)),
		"tamperProtectionSource":           llx.StringData(status.TamperProtectionSource),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderStatus), nil
}

func (d *mqlWindowsDefender) preferences() (*mqlWindowsDefenderPreferences, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)

	// Fetch Get-MpPreference up front so we can resolve the whole preferences
	// resource to null when Defender is unavailable, rather than letting each
	// sub-accessor surface a raw ErrDefenderUnavailable.
	prefs, err := windows.GetDefenderPreferences(conn)
	if err != nil {
		if errors.Is(err, windows.ErrDefenderUnavailable) {
			d.Preferences.State = plugin.StateIsSet | plugin.StateIsNull
			return nil, nil
		}
		return nil, err
	}

	res, err := CreateResource(d.MqlRuntime, "windows.defender.preferences", map[string]*llx.RawData{
		"__id": llx.StringData("windows.defender.preferences"),
	})
	if err != nil {
		return nil, err
	}

	// Seed the cache so the grouped sub-accessors reuse this result instead of
	// running Get-MpPreference a second time.
	p := res.(*mqlWindowsDefenderPreferences)
	p.prefs = prefs
	p.fetched = true
	return p, nil
}

func (d *mqlWindowsDefender) threats() ([]any, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)

	threats, err := windows.GetDefenderThreats(conn)
	if err != nil {
		if errors.Is(err, windows.ErrDefenderUnavailable) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(threats))
	for i := range threats {
		t := threats[i]
		mqlThreat, err := CreateResource(d.MqlRuntime, "windows.defender.threat", map[string]*llx.RawData{
			"__id":             llx.StringData("windows.defender.threat/" + strconv.FormatInt(t.ThreatID, 10)),
			"threatId":         llx.IntData(t.ThreatID),
			"name":             llx.StringData(t.ThreatName),
			"severityID":       llx.IntData(t.SeverityID),
			"categoryID":       llx.IntData(t.CategoryID),
			"isActive":         llx.BoolData(t.IsActive),
			"didThreatExecute": llx.BoolData(t.DidThreatExecute),
			"rollupStatus":     llx.IntData(t.RollupStatus),
			"resources":        llx.ArrayData(convert.SliceAnyToInterface(t.Resources), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlThreat)
	}
	return res, nil
}

func (d *mqlWindowsDefender) threatDetections() ([]any, error) {
	conn := d.MqlRuntime.Connection.(shared.Connection)

	detections, err := windows.GetDefenderThreatDetections(conn)
	if err != nil {
		if errors.Is(err, windows.ErrDefenderUnavailable) {
			return []any{}, nil
		}
		return nil, err
	}

	res := make([]any, 0, len(detections))
	for i := range detections {
		det := detections[i]
		mqlDet, err := CreateResource(d.MqlRuntime, "windows.defender.threatDetection", map[string]*llx.RawData{
			"__id":                           llx.StringData("windows.defender.threatDetection/" + det.DetectionID),
			"detectionId":                    llx.StringData(det.DetectionID),
			"threatId":                       llx.IntData(det.ThreatID),
			"processName":                    llx.StringData(det.ProcessName),
			"domainUser":                     llx.StringData(det.DomainUser),
			"detectionSourceTypeId":          llx.IntData(det.DetectionSourceTypeID),
			"currentThreatExecutionStatusID": llx.IntData(det.CurrentThreatExecutionStatusID),
			"threatStatusID":                 llx.IntData(det.ThreatStatusID),
			"cleaningActionID":               llx.IntData(det.CleaningActionID),
			"actionSuccess":                  llx.BoolData(det.ActionSuccess),
			"initialDetectionTime":           llx.TimeDataPtr(windows.DefenderTime(det.InitialDetectionTime)),
			"lastThreatStatusChangeTime":     llx.TimeDataPtr(windows.DefenderTime(det.LastThreatStatusChangeTime)),
			"remediationTime":                llx.TimeDataPtr(windows.DefenderTime(det.RemediationTime)),
			"resources":                      llx.ArrayData(convert.SliceAnyToInterface(det.Resources), types.String),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlDet)
	}
	return res, nil
}

// mqlWindowsDefenderPreferencesInternal caches the Get-MpPreference result so
// that the grouped accessors (scan, cloudProtection, ...) share a single query.
type mqlWindowsDefenderPreferencesInternal struct {
	lock    sync.Mutex
	fetched bool
	prefs   *windows.MpPreference
	err     error
}

// getPrefs fetches Get-MpPreference once and caches the result.
func (p *mqlWindowsDefenderPreferences) getPrefs() (*windows.MpPreference, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.fetched {
		return p.prefs, p.err
	}
	conn := p.MqlRuntime.Connection.(shared.Connection)
	p.prefs, p.err = windows.GetDefenderPreferences(conn)
	p.fetched = true
	return p.prefs, p.err
}

func (p *mqlWindowsDefenderPreferences) scan() (*mqlWindowsDefenderScanSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.scanSettings", map[string]*llx.RawData{
		"__id":                                          llx.StringData("windows.defender.scanSettings"),
		"scanParameters":                                llx.IntData(prefs.ScanParameters),
		"scanScheduleDay":                               llx.IntData(prefs.ScanScheduleDay),
		"scanScheduleTime":                              llx.StringData(prefs.ScanScheduleTimeString()),
		"scanScheduleQuickScanTime":                     llx.StringData(prefs.ScanScheduleQuickScanTimeString()),
		"scanScheduleOffset":                            llx.IntData(prefs.ScanScheduleOffset),
		"scanAvgCPULoadFactor":                          llx.IntData(prefs.ScanAvgCPULoadFactor),
		"scanOnlyIfIdleEnabled":                         llx.BoolData(prefs.ScanOnlyIfIdleEnabled),
		"checkForSignaturesBeforeRunningScan":           llx.IntData(prefs.CheckForSignaturesBeforeRunningScan),
		"disableArchiveScanning":                        llx.BoolData(prefs.DisableArchiveScanning),
		"disableEmailScanning":                          llx.BoolData(prefs.DisableEmailScanning),
		"disableRemovableDriveScanning":                 llx.BoolData(prefs.DisableRemovableDriveScanning),
		"disableScanningMappedNetworkDrivesForFullScan": llx.BoolData(prefs.DisableScanningMappedNetworkDrivesForFullScan),
		"disableScanningNetworkFiles":                   llx.BoolData(prefs.DisableScanningNetworkFiles),
		"disableCatchupFullScan":                        llx.BoolData(prefs.DisableCatchupFullScan),
		"disableCatchupQuickScan":                       llx.BoolData(prefs.DisableCatchupQuickScan),
		"disableCpuThrottleOnIdleScans":                 llx.BoolData(prefs.DisableCpuThrottleOnIdleScans),
		"enableFullScanOnBatteryPower":                  llx.BoolData(prefs.EnableFullScanOnBatteryPower),
		"enableLowCpuPriority":                          llx.BoolData(prefs.EnableLowCpuPriority),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderScanSettings), nil
}

func (p *mqlWindowsDefenderPreferences) realTimeProtection() (*mqlWindowsDefenderRealTimeSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.realTimeSettings", map[string]*llx.RawData{
		"__id":                             llx.StringData("windows.defender.realTimeSettings"),
		"disableRealtimeMonitoring":        llx.BoolData(prefs.DisableRealtimeMonitoring),
		"disableBehaviorMonitoring":        llx.BoolData(prefs.DisableBehaviorMonitoring),
		"disableIOAVProtection":            llx.BoolData(prefs.DisableIOAVProtection),
		"disableScriptScanning":            llx.BoolData(prefs.DisableScriptScanning),
		"disableIntrusionPreventionSystem": llx.BoolData(prefs.DisableIntrusionPreventionSystem),
		"realTimeScanDirection":            llx.IntData(prefs.RealTimeScanDirection),
		"enableFileHashComputation":        llx.BoolData(prefs.EnableFileHashComputation),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderRealTimeSettings), nil
}

func (p *mqlWindowsDefenderPreferences) cloudProtection() (*mqlWindowsDefenderCloudSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.cloudSettings", map[string]*llx.RawData{
		"__id":                    llx.StringData("windows.defender.cloudSettings"),
		"mapsReporting":           llx.IntData(prefs.MAPSReporting),
		"submitSamplesConsent":    llx.IntData(prefs.SubmitSamplesConsent),
		"cloudBlockLevel":         llx.IntData(prefs.CloudBlockLevel),
		"cloudExtendedTimeout":    llx.IntData(prefs.CloudExtendedTimeout),
		"disableBlockAtFirstSeen": llx.BoolData(prefs.DisableBlockAtFirstSeen),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderCloudSettings), nil
}

func (p *mqlWindowsDefenderPreferences) signatureUpdates() (*mqlWindowsDefenderSignatureSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.signatureSettings", map[string]*llx.RawData{
		"__id":                                         llx.StringData("windows.defender.signatureSettings"),
		"signatureScheduleDay":                         llx.IntData(prefs.SignatureScheduleDay),
		"signatureScheduleTime":                        llx.StringData(prefs.SignatureScheduleTimeString()),
		"signatureUpdateInterval":                      llx.IntData(prefs.SignatureUpdateInterval),
		"signatureUpdateCatchupInterval":               llx.IntData(prefs.SignatureUpdateCatchupInterval),
		"signatureFallbackOrder":                       llx.StringData(prefs.SignatureFallbackOrder),
		"signatureDefinitionUpdateFileSharesSources":   llx.StringData(prefs.SignatureDefinitionUpdateFileSharesSources),
		"signatureDisableUpdateOnStartupWithoutEngine": llx.BoolData(prefs.SignatureDisableUpdateOnStartupWithoutEngine),
		"signatureFirstAuGracePeriod":                  llx.IntData(prefs.SignatureFirstAuGracePeriod),
		"signatureAuGracePeriod":                       llx.IntData(prefs.SignatureAuGracePeriod),
		"definitionUpdatesChannel":                     llx.IntData(prefs.DefinitionUpdatesChannel),
		"engineUpdatesChannel":                         llx.IntData(prefs.EngineUpdatesChannel),
		"platformUpdatesChannel":                       llx.IntData(prefs.PlatformUpdatesChannel),
		"meteredConnectionUpdates":                     llx.BoolData(prefs.MeteredConnectionUpdates),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderSignatureSettings), nil
}

func (p *mqlWindowsDefenderPreferences) threatActions() (*mqlWindowsDefenderThreatActionSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.threatActionSettings", map[string]*llx.RawData{
		"__id":                        llx.StringData("windows.defender.threatActionSettings"),
		"severeThreatDefaultAction":   llx.IntData(prefs.SevereThreatDefaultAction),
		"highThreatDefaultAction":     llx.IntData(prefs.HighThreatDefaultAction),
		"moderateThreatDefaultAction": llx.IntData(prefs.ModerateThreatDefaultAction),
		"lowThreatDefaultAction":      llx.IntData(prefs.LowThreatDefaultAction),
		"unknownThreatDefaultAction":  llx.IntData(prefs.UnknownThreatDefaultAction),
	})
	if err != nil {
		return nil, err
	}
	mqlRes := res.(*mqlWindowsDefenderThreatActionSettings)
	mqlRes.cacheIds = prefs.ThreatIDDefaultAction_Ids
	mqlRes.cacheActions = prefs.ThreatIDDefaultAction_Actions
	return mqlRes, nil
}

func (p *mqlWindowsDefenderPreferences) controlledFolderAccess() (*mqlWindowsDefenderControlledFolderAccess, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.controlledFolderAccess", map[string]*llx.RawData{
		"__id":                llx.StringData("windows.defender.controlledFolderAccess"),
		"enabled":             llx.IntData(prefs.EnableControlledFolderAccess),
		"allowedApplications": llx.ArrayData(convert.SliceAnyToInterface(prefs.ControlledFolderAccessAllowedApplications), types.String),
		"protectedFolders":    llx.ArrayData(convert.SliceAnyToInterface(prefs.ControlledFolderAccessProtectedFolders), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderControlledFolderAccess), nil
}

func (p *mqlWindowsDefenderPreferences) networkProtection() (*mqlWindowsDefenderNetworkProtectionSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.networkProtectionSettings", map[string]*llx.RawData{
		"__id":                               llx.StringData("windows.defender.networkProtectionSettings"),
		"enableNetworkProtection":            llx.IntData(prefs.EnableNetworkProtection),
		"allowNetworkProtectionOnWinServer":  llx.BoolData(prefs.AllowNetworkProtectionOnWinServer),
		"allowNetworkProtectionDownLevel":    llx.BoolData(prefs.AllowNetworkProtectionDownLevel),
		"allowDatagramProcessingOnWinServer": llx.BoolData(prefs.AllowDatagramProcessingOnWinServer),
		"enableDnsSinkhole":                  llx.BoolData(prefs.EnableDnsSinkhole),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderNetworkProtectionSettings), nil
}

func (p *mqlWindowsDefenderPreferences) behavioralNetworkBlocks() (*mqlWindowsDefenderBehavioralNetworkBlockSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.behavioralNetworkBlockSettings", map[string]*llx.RawData{
		"__id":                                      llx.StringData("windows.defender.behavioralNetworkBlockSettings"),
		"bruteForceProtectionConfiguredState":       llx.IntData(prefs.BruteForceProtectionConfiguredState),
		"bruteForceProtectionAggressiveness":        llx.IntData(prefs.BruteForceProtectionAggressiveness),
		"bruteForceProtectionMaxBlockTime":          llx.IntData(prefs.BruteForceProtectionMaxBlockTime),
		"remoteEncryptionProtectionConfiguredState": llx.IntData(prefs.RemoteEncryptionProtectionConfiguredState),
		"remoteEncryptionProtectionAggressiveness":  llx.IntData(prefs.RemoteEncryptionProtectionAggressiveness),
		"remoteEncryptionProtectionMaxBlockTime":    llx.IntData(prefs.RemoteEncryptionProtectionMaxBlockTime),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderBehavioralNetworkBlockSettings), nil
}

func (p *mqlWindowsDefenderPreferences) localSettingOverrides() (*mqlWindowsDefenderLocalSettingOverrides, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.localSettingOverrides", map[string]*llx.RawData{
		"__id":                             llx.StringData("windows.defender.localSettingOverrides"),
		"spynetReporting":                  llx.BoolData(prefs.LocalSettingOverrideSpynetReporting),
		"realtimeMonitoring":               llx.BoolData(prefs.LocalSettingOverrideRealtimeMonitoring),
		"disableBehaviorMonitoring":        llx.BoolData(prefs.LocalSettingOverrideDisableBehaviorMonitoring),
		"disableIOAVProtection":            llx.BoolData(prefs.LocalSettingOverrideDisableIOAVProtection),
		"disableIntrusionPreventionSystem": llx.BoolData(prefs.LocalSettingOverrideDisableIntrusionPreventionSystem),
		"disableOnAccessProtection":        llx.BoolData(prefs.LocalSettingOverrideDisableOnAccessProtection),
		"scanParameters":                   llx.BoolData(prefs.LocalSettingOverrideScanParameters),
		"scanScheduleDay":                  llx.BoolData(prefs.LocalSettingOverrideScanScheduleDay),
		"avgCPULoadFactor":                 llx.BoolData(prefs.LocalSettingOverrideAvgCPULoadFactor),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderLocalSettingOverrides), nil
}

func (p *mqlWindowsDefenderPreferences) remediation() (*mqlWindowsDefenderRemediationSettings, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.remediationSettings", map[string]*llx.RawData{
		"__id":                           llx.StringData("windows.defender.remediationSettings"),
		"remediationScheduleDay":         llx.IntData(prefs.RemediationScheduleDay),
		"remediationScheduleTime":        llx.StringData(prefs.RemediationScheduleTimeString()),
		"quarantinePurgeItemsAfterDelay": llx.IntData(prefs.QuarantinePurgeItemsAfterDelay),
		"disableRestorePoint":            llx.BoolData(prefs.DisableRestorePoint),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderRemediationSettings), nil
}

func (p *mqlWindowsDefenderPreferences) exclusions() (*mqlWindowsDefenderExclusions, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}
	res, err := CreateResource(p.MqlRuntime, "windows.defender.exclusions", map[string]*llx.RawData{
		"__id":        llx.StringData("windows.defender.exclusions"),
		"paths":       llx.ArrayData(convert.SliceAnyToInterface(prefs.ExclusionPath), types.String),
		"extensions":  llx.ArrayData(convert.SliceAnyToInterface(prefs.ExclusionExtension), types.String),
		"processes":   llx.ArrayData(convert.SliceAnyToInterface(prefs.ExclusionProcess), types.String),
		"ipAddresses": llx.ArrayData(convert.SliceAnyToInterface(prefs.ExclusionIpAddress), types.String),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlWindowsDefenderExclusions), nil
}

func (p *mqlWindowsDefenderPreferences) attackSurfaceReductionRules() ([]any, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return nil, err
	}

	ids := prefs.AttackSurfaceReductionRules_Ids
	actions := prefs.AttackSurfaceReductionRules_Actions

	res := make([]any, 0, len(ids))
	for i := range ids {
		var action int64
		if i < len(actions) {
			action = actions[i]
		}
		mqlRule, err := CreateResource(p.MqlRuntime, "windows.defender.asrRule", map[string]*llx.RawData{
			"__id":   llx.StringData("windows.defender.asrRule/" + ids[i]),
			"id":     llx.StringData(ids[i]),
			"name":   llx.StringData(windows.AttackSurfaceReductionRuleNames[ids[i]]),
			"action": llx.IntData(action),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlRule)
	}
	return res, nil
}

func (p *mqlWindowsDefenderPreferences) puaProtection() (int64, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return 0, err
	}
	return prefs.PUAProtection, nil
}

func (p *mqlWindowsDefenderPreferences) uiLockdown() (bool, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return false, err
	}
	return prefs.UILockdown, nil
}

func (p *mqlWindowsDefenderPreferences) randomizeScheduleTaskTimes() (bool, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return false, err
	}
	return prefs.RandomizeScheduleTaskTimes, nil
}

func (p *mqlWindowsDefenderPreferences) disableAutoExclusions() (bool, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return false, err
	}
	return prefs.DisableAutoExclusions, nil
}

func (p *mqlWindowsDefenderPreferences) disableGenericReports() (bool, error) {
	prefs, err := p.getPrefs()
	if err != nil {
		return false, err
	}
	return prefs.DisableGenericReports, nil
}

// mqlWindowsDefenderThreatActionSettingsInternal caches the per-threat-ID
// action arrays so idActions() can zip them into typed resources.
type mqlWindowsDefenderThreatActionSettingsInternal struct {
	cacheIds     []string
	cacheActions []int64
}

func (t *mqlWindowsDefenderThreatActionSettings) idActions() ([]any, error) {
	res := make([]any, 0, len(t.cacheIds))
	for i := range t.cacheIds {
		var action int64
		if i < len(t.cacheActions) {
			action = t.cacheActions[i]
		}
		mqlAction, err := CreateResource(t.MqlRuntime, "windows.defender.threatIdAction", map[string]*llx.RawData{
			"__id":     llx.StringData("windows.defender.threatIdAction/" + t.cacheIds[i]),
			"threatId": llx.StringData(t.cacheIds[i]),
			"action":   llx.IntData(action),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlAction)
	}
	return res, nil
}
