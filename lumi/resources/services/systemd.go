package services

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/coreos/go-systemd/unit"
	"github.com/spf13/afero"
	"go.mondoo.io/mondoo/motor"
	"go.mondoo.io/mondoo/motor/transports"
)

var (
	SYSTEMD_LIST_UNITS_REGEX = regexp.MustCompile(`(?m)^(?:[^\S\n]{2}|●[^\S\n]|)(\S+)(?:[^\S\n])+(loaded|not-found|masked)(?:[^\S\n])+(\S+)(?:[^\S\n])+(\S+)(?:[^\S\n])+(.+)$`)
	serviceNameRegex         = regexp.MustCompile(`(.*)\.(service|target|socket)$`)
	errIgnored               = errors.New("ignored")
)

func ResolveSystemdServiceManager(m *motor.Motor) OSServiceManager {
	if !m.Transport.Capabilities().HasCapability(transports.Capability_RunCommand) {
		return &SystemdFSServiceManager{Fs: m.Transport.FS()}
	}
	return &SystemDServiceManager{motor: m}
}

// a line may be prefixed with nothing, whitespace or a dot
func ParseServiceSystemDUnitFiles(input io.Reader) ([]*Service, error) {
	var services []*Service
	content, err := ioutil.ReadAll(input)
	if err != nil {
		return nil, err
	}

	m := SYSTEMD_LIST_UNITS_REGEX.FindAllStringSubmatch(string(content), -1)
	for i := range m {
		name := m[i][1]
		name = strings.Replace(name, ".service", "", 1)

		s := &Service{
			Name:      name,
			Installed: m[i][2] == "loaded",
			Running:   m[i][3] == "active",
			// TODO: we may need to revist the enabled state
			Enabled:     m[i][2] == "loaded",
			Masked:      m[i][2] == "masked",
			Description: m[i][5],
			Type:        "systemd",
		}
		services = append(services, s)
	}
	return services, nil
}

// Newer linux systems use systemd as service manager
type SystemDServiceManager struct {
	motor *motor.Motor
}

func (s *SystemDServiceManager) Name() string {
	return "systemd Service Manager"
}

func (s *SystemDServiceManager) List() ([]*Service, error) {
	c, err := s.motor.Transport.RunCommand("systemctl --all list-units --type service")
	if err != nil {
		return nil, err
	}
	return ParseServiceSystemDUnitFiles(c.Stdout)
}

type SystemdFSServiceManager struct {
	Fs afero.Fs
}

// systemdUnitSearchPath is the order in which systemd looks up unit files
// We ignore anything in /run as fs scans should not represent a running system
// https://www.freedesktop.org/software/systemd/man/systemd.unit.html#Unit%20File%20Load%20Path
var systemdUnitSearchPath = []string{
	"/etc/systemd/system.control",
	"/etc/systemd/system",
	"/usr/local/lib/systemd/system",
	"/usr/lib/systemd/system",
}

type unitInfo struct {
	// name is the name of the unit without the type extension
	name string
	// uType is the type extension, for example service, target, etc
	uType string
	// description is the description that is provided in the unit section
	description string
	// deps is a list of all name.type values found in the Wants and Requires
	// fields of the Unit section
	deps []string
	// Orderings is a list of all name.type values found in the Before and
	// After fields of the Unit section
	orderings []string
	// masked is set to true of a unit is symlinked to /dev/null
	masked bool
	// missing is set to true if we have a dependency on a unit, but that
	// unit was not found in the search path
	missing bool
	// isDep is true of this unit is found in the dependency tree starting
	// from the default.target
	isDep bool
	// service is only set for socket units. It contains an optional name.target.
	// If not provided, socketname.service is activated for the socket
	service string
}

type stackEntry struct {
	unit     string
	critical bool
}
type stack []stackEntry

func (s *stack) push(v stackEntry) {
	*s = append(*s, v)
}

func (s *stack) pop() stackEntry {
	n := len(*s) - 1
	v := (*s)[n]
	*s = (*s)[:n]
	return v
}

func (s *stack) len() int {
	return len(*s)
}

func (s *SystemdFSServiceManager) Name() string {
	return "systemd FS Service Manager"
}

func (s *SystemdFSServiceManager) List() ([]*Service, error) {
	enabledUnits, err := s.traverse()
	if err != nil {
		return nil, err
	}
	services := make([]*Service, 0, len(enabledUnits))
	for _, v := range enabledUnits {
		if v.uType != "service" {
			continue
		}
		services = append(services, &Service{
			Name:        v.name,
			Type:        v.uType,
			Description: v.description,
			State:       ServiceUnknown,
			Installed:   !v.missing,
			Enabled:     !v.missing && v.isDep,
			Masked:      v.masked,
		})
	}
	return services, nil
}

