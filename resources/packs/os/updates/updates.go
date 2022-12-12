package updates

import (
	"errors"

	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/providers/os"
)

type OperatingSystemUpdate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Category    string `json:"category"`
	Restart     bool   `json:"restart"`
	Format      string `json:"format"`
}

type OperatingSystemUpdateManager interface {
	Name() string
	List() ([]OperatingSystemUpdate, error)
}

// ResolveSystemUpdateManager uses the local system updated to ask for updates
func ResolveSystemUpdateManager(motor *motor.Motor) (OperatingSystemUpdateManager, error) {
	var um OperatingSystemUpdateManager

	pf, err := motor.Platform()
	if err != nil {
		return nil, err
	}

	osProvider, isOSProvider := motor.Provider.(os.OperatingSystemProvider)
	if !isOSProvider {
		return nil, errors.New("package manager is not supported for platform: " + pf.Name)
	}

	// TODO: use OS family and select package manager
	switch pf.Name {
	case "opensuse", "sles", "opensuse-leap", "opensuse-tumbleweed": // suse family
		um = &SuseUpdateManager{provider: osProvider}
	case "windows":
		um = &WindowsUpdateManager{provider: osProvider}
	default:
		return nil, errors.New("your platform is not supported by os updates resource")
	}
	return um, nil
}
