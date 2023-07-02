package core

import (
	"fmt"
	"regexp"

	"errors"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core/packages"
)

var PKG_IDENTIFIER = regexp.MustCompile(`^(.*):\/\/(.*)\/(.*)\/(.*)$`)

func (p *mqlPackage) init(args *resources.Args) (*resources.Args, Package, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["name"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := p.MotorRuntime.CreateResource("packages")
	if err != nil {
		return nil, nil, err
	}
	packages := obj.(Packages)

	_, err = packages.List()
	if err != nil {
		return nil, nil, err
	}

	c, ok := packages.MqlResource().Cache.Load("_map")
	if !ok {
		return nil, nil, errors.New("cannot get map of packages")
	}
	cmap := c.Data.(map[string]Package)

	pkg := cmap[name]
	if pkg != nil {
		return nil, pkg, nil
	}

	// if the package cannot be found, we init it as an empty package
	(*args)["version"] = ""
	(*args)["arch"] = ""
	(*args)["format"] = ""
	(*args)["epoch"] = ""
	(*args)["description"] = ""
	(*args)["available"] = ""
	(*args)["installed"] = false

	return args, nil, nil
}

// A system package cannot be installed twice but there are edge cases:
// - the same package name could be installed for multiple archs
// - linux-kernel package get extra treatment and can co-exist in multiple versions
// We use identifiers similar to grafeas artifact identifier for packages
// - deb://name/version/arch
// - rpm://name/version/arch
func (p *mqlPackage) id() (string, error) {
	name, _ := p.Name()
	version, _ := p.Version()
	arch, _ := p.Arch()
	format, _ := p.Format()
	return format + "://" + name + "/" + version + "/" + arch, nil
}

func (p *mqlPackage) GetStatus() (string, error) {
	return "", nil
}

func (p *mqlPackage) GetOutdated() (bool, error) {
	av, err := p.Available()
	if err == nil && len(av) > 0 {
		return true, nil
	}
	return false, nil
}

func (p *mqlPackage) GetOrigin() (string, error) {
	return "", nil
}

func (p *mqlPackages) id() (string, error) {
	return "packages", nil
}

func (p *mqlPackages) GetList() ([]interface{}, error) {
	// find suitable package manager
	pm, err := packages.ResolveSystemPkgManager(p.MotorRuntime.Motor)
	if pm == nil || err != nil {
		return nil, fmt.Errorf("could not detect suitable package manager for platform")
	}

	// retrieve all system packages
	osPkgs, err := pm.List()
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve package list for platform"))
	}
	log.Debug().Int("packages", len(osPkgs)).Msg("mql[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	osAvailablePkgs, err := pm.Available()
	if err != nil {
		log.Debug().Err(err).Msg("mql[packages]> could not retrieve available updates")
		osAvailablePkgs = map[string]packages.PackageUpdate{}
	}
	log.Debug().Int("updates", len(osAvailablePkgs)).Msg("mql[packages]> available updates")

	// make available updates easily findable
	// we use packagename-arch as identifier
	availableMap := make(map[string]packages.PackageUpdate)
	for _, a := range osAvailablePkgs {
		availableMap[a.Name+"/"+a.Arch] = a
	}

	// create MQL package os for each package
	pkgs := make([]interface{}, len(osPkgs))
	namedMap := map[string]Package{}
	for i, osPkg := range osPkgs {
		// check if we found a newer version
		available := ""
		update, ok := availableMap[osPkg.Name+"/"+osPkg.Arch]
		if ok {
			available = update.Available
			log.Debug().Str("package", osPkg.Name).Str("available", update.Available).Msg("mql[packages]> found newer version")
		}

		pkg, err := p.MotorRuntime.CreateResource("package",
			"name", osPkg.Name,
			"version", osPkg.Version,
			"available", available,
			"epoch", "", // TODO: support Epoch
			"arch", osPkg.Arch,
			"status", osPkg.Status,
			"description", osPkg.Description,
			"format", osPkg.Format,
			"installed", true,
			"origin", osPkg.Origin,
		)
		if err != nil {
			return nil, err
		}

		pkgs[i] = pkg
		namedMap[osPkg.Name] = pkg.(Package)
	}

	p.Cache.Store("_map", &resources.CacheEntry{Data: namedMap})

	// return the packages as new entries
	return pkgs, nil
}
