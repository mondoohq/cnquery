package os

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/motorid/hostname"
	"go.mondoo.com/cnquery/motor/platform"
	"go.mondoo.com/cnquery/motor/providers/container/docker_snapshot"
	"go.mondoo.com/cnquery/motor/providers/os"
	"go.mondoo.com/cnquery/motor/providers/tar"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/core/packages"
	"go.mondoo.com/cnquery/resources/packs/core/platformid"
	"go.mondoo.com/cnquery/resources/packs/os/info"
	"go.mondoo.com/cnquery/resources/packs/os/reboot"
	"go.mondoo.com/cnquery/resources/packs/os/systemd"
	"go.mondoo.com/cnquery/resources/packs/os/uptime"
	"go.mondoo.com/cnquery/resources/packs/os/windows"
)

var Registry = info.Registry

func init() {
	Init(Registry)
	Registry.Add(core.Registry)
}

func osProvider(motor *motor.Motor) (os.OperatingSystemProvider, error) {
	provider, ok := motor.Provider.(os.OperatingSystemProvider)
	if !ok {
		return nil, fmt.Errorf("provider is not an operating system provider")
	}
	return provider, nil
}

func (p *mqlOs) id() (string, error) {
	return "os", nil
}

func (p *mqlOs) GetRebootpending() (interface{}, error) {
	// it is a container image, a reboot is never required
	switch p.MotorRuntime.Motor.Provider.(type) {
	case *docker_snapshot.DockerSnapshotProvider:
		return false, nil
	case *tar.Provider:
		return false, nil
	}

	// check photon
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}
	if pf.Name == "photon" {
		// get installed kernel and check if the found one is running
		mqlKernel, err := p.MotorRuntime.CreateResource("kernel")
		if err != nil {
			return nil, err
		}
		kernel := mqlKernel.(core.Kernel)
		kernelInstalled, err := kernel.Installed()
		if err != nil {
			return nil, err
		}

		kernels := []core.KernelVersion{}
		data, err := json.Marshal(kernelInstalled)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(data), &kernels)
		if err != nil {
			return nil, err
		}

		// we should only have one kernel here
		if len(kernels) != 1 {
			return nil, errors.New("unexpected kernel list result for photon os")
		}

		return !kernels[0].Running, nil
	}

	// TODO: move more logic into MQL to leverage its cache
	// try to collect if a reboot is required, fails for static images
	rb, err := reboot.New(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}
	return rb.RebootPending()
}

