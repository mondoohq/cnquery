// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/settings"
)

// mqlDatabricksWorkspaceConfInternal caches the workspace security-hardening
// settings. Each toggle is a distinct settings API call, so they are fetched
// once together and shared across the computed fields that expose them.
type mqlDatabricksWorkspaceConfInternal struct {
	hardeningOnce sync.Once
	hardening     workspaceHardening
	hardeningErr  error
}

// workspaceHardening holds the resolved workspace security-hardening toggles.
// A per-setting API call that fails (for example because the setting is not
// available on the workspace tier) leaves that value at its zero state, which
// reads as "not hardened" for audit purposes.
type workspaceHardening struct {
	cspEnabled          bool
	cspStandards        []any
	esmEnabled          bool
	restrictAdminStatus string
	autoClusterUpdate   bool
	disableLegacyAccess bool
}

// loadHardening fetches the workspace security-hardening settings once and
// caches the result. The plane error (connecting to the account console rather
// than a workspace) is captured so every hardening field reports it
// consistently. Individual setting failures are swallowed and leave the
// corresponding value at its zero state.
func (r *mqlDatabricksWorkspaceConf) loadHardening() (workspaceHardening, error) {
	r.hardeningOnce.Do(func() {
		ws, err := workspaceClient(r.MqlRuntime)
		if err != nil {
			r.hardeningErr = err
			return
		}
		ctx := context.Background()
		h := workspaceHardening{cspStandards: []any{}}

		if csp, err := ws.Settings.ComplianceSecurityProfile().Get(ctx, settings.GetComplianceSecurityProfileSettingRequest{}); err == nil && csp != nil {
			h.cspEnabled = csp.ComplianceSecurityProfileWorkspace.IsEnabled
			for _, std := range csp.ComplianceSecurityProfileWorkspace.ComplianceStandards {
				h.cspStandards = append(h.cspStandards, string(std))
			}
		}

		if esm, err := ws.Settings.EnhancedSecurityMonitoring().Get(ctx, settings.GetEnhancedSecurityMonitoringSettingRequest{}); err == nil && esm != nil {
			h.esmEnabled = esm.EnhancedSecurityMonitoringWorkspace.IsEnabled
		}

		if rwa, err := ws.Settings.RestrictWorkspaceAdmins().Get(ctx, settings.GetRestrictWorkspaceAdminsSettingRequest{}); err == nil && rwa != nil {
			h.restrictAdminStatus = string(rwa.RestrictWorkspaceAdmins.Status)
		}

		if acu, err := ws.Settings.AutomaticClusterUpdate().Get(ctx, settings.GetAutomaticClusterUpdateSettingRequest{}); err == nil && acu != nil {
			h.autoClusterUpdate = acu.AutomaticClusterUpdateWorkspace.Enabled
		}

		if dla, err := ws.Settings.DisableLegacyAccess().Get(ctx, settings.GetDisableLegacyAccessRequest{}); err == nil && dla != nil {
			h.disableLegacyAccess = dla.DisableLegacyAccess.Value
		}

		r.hardening = h
	})
	return r.hardening, r.hardeningErr
}

func (r *mqlDatabricksWorkspaceConf) complianceSecurityProfileEnabled() (bool, error) {
	h, err := r.loadHardening()
	if err != nil {
		return false, err
	}
	return h.cspEnabled, nil
}

func (r *mqlDatabricksWorkspaceConf) complianceSecurityStandards() ([]any, error) {
	h, err := r.loadHardening()
	if err != nil {
		return nil, err
	}
	return h.cspStandards, nil
}

func (r *mqlDatabricksWorkspaceConf) enhancedSecurityMonitoringEnabled() (bool, error) {
	h, err := r.loadHardening()
	if err != nil {
		return false, err
	}
	return h.esmEnabled, nil
}

func (r *mqlDatabricksWorkspaceConf) restrictWorkspaceAdminsStatus() (string, error) {
	h, err := r.loadHardening()
	if err != nil {
		return "", err
	}
	return h.restrictAdminStatus, nil
}

func (r *mqlDatabricksWorkspaceConf) automaticClusterUpdateEnabled() (bool, error) {
	h, err := r.loadHardening()
	if err != nil {
		return false, err
	}
	return h.autoClusterUpdate, nil
}

func (r *mqlDatabricksWorkspaceConf) disableLegacyAccess() (bool, error) {
	h, err := r.loadHardening()
	if err != nil {
		return false, err
	}
	return h.disableLegacyAccess, nil
}
