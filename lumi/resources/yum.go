package resources

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/lumi/resources/yum"
	"go.mondoo.io/mondoo/stringx"
)

var supportedPlatforms = []string{"amazonlinux"}

func (y *lumiYum) id() (string, error) {
	return "yum", nil
}

func (y *lumiYum) GetRepos() ([]interface{}, error) {
	pf, err := y.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if !pf.IsFamily("redhat") && !stringx.Contains(supportedPlatforms, pf.Name) {
		return nil, errors.New("yum.vars is only supported on redhat-based platforms")
	}

	cmd, err := y.Runtime.Motor.Transport.RunCommand("yum -v repolist all")
	if err != nil {
		return nil, errors.Wrap(err, "could not retrieve yum repo list")
	}

	if cmd.ExitStatus != 0 {
		return nil, errors.New("could not retrieve yum repo list")
	}

	repos, err := yum.ParseRepos(cmd.Stdout)
	if err != nil {
		return nil, err
	}

	lumiRepos := make([]interface{}, len(repos))
	for i, repo := range repos {
		f, err := y.Runtime.CreateResource("file", "path", repo.Filename)
		if err != nil {
			return nil, err
		}

		lumiRepo, err := y.Runtime.CreateResource("yum.repo",
			"id", repo.Id,
			"name", repo.Name,
			"status", repo.Status,
			"baseurl", sliceInterface(repo.Baseurl),
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
		lumiRepos[i] = lumiRepo
	}

	return lumiRepos, nil
}

var rhel67release = regexp.MustCompile(`^[6|7].*$`)

func (y *lumiYum) GetVars() (map[string]interface{}, error) {
	pf, err := y.Runtime.Motor.Platform()
	if err != nil {
		return nil, err
	}

	if !pf.IsFamily("redhat") {
		return nil, errors.New("yum.vars is only supported on redhat-based platforms")
	}

	// use dnf script
	script := yum.Rhel8VarsCommand

	// fallback for older versions like 6 and 7 version to use yum script
	if rhel67release.MatchString(pf.Release) {
		script = yum.Rhel6VarsCommand
	}

	cmd, err := y.Runtime.Motor.Transport.RunCommand(script)
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

func (y *lumiYumRepo) id() (string, error) {
	return y.Id()
}

func (p *lumiYumRepo) init(args *lumi.Args) (*lumi.Args, YumRepo, error) {
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

	obj, err := p.Runtime.CreateResource("yum")
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

func (y *lumiYumRepo) GetEnabled() (bool, error) {
	status, err := y.Status()
	if err != nil {
		return false, err
	}
	return strings.ToLower(status) == "enabled", nil
}