func (p *mqlOs) getUnixEnv() (map[string]interface{}, error) {
	rawCmd, err := p.MotorRuntime.CreateResource("command", "command", "env")
	if err != nil {
		return nil, err
	}
	cmd := rawCmd.(Command)

	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	lines := strings.Split(out, "\n")
	for i := range lines {
		parts := strings.SplitN(lines[i], "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[parts[0]] = parts[1]
	}

	return res, nil
}

func (p *mqlOs) getWindowsEnv() (map[string]interface{}, error) {
	rawCmd, err := p.MotorRuntime.CreateResource("powershell",
		"script", "Get-ChildItem Env:* | ConvertTo-Json",
	)
	if err != nil {
		return nil, err
	}
	cmd := rawCmd.(Powershell)

	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	return windows.ParseEnv(strings.NewReader(out))
}

func (p *mqlOs) GetEnv() (map[string]interface{}, error) {
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if pf.IsFamily("windows") {
		return p.getWindowsEnv()
	}

	return p.getUnixEnv()
}

func (p *mqlOs) GetPath() ([]interface{}, error) {
	env, err := p.Env()
	if err != nil {
		return nil, err
	}

	rawPath, ok := env["PATH"]
	if !ok {
		return []interface{}{}, nil
	}

	path := rawPath.(string)
	parts := strings.Split(path, ":")
	res := make([]interface{}, len(parts))
	for i := range parts {
		res[i] = parts[i]
	}

	return res, nil
}

// returns uptime in nanoseconds
func (p *mqlOs) GetUptime() (*time.Time, error) {
	uptime, err := uptime.New(p.MotorRuntime.Motor)
	if err != nil {
		return core.MqlTime(llx.DurationToTime(0)), err
	}

	t, err := uptime.Duration()
	if err != nil {
		return core.MqlTime(llx.DurationToTime(0)), err
	}

	// we get nano seconds but duration to time only takes seconds
	bootTime := time.Now().Add(-t)
	up := time.Now().Unix() - bootTime.Unix()
	return core.MqlTime(llx.DurationToTime(up)), nil
}

func (p *mqlOsUpdate) id() (string, error) {
	name, _ := p.Name()
	return name, nil
}

func (p *mqlOs) GetUpdates() ([]interface{}, error) {
	// find suitable system updates
	um, err := packages.ResolveSystemUpdateManager(p.MotorRuntime.Motor)
	if um == nil || err != nil {
		return nil, fmt.Errorf("could not detect suiteable update manager for platform")
	}

	// retrieve all system updates
	updates, err := um.List()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve updates list for platform")
	}

	// create MQL update resources for each update
	osupdates := make([]interface{}, len(updates))
	log.Debug().Int("updates", len(updates)).Msg("mql[updates]> found system updates")
	for i, update := range updates {

		mqlOsUpdate, err := p.MotorRuntime.CreateResource("os.update",
			"name", update.Name,
			"severity", update.Severity,
			"category", update.Category,
			"restart", update.Restart,
			"format", update.Format,
		)
		if err != nil {
			return nil, err
		}

		osupdates[i] = mqlOsUpdate.(OsUpdate)
	}

	// return the packages as new entries
	return osupdates, nil
}

func (s *mqlOs) GetHostname() (string, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	return hostname.Hostname(osProvider, platform)
}

func (p *mqlOs) GetName() (string, error) {
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", err
	}

	if !pf.IsFamily(platform.FAMILY_UNIX) && !pf.IsFamily(platform.FAMILY_WINDOWS) {
		return "", errors.New("your platform is not supported by operating system resource")
	}

	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	if pf.IsFamily(platform.FAMILY_LINUX) {
		lf, err := p.MotorRuntime.CreateResource("file", "path", "/etc/machine-info")
		if err != nil {
			return "", err
		}
		file := lf.(core.File)

		exists, err := file.Exists()
		if err != nil {
			return "", err
		}
		// if the file does not exist, the pretty hostname is just empty
		if !exists {
			return "", nil
		}

		err = p.MotorRuntime.WatchAndCompute(file, "content", p, "name")
		if err != nil {
			return "", err
		}

		// gather content
		data, err := file.Content()
		if err != nil {
			return "", err
		}

		mi, err := systemd.ParseMachineInfo(strings.NewReader(data))
		if err != nil {
			return "", err
		}

		if mi.PrettyHostname != "" {
			return mi.PrettyHostname, nil
		}
	}

	// return plain hostname, this also happens for linux if no pretty name was found
	if pf.IsFamily(platform.FAMILY_UNIX) {
		return hostname.Hostname(osProvider, pf)
	}

	if pf.IsFamily(platform.FAMILY_WINDOWS) {

		// try to get the computer name from env
		env, err := p.getWindowsEnv()
		if err == nil {
			val, ok := env["COMPUTERNAME"]
			if ok {
				return val.(string), nil
			}
		}

		// fallback to hostname
		return hostname.Hostname(osProvider, pf)
	}

	return "", errors.New("your platform is not supported by operating system resource")
}

// returns the OS native machine UUID/GUID
func (s *mqlOs) GetMachineid() (string, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	uuidProvider, err := platformid.MachineIDProvider(osProvider, platform)
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid")
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	return id, nil
}

func (p *mqlOsBase) id() (string, error) {
	ident := p.MotorRuntime.Asset.GetMrn()
	if ident == "" {
		ident = strings.Join(p.MotorRuntime.Asset.PlatformIds, ",")
	}
	return "os.base(" + ident + ")", nil
}

