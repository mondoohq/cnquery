package resources

import (
	"errors"
	"os"

	"go.mondoo.io/mondoo/vadvisor/specs/cvss"

	uuid "github.com/gofrs/uuid/v3"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/packages"
	"go.mondoo.io/mondoo/vadvisor/api"
)

var Scanner *packages.Scanner

// TODO: make this a harmonized approach with mondoo vuln command
var MONDOO_API = "https://api.mondoo.app"

// allow overwrite of the API url by an environment variable
func init() {
	if len(os.Getenv("MONDOO_API")) > 0 {
		MONDOO_API = os.Getenv("MONDOO_API")
	}

	Scanner = &packages.Scanner{
		MondooApiUrl: MONDOO_API,
	}
}

func (c *lumiCvss) id() (string, error) {
	return uuid.Must(uuid.NewV4()).String(), nil
}

func (c *lumiCvss) GetScore() (float64, error) {
	v, _ := c.Vector()
	log.Debug().Str("vector", v).Msg("cvss vector")
	score, err := cvss.New(v)
	if err != nil {
		return 0, err
	}
	return score.Score, nil
}

func (c *lumiCve) id() (string, error) {
	return c.Id()
}

func (c *lumiCve) GetScores() ([]interface{}, error) {
	id, err := c.Id()
	if err != nil {
		return nil, err
	}

	cve, err := Scanner.GetCve(id)
	if err != nil {
		return nil, err
	}

	scores := make([]interface{}, len(cve.Cvss))
	for i := range cve.Cvss {
		entry := cve.Cvss[i]
		args := make(lumi.Args)
		args["vector"] = entry.Vector
		args["source"] = entry.Source

		e, err := newCvss(c.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("cve", cve.Id).Msg("lumi[cve]> could not create cvss resource")
			continue
		}
		scores[i] = e.(Cvss)
	}
	return scores, nil
}

func (c *lumiCve) GetCvss() (interface{}, error) {
	cvsslist, err := c.GetScores()
	if err != nil {
		return nil, err
	}

	return maxCvssScore(cvsslist)
}

func maxCvssScore(cvsslist []interface{}) (Cvss, error) {
	// no entry, no return :-)
	if len(cvsslist) == 0 {
		return nil, nil
	}

	res, ok := cvsslist[0].(Cvss)
	if !ok {
		return nil, errors.New("entry is no cvss")
	}

	// easy, we just have one entry
	if len(cvsslist) == 1 {
		return res, nil
	}

	// fun starts, we need to compare cvss scores now
	max := res
	maxVector, _ := max.Vector()
	maxScore, err := cvss.New(maxVector)
	if err != nil {
		return nil, errors.New("no valid cvss vector")
	}

	for i := 1; i < len(cvsslist); i++ {
		entry, ok := cvsslist[i].(Cvss)
		if !ok {
			return nil, errors.New("entry is no cvss")
		}
		vector, _ := entry.Vector()
		score, err := cvss.New(vector)
		if err != nil {
			return nil, errors.New("no valid cvss vector")
		}

		if maxScore.Compare(score) < 0 {
			max = entry
			maxVector = vector
			maxScore = score
		}
	}

	return max, nil
}

func genLumiCvss(runtime *lumi.Runtime, cvss *api.CVSS) (Cvss, error) {
	if cvss == nil {
		return nil, errors.New("cvss value needs to be set")
	}
	args := make(lumi.Args)
	log.Debug().Str("vector", cvss.Vector).Msg("create cvss")
	args["vector"] = cvss.Vector
	args["source"] = cvss.Source
	e, err := newCvss(runtime, &args)
	if err != nil {
		log.Error().Err(err).Msg("lumi[cvss]> could not generate cvss resource")
		return nil, err
	}
	return e.(Cvss), nil
}

func (c *lumiCve) GetSummary() (string, error) {
	id, err := c.Id()
	if err != nil {
		return "", err
	}

	cve, err := Scanner.GetCve(id)
	if err != nil {
		return "", err
	}

	return cve.Summary, nil
}

