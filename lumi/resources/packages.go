package resources

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/packages"
)

var (
	PKG_IDENTIFIER = regexp.MustCompile(`^(.*):\/\/(.*)\/(.*)\/(.*)$`)
)

func (p *lumiPackage) init(args *lumi.Args) (*lumi.Args, error) {
	if len(*args) > 2 {
		return args, nil
	}

	name := (*args)["name"]
	if name == nil {
		return args, nil
	}

	nameS, ok := name.(string)
	if !ok {
		return args, nil
	}

	obj, err := p.Runtime.CreateResource("packages")
	if err != nil {
		return nil, err
	}
	packages := obj.(Packages)

	_, err = packages.List()
	if err != nil {
		return nil, err
	}

	c, ok := packages.LumiResource().Cache.Load("_map")
	if !ok {
		return nil, errors.New("Cannot get map of packages")
	}
	cmap := c.Data.(map[string]Package)

	pkg := cmap[nameS]
	if pkg == nil {
		(*args)["version"] = ""
		(*args)["arch"] = ""
		(*args)["format"] = ""
		(*args)["epoch"] = ""
		(*args)["description"] = ""
		(*args)["available"] = ""
		(*args)["installed"] = false
	} else {
		// TODO: do this instead of duplicating it!
		// (*args)["id"] = pkg.LumiResource().Id

		(*args)["version"], _ = pkg.Version()
		(*args)["arch"], _ = pkg.Arch()
		(*args)["format"], _ = pkg.Format()
		(*args)["epoch"], _ = pkg.Epoch()
		(*args)["description"], _ = pkg.Description()
		(*args)["available"], _ = pkg.Available()
		(*args)["installed"], _ = pkg.Installed()
	}

	// fmt.Println(logger.PrettyJSON(arr))

	// (*args)["name"] = m[2]
	// (*args)["version"] = m[3]
	// (*args)["arch"] = m[4]
	// (*args)["format"] = m[1]

	// // set values to pass resource creation step
	// (*args)["epoch"] = ""
	// (*args)["description"] = ""
	// (*args)["available"] = ""

	// delete(*args, "id")

	return args, nil
}

// A system package cannot be installed twice but there are edge cases:
// - the same package name could be installed for multiple archs
// - linux-kernel package get extra treatment and can co-exist in multiple versions
// We use identifiers similar to grafeas artifact identifier for packages
// - deb://name/version/arch
// - rpm://name/version/arch
func (p *lumiPackage) id() (string, error) {
	name, _ := p.Name()
	version, _ := p.Version()
	arch, _ := p.Arch()
	format, _ := p.Format()
	return format + "://" + name + "/" + version + "/" + arch, nil
}

func (p *lumiPackage) GetStatus() (string, error) {
	return "nil", nil
}

func (p *lumiPackage) GetOutdated() (bool, error) {
	av, err := p.Available()
	if err == nil && len(av) > 0 {
		return true, nil
	}
	return false, nil
}

func (p *lumiPackages) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (p *lumiPackages) id() (string, error) {
	return "packages", nil
}

func (p *lumiPackages) GetList() ([]interface{}, error) {

	// find suitable package manager
	pm, err := packages.ResolveSystemPkgManager(p.Runtime.Motor)
	if pm == nil || err != nil {
		return nil, fmt.Errorf("Could not detect suiteable package manager for platform")
	}

	// retrieve all system packages
	osPkgs, err := pm.List()
	if err != nil {
		return nil, fmt.Errorf("Could not retrieve package list for platform")
	}
	log.Debug().Int("packages", len(osPkgs)).Msg("lumi[packages]> installed packages")

	// TODO: do we really need to make this a blocking call, we could update available updates async
	// we try to retrieve the available updates
	osAvailablePkgs, err := pm.Available()
	if err != nil {
		log.Warn().Err(err).Msg("lumi[packages]> could not retrieve available updates")
		osAvailablePkgs = map[string]packages.PackageUpdate{}
	}
	log.Debug().Int("updates", len(osAvailablePkgs)).Msg("lumi[packages]> available updates")

	// make available updates easily findable
	// we use packagename-arch as identifier
	availableMap := make(map[string]packages.PackageUpdate)
	for _, a := range osAvailablePkgs {
		availableMap[a.Name+"/"+a.Arch] = a
	}

	// create lumi package resources for each package
	pkgs := make([]interface{}, len(osPkgs))
	namedMap := map[string]Package{}
	for i, osPkg := range osPkgs {

		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["name"] = osPkg.Name
		args["version"] = osPkg.Version
		args["arch"] = osPkg.Arch
		args["status"] = osPkg.Status
		args["description"] = osPkg.Description
		args["format"] = pm.Format()
		args["installed"] = true

		// check if we found a newer version
		args["available"] = ""
		update, ok := availableMap[osPkg.Name+"/"+osPkg.Arch]
		if ok {
			args["available"] = update.Available
			log.Debug().Str("package", osPkg.Name).Str("available", update.Available).Msg("lumi[packages]> found newer version")
		}

		e, err := newPackage(p.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("package", osPkg.Name).Msg("lumi[packages]> could not create package resource")
			continue
		}

		pkgs[i] = e.(Package)
		namedMap[osPkg.Name] = e.(Package)
	}

	p.Cache.Store("_map", &lumi.CacheEntry{Data: namedMap})

	// return the packages as new entries
	return pkgs, nil
}