func (p *mqlOsBase) GetRebootpending() (interface{}, error) {
	// it is a container image, a reboot is never required
	switch p.MotorRuntime.Motor.Provider.(type) {
	case *docker_snapshot.DockerSnapshotProvider:
		return false, nil
	case *tar.Provider:
		return false, nil
	}

	// check photon
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}
	if pf.Name == "photon" {
		// get installed kernel and check if the found one is running
		mqlKernel, err := p.MotorRuntime.CreateResource("kernel")
		if err != nil {
			return nil, err
		}
		kernel := mqlKernel.(core.Kernel)
		kernelInstalled, err := kernel.Installed()
		if err != nil {
			return nil, err
		}

		kernels := []core.KernelVersion{}
		data, err := json.Marshal(kernelInstalled)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(data), &kernels)
		if err != nil {
			return nil, err
		}

		// we should only have one kernel here
		if len(kernels) != 1 {
			return nil, errors.New("unexpected kernel list result for photon os")
		}

		return !kernels[0].Running, nil
	}

	// TODO: move more logic into MQL to leverage its cache
	// try to collect if a reboot is required, fails for static images
	rb, err := reboot.New(p.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}
	return rb.RebootPending()
}

func (p *mqlOsBase) getUnixEnv() (map[string]interface{}, error) {
	rawCmd, err := p.MotorRuntime.CreateResource("command", "command", "env")
	if err != nil {
		return nil, err
	}
	cmd := rawCmd.(Command)

	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	lines := strings.Split(out, "\n")
	for i := range lines {
		parts := strings.SplitN(lines[i], "=", 2)
		if len(parts) != 2 {
			continue
		}
		res[parts[0]] = parts[1]
	}

	return res, nil
}

func (p *mqlOsBase) getWindowsEnv() (map[string]interface{}, error) {
	rawCmd, err := p.MotorRuntime.CreateResource("powershell",
		"script", "Get-ChildItem Env:* | ConvertTo-Json",
	)
	if err != nil {
		return nil, err
	}
	cmd := rawCmd.(Powershell)

	out, err := cmd.Stdout()
	if err != nil {
		return nil, err
	}

	return windows.ParseEnv(strings.NewReader(out))
}

func (p *mqlOsBase) GetEnv() (map[string]interface{}, error) {
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if pf.IsFamily("windows") {
		return p.getWindowsEnv()
	}

	return p.getUnixEnv()
}

func (p *mqlOsBase) GetPath() ([]interface{}, error) {
	env, err := p.Env()
	if err != nil {
		return nil, err
	}

	rawPath, ok := env["PATH"]
	if !ok {
		return []interface{}{}, nil
	}

	path := rawPath.(string)
	parts := strings.Split(path, ":")
	res := make([]interface{}, len(parts))
	for i := range parts {
		res[i] = parts[i]
	}

	return res, nil
}

// returns uptime in nanoseconds
func (p *mqlOsBase) GetUptime() (*time.Time, error) {
	uptime, err := uptime.New(p.MotorRuntime.Motor)
	if err != nil {
		return core.MqlTime(llx.DurationToTime(0)), err
	}

	t, err := uptime.Duration()
	if err != nil {
		return core.MqlTime(llx.DurationToTime(0)), err
	}

	// we get nano seconds but duration to time only takes seconds
	bootTime := time.Now().Add(-t)
	up := time.Now().Unix() - bootTime.Unix()
	return core.MqlTime(llx.DurationToTime(up)), nil
}

func (p *mqlOsBase) GetUpdates() ([]interface{}, error) {
	// find suitable system updates
	um, err := packages.ResolveSystemUpdateManager(p.MotorRuntime.Motor)
	if um == nil || err != nil {
		return nil, fmt.Errorf("could not detect suiteable update manager for platform")
	}

	// retrieve all system updates
	updates, err := um.List()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve updates list for platform")
	}

	// create MQL update resources for each update
	osupdates := make([]interface{}, len(updates))
	log.Debug().Int("updates", len(updates)).Msg("mql[updates]> found system updates")
	for i, update := range updates {

		mqlOsUpdate, err := p.MotorRuntime.CreateResource("os.update",
			"name", update.Name,
			"severity", update.Severity,
			"category", update.Category,
			"restart", update.Restart,
			"format", update.Format,
		)
		if err != nil {
			return nil, err
		}

		osupdates[i] = mqlOsUpdate.(OsUpdate)
	}

	// return the packages as new entries
	return osupdates, nil
}