func (a *lumiAdvisory) init(args *lumi.Args) (*lumi.Args, error) {
	// nothing to do yet...
	return args, nil
}

func (a *lumiAdvisory) id() (string, error) {
	return a.Id()
}

// TODO: do we need to cache the data for quick access?
func (a *lumiAdvisory) GetName() (string, error) {
	id, err := a.Id()
	if err != nil {
		return "", err
	}

	advisory, err := Scanner.GetAdvisory(id)
	if err != nil {
		return "", err
	}

	return advisory.Title, nil
}

// TODO: do we need to cache the data for quick access?
func (a *lumiAdvisory) GetDescription() (string, error) {
	id, err := a.Id()
	if err != nil {
		return "", err
	}

	advisory, err := Scanner.GetAdvisory(id)
	if err != nil {
		return "", err
	}

	return advisory.Description, nil
}

func (a *lumiAdvisory) GetFixed() ([]interface{}, error) {
	id, err := a.Id()
	if err != nil {
		return nil, err
	}

	advisory, err := Scanner.GetAdvisory(id)
	if err != nil {
		return nil, err
	}

	fixedPkgs := make([]interface{}, len(advisory.Fixed))
	for i := range advisory.Fixed {
		fixed := advisory.Fixed[i]
		args := make(lumi.Args)
		args["name"] = fixed.Name
		args["version"] = fixed.Version
		args["format"] = fixed.Format
		args["arch"] = fixed.Arch

		e, err := newPackage(a.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("package", fixed.Name).Msg("lumi[advisories]> could not create package resource")
			continue
		}
		fixedPkgs[i] = e.(Package)
	}
	return fixedPkgs, nil
}

func (a *lumiAdvisory) GetCvss() (interface{}, error) {
	cvelist, err := a.GetCves()
	if err != nil {
		return nil, err
	}

	allscores := []interface{}{}

	// iterate over all cves and collect cvss
	for i := range cvelist {
		cve, ok := cvelist[i].(Cve)
		if !ok {
			return nil, errors.New("list contains non-valid cve entry")
		}
		cvescores, err := cve.Scores()
		if err != nil {
			return nil, err
		}
		allscores = append(allscores, cvescores...)
	}

	res, err := maxCvssScore(allscores)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, errors.New("could not determine score")
	}

	return res, nil
}

func (a *lumiAdvisory) GetCves() ([]interface{}, error) {

	id, err := a.Id()
	if err != nil {
		return nil, err
	}
	advisory, err := Scanner.GetAdvisory(id)
	if err != nil {
		return nil, err
	}

	// we cannot create the list with len, since we may skip entries
	cveList := []interface{}{}
	log.Debug().Str("id", id).Int("length", len(advisory.Cves)).Msg("found cves")
	for i := range advisory.Cves {
		cve := advisory.Cves[i]
		args := make(lumi.Args)
		args["id"] = cve.Id
		args["summary"] = cve.Summary

		e, err := newCve(a.Runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("cve", cve.Id).Msg("lumi[packages]> could not create package resource")
			continue
		}
		cveList = append(cveList, e.(Cve))
	}

	// TODO: sort list by criticality
	return cveList, nil
}

func (a *lumiAdvisories) init(args *lumi.Args) (*lumi.Args, error) {
	return args, nil
}

func (a *lumiAdvisories) id() (string, error) {
	return "advisories", nil
}

