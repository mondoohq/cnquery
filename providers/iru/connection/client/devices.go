// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package client

import (
	"encoding/json"
	"net/url"
	"time"
)

// Device is the summary row returned by GET /v1/devices.
type Device struct {
	DeviceID         string   `json:"device_id"`
	DeviceName       string   `json:"device_name"`
	Model            string   `json:"model"`
	SerialNumber     string   `json:"serial_number"`
	Platform         string   `json:"platform"`
	OSVersion        string   `json:"os_version"`
	LastCheckIn      string   `json:"last_check_in"`
	User             *User    `json:"user,omitempty"`
	AssetTag         string   `json:"asset_tag"`
	AgentInstallDate string   `json:"agent_install_date"`
	BlueprintID      string   `json:"blueprint_id"`
	BlueprintName    string   `json:"blueprint_name"`
	MDMEnabled       string   `json:"mdm_enabled"`
	AgentVersion     string   `json:"agent_version"`
	Tags             []string `json:"tags"`
	UDID             string   `json:"udid"`
}

// DeviceDetails is the rich response from GET /v1/devices/{id}/details. The
// API returns a sprawling nested object — we expose the subsections we map
// to typed fields and leave the rest available via raw JSON where useful.
type DeviceDetails struct {
	General           DeviceDetailsGeneral        `json:"general"`
	MDM               DeviceDetailsMDM            `json:"mdm"`
	ActivationLock    DeviceDetailsActivationLock `json:"activation_lock"`
	HardwareOverview  DeviceDetailsHardware       `json:"hardware_overview"`
	Volumes           []DeviceDetailsVolume       `json:"volumes"`
	FileVault         DeviceDetailsFileVault      `json:"filevault"`
	SecurityInfo      DeviceDetailsSecurityInfo   `json:"security_information"`
	InstalledProfiles []DeviceDetailsProfile      `json:"installed_profiles"`
	Users             DeviceDetailsUsers          `json:"users"`
	NetworkInfo       DeviceDetailsNetwork        `json:"network"`
}

type DeviceDetailsGeneral struct {
	DeviceID       string `json:"device_id"`
	DeviceName     string `json:"device_name"`
	AssetTag       string `json:"asset_tag"`
	LastCheckIn    string `json:"last_check_in"`
	LastEnrollment string `json:"last_enrollment"`
	IsSupervised   bool   `json:"supervised"`
	IsRemovable    bool   `json:"is_removable"`
}

type DeviceDetailsMDM struct {
	MDMEnabled             bool   `json:"mdm_enabled"`
	UserApprovedMDM        bool   `json:"user_approved_mdm"`
	DEPEnrolled            bool   `json:"dep_enrolled"`
	InstalledFromDEP       bool   `json:"installed_from_dep"`
	EnrollmentDate         string `json:"enrollment_date"`
	BootstrapTokenEscrowed bool   `json:"bootstrap_token_escrowed"`
}

type DeviceDetailsActivationLock struct {
	BypassCodePresent     bool `json:"bypass_code_present"`
	ActivationLockEnabled bool `json:"activation_lock_enabled"`
}

type DeviceDetailsHardware struct {
	Model             string `json:"model"`
	ModelIdentifier   string `json:"model_identifier"`
	ProcessorName     string `json:"processor_name"`
	ProcessorSpeed    string `json:"processor_speed"`
	NumberOfCores     int    `json:"number_of_cores"`
	MemoryBytes       int64  `json:"memory_bytes"`
	SerialNumber      string `json:"serial_number"`
	UDID              string `json:"udid"`
	BluetoothMAC      string `json:"bluetooth_mac"`
	WifiMAC           string `json:"wifi_mac"`
	EthernetMAC       string `json:"ethernet_mac"`
	IsAppleSilicon    bool   `json:"is_apple_silicon"`
	BatteryCycleCount int    `json:"battery_cycle_count"`
}

type DeviceDetailsVolume struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
	SizeBytes  int64  `json:"size_bytes"`
	FreeBytes  int64  `json:"free_bytes"`
	Encrypted  bool   `json:"encrypted"`
	BootVolume bool   `json:"boot_volume"`
}

