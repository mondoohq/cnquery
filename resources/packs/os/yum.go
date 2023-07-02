package os

import (
	"fmt"
	"regexp"
	"strings"

	"errors"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/core"
	"go.mondoo.com/cnquery/resources/packs/os/yum"
	"go.mondoo.com/cnquery/stringx"
)

var supportedPlatforms = []string{"amazonlinux"}

func (y *mqlYum) id() (string, error) {
	return "yum", nil
}

func (y *mqlYum) GetRepos() ([]interface{}, error) {
	pf, err := y.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if !pf.IsFamily("redhat") && !stringx.Contains(supportedPlatforms, pf.Name) {
		return nil, errors.New("yum.repos is only supported on redhat-based platforms")
	}

	osProvider, err := osProvider(y.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	cmd, err := osProvider.RunCommand("yum -v repolist all")
	if err != nil {
		return nil, errors.Join(err, errors.New("could not retrieve yum repo list"))
	}

	if cmd.ExitStatus != 0 {
		return nil, errors.New("could not retrieve yum repo list")
	}

	repos, err := yum.ParseRepos(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	mqlRepos := make([]interface{}, len(repos))
	for i, repo := range repos {
		f, err := y.MotorRuntime.CreateResource("file", "path", repo.Filename)
		if err != nil {
			return nil, err
		}

		mqlRepo, err := y.MotorRuntime.CreateResource("yum.repo",
			"id", repo.Id,
			"name", repo.Name,
			"status", repo.Status,
			"baseurl", core.StrSliceToInterface(repo.Baseurl),
			"expire", repo.Expire,
			"filename", repo.Filename,
			"file", f,
			"revision", repo.Revision,
			"pkgs", repo.Pkgs,
			"size", repo.Size,
			"mirrors", repo.Mirrors,
		)
		if err != nil {
			return nil, err
		}
		mqlRepos[i] = mqlRepo
	}

	return mqlRepos, nil
}

var rhel67release = regexp.MustCompile(`^[6|7].*$`)

func (y *mqlYum) GetVars() (map[string]interface{}, error) {
	pf, err := y.MotorRuntime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if !pf.IsFamily("redhat") && !stringx.Contains(supportedPlatforms, pf.Name) {
		return nil, errors.New("yum.vars is only supported on redhat-based platforms")
	}

	osProvider, err := osProvider(y.MotorRuntime.Motor)
	if err != nil {
		return nil, err
	}

	// use dnf script as default
	script := fmt.Sprintf(yum.DnfVarsCommand, yum.PythonRhel)
	if !pf.IsFamily("redhat") {
		// eg. amazon linux does not ship with /usr/libexec/platform-python
		script = fmt.Sprintf(yum.DnfVarsCommand, yum.Python3)
	}

	// fallback for older versions like 6 and 7 version to use yum script
	if rhel67release.MatchString(pf.Version) {
		script = yum.Rhel6VarsCommand
	}

	cmd, err := osProvider.RunCommand(script)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve yum variables")
	}

	vars, err := yum.ParseVariables(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	res := map[string]interface{}{}
	for k := range vars {
		res[k] = vars[k]
	}

	return res, nil
}

func (y *mqlYumRepo) id() (string, error) {
	return y.Id()
}

func (p *mqlYumRepo) init(args *resources.Args) (*resources.Args, YumRepo, error) {
	if len(*args) > 2 {
		return args, nil, nil
	}

	nameRaw := (*args)["id"]
	if nameRaw == nil {
		return args, nil, nil
	}

	name, ok := nameRaw.(string)
	if !ok {
		return args, nil, nil
	}

	obj, err := p.MotorRuntime.CreateResource("yum")
	if err != nil {
		return nil, nil, err
	}
	yumResource := obj.(Yum)

	repos, err := yumResource.Repos()
	if err != nil {
		return nil, nil, err
	}

	for i := range repos {
		selected := repos[i].(YumRepo)
		id, err := selected.Id()
		if err == nil && id == name {
			return nil, selected, nil
		}
	}

	// if the repo cannot be found we return an error
	return nil, nil, errors.New("could not find yum repo " + name)
}

func (y *mqlYumRepo) GetEnabled() (bool, error) {
	status, err := y.Status()
	if err != nil {
		return false, err
	}
	return strings.ToLower(status) == "enabled", nil
}