// traverse traverses the root target and finds units. This implementation is
// incomplete. It only looks at targets, services, and sockets, so at least
// mounts and timers are missing. Also, handling of templates is probably not
// fully correct. The implicit and default dependencies for types are also
// not accounted for
func (s *SystemdFSServiceManager) traverse() (map[string]*unitInfo, error) {
	loadedUnits := map[string]*unitInfo{}
	stack := new(stack)
	stack.push(stackEntry{
		critical: true,
		unit:     "default.target",
	})
	for stack.len() > 0 {
		u := stack.pop()
		if l, ok := loadedUnits[u.unit]; ok {
			if !l.isDep && u.critical {
				// We need to revisit all the already loaded units
				// and mark them as a dependency
				l.isDep = true
				for _, v := range l.deps {
					stack.push(stackEntry{
						unit:     v,
						critical: true,
					})
				}
			}
			continue
		}
		uInfo, err := s.findUnit(u.unit)
		if err != nil {
			if errors.Is(err, errIgnored) {
				continue
			}
			return nil, err
		}
		for _, d := range uInfo.deps {
			stack.push(stackEntry{
				unit:     d,
				critical: u.critical,
			})
		}
		for _, d := range uInfo.orderings {
			stack.push(stackEntry{
				unit:     d,
				critical: false,
			})
		}
		if uInfo.uType == "socket" {
			d := uInfo.service
			if d == "" {
				d = fmt.Sprintf("%s.service", uInfo.name)
			}
			stack.push(stackEntry{
				unit:     d,
				critical: u.critical,
			})
		}
		uInfo.isDep = uInfo.isDep || u.critical
		loadedUnits[u.unit] = uInfo
	}
	return loadedUnits, nil
}

func (s *SystemdFSServiceManager) findUnit(unitName string) (*unitInfo, error) {
	name, uType, ok := unitNameAndType(unitName)
	if !ok {
		return nil, errIgnored
	}
	uInfo := unitInfo{
		name:  name,
		uType: uType,
	}

	for _, p := range systemdUnitSearchPath {
		var err error
		var fsInfo fs.FileInfo

		uName := unitName
		unitPath := path.Join(p, unitName)

		// We try to lstat if we can. We want to know if the file is
		// a symlink because symlinks are aliases
		if lstater, ok := s.Fs.(afero.Lstater); ok {
			fsInfo, _, err = lstater.LstatIfPossible(unitPath)
		} else {
			fsInfo, err = s.Fs.Stat(unitPath)
		}
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		findNames := []string{uName}
		if fsInfo.Mode()&fs.ModeSymlink != 0 {
			// If its a symlink, we need to get the real name
			// TODO: check if this needs to be done recursively
			if lr, ok := s.Fs.(afero.LinkReader); ok {
				linkPath, err := lr.ReadlinkIfPossible(unitPath)
				if err != nil {
					return nil, err
				}
				if linkPath == "/dev/null" {
					uInfo.masked = true
					return &uInfo, nil
				} else {
					linkedName := path.Base(linkPath)
					name, uType, ok := unitNameAndType(linkedName)
					// The rules for aliasing only allow same type to same
					// type. So foo.service -> bar.service, but not
					// foo.service -> bar.socket
					if !ok || (uInfo.uType != uType) {
						return nil, fmt.Errorf("invalid unit %s", linkedName)
					}
					uInfo.name = name
					findNames = append(findNames, linkedName)
				}
			}
		}

		if err := s.readUnit(unitPath, &uInfo); err != nil {
			return nil, err
		}

		// We need to search for deps from directories based on both the
		// real name and aliased name
		for _, n := range findNames {
			dirDeps, err := s.findDeps(n)
			if err != nil {
				return nil, err
			}
			uInfo.deps = append(uInfo.deps, dirDeps...)
		}

		return &uInfo, nil
	}

	uInfo.missing = true
	return &uInfo, nil
}

// readUnit reads the unit file:
// deps are pulled from the Wants and Requires keys of the Unit section
// description is pulled from the Description key of the Unit section
// orderings are pulled from the Before and After keys of the Unit section
func (s *SystemdFSServiceManager) readUnit(unitPath string, uInfo *unitInfo) error {
	f, err := s.Fs.Open(unitPath)
	if err != nil {
		return err
	}
	opts, err := unit.Deserialize(f)
	if err != nil {
		return err
	}

	for _, o := range opts {
		if o.Section == "Unit" && (o.Name == "Wants" || o.Name == "Requires") {
			deps := strings.Fields(o.Value)
			for _, d := range deps {
				if serviceNameRegex.MatchString(d) {
					uInfo.deps = append(uInfo.deps, d)
				}
			}
		} else if o.Section == "Unit" && o.Name == "Description" {
			uInfo.description = o.Value
		} else if o.Section == "Unit" && (o.Name == "Before" || o.Name == "After") {
			orderings := strings.Fields(o.Value)
			for _, d := range orderings {
				if serviceNameRegex.MatchString(d) {
					uInfo.orderings = append(uInfo.orderings, d)
				}
			}
		} else if o.Section == "Socket" && o.Name == "Service" {
			uInfo.service = o.Value
		}
	}

	return nil
}

// findDeps looks up the dependencies for the given unit name (foo.service)
// by looking up all the links in the foo.service.wants and foo.service.requires
// directories
func (s *SystemdFSServiceManager) findDeps(unitName string) ([]string, error) {
	deps := []string{}
	for _, searchPath := range systemdUnitSearchPath {
		paths := []string{
			path.Join(searchPath, fmt.Sprintf("%s.wants", unitName)),
			path.Join(searchPath, fmt.Sprintf("%s.requires", unitName)),
		}

		for _, p := range paths {
			_, err := s.Fs.Stat(p)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			unitLinks, err := afero.ReadDir(s.Fs, p)
			if err != nil {
				return nil, err
			}
			for _, unitLink := range unitLinks {
				if serviceNameRegex.MatchString(unitLink.Name()) {
					deps = append(deps, unitLink.Name())
				}
			}
		}
	}

	return deps, nil
}

func unitNameAndType(n string) (name string, uType string, ok bool) {
	matches := serviceNameRegex.FindStringSubmatch(n)
	if len(matches) > 1 {
		name = matches[1]
	}
	if len(matches) > 2 {
		uType = matches[2]
	}
	if len(matches) == 3 {
		ok = true
	}
	return
}