type DeviceDetailsFileVault struct {
	Enabled             bool   `json:"enabled"`
	RecoveryKeyEscrowed bool   `json:"recovery_key_escrowed"`
	RecoveryKeyType     string `json:"recovery_key_type"`
}

type DeviceDetailsSecurityInfo struct {
	GatekeeperEnabled      bool   `json:"gatekeeper_enabled"`
	SIPEnabled             bool   `json:"sip_enabled"`
	FirewallEnabled        bool   `json:"firewall_enabled"`
	FirewallStealthMode    bool   `json:"firewall_stealth_mode_enabled"`
	XProtectVersion        string `json:"xprotect_version"`
	MRTVersion             string `json:"mrt_version"`
	AutoUpdateEnabled      bool   `json:"auto_update_enabled"`
	RemoteDesktopEnabled   bool   `json:"remote_desktop_enabled"`
	ScreensaverLockEnabled bool   `json:"screensaver_lock_enabled"`
	FindMyEnabled          bool   `json:"find_my_enabled"`
	SecureBootLevel        string `json:"secure_boot_level"`
}

type DeviceDetailsProfile struct {
	Identifier   string `json:"identifier"`
	DisplayName  string `json:"display_name"`
	Organization string `json:"organization"`
	UUID         string `json:"uuid"`
	InstalledBy  string `json:"installed_by"`
	IsRemovable  bool   `json:"is_removable"`
	IsEncrypted  bool   `json:"is_encrypted"`
}

type DeviceDetailsUsers struct {
	RegularUsers []DeviceDetailsLocalUser `json:"regular_users"`
}

type DeviceDetailsLocalUser struct {
	Username       string `json:"username"`
	FullName       string `json:"full_name"`
	UID            string `json:"uid"`
	IsAdmin        bool   `json:"is_admin"`
	HomePath       string `json:"home_path"`
	HasSecureToken bool   `json:"has_secure_token"`
}

type DeviceDetailsNetwork struct {
	IPAddress     string `json:"ip_address"`
	PublicIP      string `json:"public_ip"`
	HostName      string `json:"host_name"`
	LocalHostname string `json:"local_hostname"`
}

// App is one row from GET /v1/devices/{id}/apps.
type App struct {
	BundleID     string `json:"bundle_id"`
	Name         string `json:"app_name"`
	Version      string `json:"version"`
	ShortVersion string `json:"short_version"`
	Path         string `json:"path"`
	TeamID       string `json:"team_identifier"`
	IsAppleApp   bool   `json:"is_apple_app"`
}

// ListDevices walks /v1/devices with limit/offset pagination.
func (c *Client) ListDevices() ([]Device, error) {
	var all []Device
	err := c.paginate("/api/v1/devices", func(raw json.RawMessage) (int, error) {
		var page []Device
		if err := json.Unmarshal(raw, &page); err != nil {
			return 0, err
		}
		all = append(all, page...)
		return len(page), nil
	})
	if err != nil {
		return nil, err
	}
	return all, nil
}

// GetDeviceDetails fetches /v1/devices/{id}/details.
func (c *Client) GetDeviceDetails(id string) (*DeviceDetails, error) {
	var out DeviceDetails
	if err := c.do("/api/v1/devices/"+url.PathEscape(id)+"/details", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDeviceApps fetches /v1/devices/{id}/apps. The endpoint returns either a
// bare array of apps or an envelope with `apps`/`results`; we tolerate both.
func (c *Client) GetDeviceApps(id string) ([]App, error) {
	var raw json.RawMessage
	if err := c.do("/api/v1/devices/"+url.PathEscape(id)+"/apps", nil, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] == '[' {
		var apps []App
		if err := json.Unmarshal(raw, &apps); err != nil {
			return nil, err
		}
		return apps, nil
	}
	var envelope struct {
		Apps    []App `json:"apps"`
		Results []App `json:"results"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Apps != nil {
		return envelope.Apps, nil
	}
	return envelope.Results, nil
}

// ParseTime parses an Iru timestamp; returns nil on empty/invalid input.
func ParseTime(s string) *time.Time {
	if s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000000Z"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}
