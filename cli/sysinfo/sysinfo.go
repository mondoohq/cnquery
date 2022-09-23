package sysinfo

import (
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/execruntime"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/motorid"
	"go.mondoo.com/cnquery/motor/motorid/hostname"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/local"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/resources/packs/core/networkinterface"
)

type sysInfoConfig struct {
	m *motor.Motor
}

type SystemInfoOption func(t *sysInfoConfig) error

func WithMotor(m *motor.Motor) SystemInfoOption {
	return func(c *sysInfoConfig) error {
		c.m = m
		return nil
	}
}

type SystemInfo struct {
	Version    string             `json:"version,omitempty"`
	Build      string             `json:"build,omitempty"`
	Platform   *platform.Platform `json:"platform,omitempty"`
	IP         string             `json:"ip,omitempty"`
	Hostname   string             `json:"platform_hostname,omitempty"`
	Labels     map[string]string  `json:"labels,omitempty"`
	PlatformId string             `json:"platform_id,omitempty"`
}

func GatherSystemInfo(opts ...SystemInfoOption) (*SystemInfo, error) {
	cfg := &sysInfoConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.m == nil {
		provider, err := local.New()
		if err != nil {
			return nil, err
		}

		m, err := motor.New(provider)
		if err != nil {
			return nil, err
		}
		cfg.m = m
	}

	sysInfo := &SystemInfo{
		Version: cnquery.GetVersion(),
		Build:   cnquery.GetBuild(),
	}

	pi, err := cfg.m.Platform()
	if err == nil {
		sysInfo.Platform = pi

		idDetector := providers.HostnameDetector
		if pi.IsFamily(platform.FAMILY_WINDOWS) {
			idDetector = providers.MachineIdDetector
		}

		platformIDs, _, err := motorid.GatherPlatformIDs(cfg.m.Provider, pi, idDetector)
		if err == nil && len(platformIDs) > 0 {
			sysInfo.PlatformId = platformIDs[0]
		}
	}

	var ip string
	ipAddr, err := networkinterface.GetOutboundIP()
	if err == nil {
		ip = ipAddr.String()
	}
	sysInfo.IP = ip

	var hn string
	osProvider, isOSProvider := cfg.m.Provider.(os.OperatingSystemProvider)
	pf, err := cfg.m.Platform()
	if isOSProvider && err == nil {
		hn, err = hostname.Hostname(osProvider, pf)
		if err != nil {
			log.Debug().Err(err).Msg("could not determine hostname")
		}
	}
	sysInfo.Hostname = hn

	// detect the execution runtime
	execEnv := execruntime.Detect()
	sysInfo.Labels = map[string]string{
		"environment": execEnv.Id,
	}

	return sysInfo, nil
}
