package yum

// To support static analysis, we need to extend the current implementation:
//
// - read repo info from file system as is
// - read variables from file system as is

// https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/6/html/deployment_guide/sec-using_yum_variables
// /etc/yum.conf
// /etc/yum.repos.d/*.repo

// References:
// - https://unix.stackexchange.com/questions/19701/yum-how-can-i-view-variables-like-releasever-basearch-yum0
// - https://docs.centos.org/en-US/8-docs/managing-userspace-components/assembly_using-appstream/

import (
	"bufio"
	"encoding/json"
	"io"
	"io/ioutil"
	"regexp"
	"strings"
)

const (
	RhelYumRepoListCommand = "yum -v repolist all"
	Rhel8VarsCommand       = "/usr/libexec/platform-python -c 'import dnf, json; db = dnf.dnf.Base(); print(json.dumps(db.conf.substitutions))'"
	Rhel6VarsCommand       = "python -c 'import yum, json; yb = yum.YumBase(); print json.dumps(yb.conf.yumvar)'"
)

type YumRepo struct {
	Id       string
	Name     string
	Status   string
	Revision string
	Updated  string
	Pkgs     string
	Size     string
	Mirrors  string
	Expire   string
	Filename string
	Baseurl  []string
	Filter   string
}

var yumrepoline = regexp.MustCompile(`^\s*([^:\s]*)(?:\s)*:\s(.*)$`)
var yumbaseurl = regexp.MustCompile(`^(.*?)(?:\(.*\))*$`)

const (
	Id       = "Repo-id"
	Name     = "Repo-name"
	Status   = "Repo-status"
	Revision = "Repo-revision"
	Updated  = "Repo-updated"
	Pkgs     = "Repo-pkgs"
	Size     = "Repo-size"
	Mirrors  = "Repo-mirrors"
	Baseurl  = "Repo-baseurl"
	Expire   = "Repo-expire"
	Filter   = "Filter"
	Filename = "Repo-filename"
)

func ParseVariables(r io.Reader) (map[string]string, error) {
	content, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	data := map[string]string{}
	err = json.Unmarshal(content, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Parses the output of yum -v repolist all
// It requires yum to be installed
func ParseRepos(r io.Reader) ([]*YumRepo, error) {
	res := []*YumRepo{}

	var entry *YumRepo
	add := func(new *YumRepo) {
		if entry == nil {
			return
		}
		res = append(res, new)
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		m := yumrepoline.FindStringSubmatch(line)
		if len(m) == 3 {
			key := strings.TrimSpace(m[1])
			value := strings.TrimSpace(m[2])

			switch key {
			case Id:
				add(entry)
				entry = &YumRepo{Id: value}
			case Name:
				entry.Name = value
			case Status:
				entry.Status = value
			case Revision:
				entry.Revision = value
			case Updated:
				entry.Updated = value
			case Pkgs:
				entry.Pkgs = value
			case Size:
				entry.Size = value
			case Mirrors:
				entry.Mirrors = value
			case Baseurl:
				// remove (0 more)
				// split by ,
				m := yumbaseurl.FindStringSubmatch(value)
				if m != nil && len(m) >= 2 {
					entries := strings.Split(m[1], ",")
					entry.Baseurl = []string{}
					for i := range entries {
						entry.Baseurl = append(entry.Baseurl, strings.TrimSpace(entries[i]))
					}
				}
			case Expire:
				entry.Expire = value
			case Filter:
				entry.Filter = value
			case Filename:
				entry.Filename = value
			}
		}
	}

	// add last entry
	add(entry)

	return res, nil
}