func (s *mqlOsBase) GetHostname() (string, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	return hostname.Hostname(osProvider, platform)
}

func (p *mqlOsBase) GetName() (string, error) {
	pf, err := p.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", err
	}

	if !pf.IsFamily(platform.FAMILY_UNIX) && !pf.IsFamily(platform.FAMILY_WINDOWS) {
		return "", errors.New("your platform is not supported by operating system resource")
	}

	osProvider, err := osProvider(p.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	if pf.IsFamily(platform.FAMILY_LINUX) {
		lf, err := p.MotorRuntime.CreateResource("file", "path", "/etc/machine-info")
		if err != nil {
			return "", err
		}
		file := lf.(core.File)

		exists, err := file.Exists()
		if err != nil {
			return "", err
		}
		// if the file does not exist, the pretty hostname is just empty
		if !exists {
			return "", nil
		}

		err = p.MotorRuntime.WatchAndCompute(file, "content", p, "name")
		if err != nil {
			return "", err
		}

		// gather content
		data, err := file.Content()
		if err != nil {
			return "", err
		}

		mi, err := systemd.ParseMachineInfo(strings.NewReader(data))
		if err != nil {
			return "", err
		}

		if mi.PrettyHostname != "" {
			return mi.PrettyHostname, nil
		}
	}

	// return plain hostname, this also happens for linux if no pretty name was found
	if pf.IsFamily(platform.FAMILY_UNIX) {
		return hostname.Hostname(osProvider, pf)
	}

	if pf.IsFamily(platform.FAMILY_WINDOWS) {

		// try to get the computer name from env
		env, err := p.getWindowsEnv()
		if err == nil {
			val, ok := env["COMPUTERNAME"]
			if ok {
				return val.(string), nil
			}
		}

		// fallback to hostname
		return hostname.Hostname(osProvider, pf)
	}

	return "", errors.New("your platform is not supported by operating system resource")
}

// returns the OS native machine UUID/GUID
func (s *mqlOsBase) GetMachineid() (string, error) {
	platform, err := s.MotorRuntime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	osProvider, err := osProvider(s.MotorRuntime.Motor)
	if err != nil {
		return "", err
	}

	uuidProvider, err := platformid.MachineIDProvider(osProvider, platform)
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid")
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.Wrap(err, "cannot determine platform uuid")
	}

	return id, nil
}

func (s *mqlOsBase) GetMachine() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("machine")
}

func (s *mqlOsBase) GetGroups() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("groups")
}

func (s *mqlOsBase) GetUsers() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("users")
}

func (s *mqlOsUnix) id() (string, error) {
	ident := s.MotorRuntime.Asset.GetMrn()
	if ident == "" {
		ident = strings.Join(s.MotorRuntime.Asset.PlatformIds, ",")
	}
	return "os.unix(" + ident + ")", nil
}

func (s *mqlOsUnix) GetBase() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("os.base")
}

func (s *mqlOsLinux) id() (string, error) {
	ident := s.MotorRuntime.Asset.GetMrn()
	if ident == "" {
		ident = strings.Join(s.MotorRuntime.Asset.PlatformIds, ",")
	}
	return "os.linux(" + ident + ")", nil
}

func (s *mqlOsLinux) GetUnix() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("os.unix")
}

func (s *mqlOsLinux) GetIptables() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("iptables")
}

func (s *mqlOsLinux) GetIp6tables() (resources.ResourceType, error) {
	return s.MotorRuntime.CreateResource("ip6tables")
}
