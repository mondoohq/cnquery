package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	"go.mondoo.io/mondoo/lumi/resources/reboot"
	"go.mondoo.io/mondoo/lumi/resources/systemd"
	"go.mondoo.io/mondoo/motor/platform"
	"go.mondoo.io/mondoo/motor/transports/docker/image"
	"go.mondoo.io/mondoo/motor/transports/docker/snapshot"
	"go.mondoo.io/mondoo/motor/transports/tar"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/llx"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/lumi/resources/platformid"
	"go.mondoo.io/mondoo/lumi/resources/uptime"
	"go.mondoo.io/mondoo/lumi/resources/windows"
	"go.mondoo.io/mondoo/motor/motorid/hostname"
)

func (p *lumiOs) id() (string, error) {
	return "os", nil
}

func (p *lumiOs) GetRebootpending() (interface{}, error) {
	// it is a container image, a reboot is never required
	switch p.Runtime.Motor.Transport.(type) {
	case *image.DockerImageTransport:
		return false, nil
	case *snapshot.DockerSnapshotTransport:
		return false, nil
	case *tar.Transport:
		return false, nil
	}

	// check photon
	pf, err := p.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}
	if pf.Name == "photon" {
		// get installed kernel and check if the found one is running
		lumiKernel, err := p.Runtime.CreateResource("kernel")
		if err != nil {
			return nil, err
		}
		kernel := lumiKernel.(Kernel)
		kernelInstalled, err := kernel.Installed()
		if err != nil {
			return nil, err
		}

		kernels := []KernelVersion{}
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

	// TODO: move more logic into lumi to leverage its cache
	// try to collect if a reboot is required, fails for static images
	rb, err := reboot.New(p.Runtime.Motor)
	if err != nil {
		return nil, err
	}
	return rb.RebootPending()
}

func (p *lumiOs) getUnixEnv() (map[string]interface{}, error) {
	rawCmd, err := p.Runtime.CreateResource("command", "command", "env")
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

func (p *lumiOs) getWindowsEnv() (map[string]interface{}, error) {
	rawCmd, err := p.Runtime.CreateResource("powershell",
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

func (p *lumiOs) GetEnv() (map[string]interface{}, error) {
	pf, err := p.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if pf.IsFamily("windows") {
		return p.getWindowsEnv()
	}

	return p.getUnixEnv()
}

func (p *lumiOs) GetPath() ([]interface{}, error) {
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
func (p *lumiOs) GetUptime() (*time.Time, error) {

	uptime, err := uptime.New(p.Runtime.Motor)
	if err != nil {
		return LumiTime(llx.DurationToTime(0)), err
	}

	t, err := uptime.Duration()
	if err != nil {
		return LumiTime(llx.DurationToTime(0)), err
	}

	// we get nano seconds but duration to time only takes seconds
	bootTime := time.Now().Add(-t)
	up := time.Now().Unix() - bootTime.Unix()
	return LumiTime(llx.DurationToTime(up)), nil
}

func (p *lumiOsUpdate) id() (string, error) {
	name, _ := p.Name()
	return name, nil
}

func (p *lumiOs) GetUpdates() ([]interface{}, error) {
	// find suitable system updates
	um, err := packages.ResolveSystemUpdateManager(p.Runtime.Motor)
	if um == nil || err != nil {
		return nil, fmt.Errorf("could not detect suiteable update manager for platform")
	}

	// retrieve all system updates
	updates, err := um.List()
	if err != nil {
		return nil, fmt.Errorf("could not retrieve updates list for platform")
	}

	// create lumi update resources for each update
	osupdates := make([]interface{}, len(updates))
	log.Debug().Int("updates", len(updates)).Msg("lumi[updates]> found system updates")
	for i, update := range updates {

		lumiOsUpdate, err := p.Runtime.CreateResource("os.update",
			"name", update.Name,
			"severity", update.Severity,
			"category", update.Category,
			"restart", update.Restart,
			"format", um.Format(),
		)
		if err != nil {
			return nil, err
		}

		osupdates[i] = lumiOsUpdate.(OsUpdate)
	}

	// return the packages as new entries
	return osupdates, nil
}

func (s *lumiOs) GetHostname() (string, error) {
	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	return hostname.Hostname(s.Runtime.Motor.Transport, platform)
}

func (p *lumiOs) GetName() (string, error) {
	pf, err := p.Runtime.Motor.Platform()
	if err != nil {
		return "", err
	}

	if !pf.IsFamily(platform.FAMILY_UNIX) && !pf.IsFamily(platform.FAMILY_WINDOWS) {
		return "", errors.New("your platform is not supported by operating system resource")
	}

	if pf.IsFamily(platform.FAMILY_LINUX) {
		lf, err := p.Runtime.CreateResource("file", "path", "/etc/machine-info")
		if err != nil {
			return "", err
		}
		file := lf.(File)

		exists, err := file.Exists()
		if err != nil {
			return "", err
		}
		// if the file does not exist, the pretty hostname is just empty
		if !exists {
			return "", nil
		}

		err = p.Runtime.WatchAndCompute(file, "content", p, "name")
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
		return hostname.Hostname(p.Runtime.Motor.Transport, pf)
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
		return hostname.Hostname(p.Runtime.Motor.Transport, pf)
	}

	return "", errors.New("your platform is not supported by operating system resource")
}

// returns the OS native machine UUID/GUID
func (s *lumiOs) GetMachineid() (string, error) {
	platform, err := s.Runtime.Motor.Platform()
	if err != nil {
		return "", errors.New("cannot determine platform uuid")
	}

	var uuidProvider platformid.UniquePlatformIDProvider
	for i := range platform.Family {
		if platform.Family[i] == "linux" {
			uuidProvider = &platformid.LinuxIdProvider{Motor: s.Runtime.Motor}
		}
	}

	if uuidProvider == nil && platform.Name == "macos" {
		uuidProvider = &platformid.MacOSIdProvider{Motor: s.Runtime.Motor}
	}

	if uuidProvider == nil {
		return "", errors.New("cannot determine platform uuid for " + platform.Name)
	}

	id, err := uuidProvider.ID()
	if err != nil {
		return "", errors.New("cannot determine platform uuid on known system " + platform.Name)
	}

	return id, nil
}