func (a *lumiAdvisories) GetCves() ([]interface{}, error) {
	advisories, err := a.GetList()
	if err != nil {
		return nil, err
	}

	// iterate over each advisory and extract the list of cves
	cveMap := make(map[string]interface{})
	for i := range advisories {
		advisory := advisories[i].(Advisory)
		advisoryCveList, err := advisory.Cves()
		if err != nil {
			log.Error().Err(err).Msg("could not determine cves")
			continue
		}

		if len(advisoryCveList) == 0 {
			continue
		}

		// store each cve into map
		for j := range advisoryCveList {
			cve := advisoryCveList[j].(Cve)
			id, err := cve.Id()
			if err != nil {
				id, _ := advisory.Id()
				log.Error().Err(err).Str("advisory", id).Msg("advisories> could not gather cve data")
				continue
			}

			// check that cve is not nil
			if cve == nil {
				log.Warn().Err(err).Str("advisory", id).Msg("advisories> found empty cve data. Please report a bug.")
				continue
			}
			cveMap[id] = cve
		}
	}

	// convert map to slice
	cveList := make([]interface{}, len(cveMap))
	i := 0
	for key := range cveMap {
		val := cveMap[key]
		cveList[i] = val
		i++
	}

	// TODO: sort slice by cvss vector
	return cveList, nil
}

func (a *lumiAdvisories) GetMaxcvss() (interface{}, error) {
	advisories, err := a.GetList()
	if err != nil {
		return nil, err
	}

	list := []interface{}{}
	for i := range advisories {
		advisory := advisories[i].(Advisory)
		cvss, err := advisory.Cvss()
		if err != nil {
			log.Error().Err(err).Msg("could not determine cves")
			continue
		}
		list = append(list, cvss)
	}

	return maxCvssScore(list)
}

func (a *lumiAdvisories) GetList() ([]interface{}, error) {
	lumiPkgs, err := a.Packages()
	if err != nil {
		return nil, err
	}

	list, err := lumiPkgs.List()
	if err != nil {
		return nil, err
	}

	// iterate interface array and convert that to a package array
	pkgs := make([]Package, len(list))
	for i, pkg := range list {
		pkgs[i] = pkg.(Package)
	}

	// search for advisories
	return findAdvisories(a.Runtime, pkgs)
}

// implement handling for package and packages resource
func (p *lumiPackage) GetAdvisories() ([]interface{}, error) {
	// search for advisories
	return findAdvisories(p.Runtime, []Package{p})
}

func (p *lumiPackages) GetAdvisories() (interface{}, error) {
	args := make(lumi.Args)
	args["packages"] = p
	return newAdvisories(p.Runtime, &args)
}

// searches all advisories for given packages
func findAdvisories(runtime *lumi.Runtime, lumiPackages []Package) ([]interface{}, error) {
	platform, err := runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	pkgs := []*api.Package{}
	for _, d := range lumiPackages {
		name, _ := d.Name()
		version, _ := d.Version()
		format, _ := d.Format()
		arch, _ := d.Arch()

		pkgs = append(pkgs, &api.Package{
			Name:    name,
			Version: version,
			Format:  format,
			Arch:    arch,
		})
	}

	report, err := Scanner.Analyze(&api.ScanJob{
		Platform: &api.Platform{
			Name:    platform.Name,
			Release: platform.Release,
			Arch:    platform.Arch,
		},
		Packages: pkgs,
	})
	if err != nil {
		return nil, err
	}

	// iterate over results and create lumi advisory objects
	lumiAdvisories := make([]interface{}, len(report.Advisories))
	for i := range report.Advisories {
		advisory := report.Advisories[i]
		// set init arguments for the lumi package resource
		args := make(lumi.Args)
		args["id"] = advisory.Id

		// search for the affected packages
		// TODO: should we do that with a dynamic query?
		// TODO: we need to get rid of the epoch
		var lumiAffectedPkgs []interface{}
		for j := range advisory.Affected {
			affected := advisory.Affected[j]

			// list over packages to find the instance
			for _, p := range lumiPackages {
				name, _ := p.Name()
				// we already matched version via the api, therefore we do not need to do this here again
				if name == affected.Name {
					lumiAffectedPkgs = append(lumiAffectedPkgs, p)
					break
				}
			}
		}
		args["affected"] = lumiAffectedPkgs

		e, err := newAdvisory(runtime, &args)
		if err != nil {
			log.Error().Err(err).Str("advisory", advisory.Id).Msg("lumi[advisory]> could not create advisory resource")
			continue
		}
		lumiAdvisories[i] = e.(Advisory)
	}

	return lumiAdvisories, nil
}
