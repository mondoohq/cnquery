package vagrant

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

var ParseVagrantStatusRegex = regexp.MustCompile(`^(.*?)\s+(not created|running)\s(?:.*)$`)

func ParseVagrantStatus(r io.Reader) (map[string]bool, error) {
	res := make(map[string]bool)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		m := ParseVagrantStatusRegex.FindStringSubmatch(line)
		if len(m) == 3 {
			running := false

			if m[2] == "running" {
				running = true
			}

			res[m[1]] = running
		}

	}
	return res, nil
}

type VagrantVmSSHConfig struct {
	Host     string
	HostName string
	User     string
	Port     int
	// eg /dev/null
	UserKnownHostsFile string
	// enabled StrictHostKeyChecking - "yes" || "no"
	StrictHostKeyChecking string
	// enabled password authentication - "yes" || "no"
	PasswordAuthentication string
	// eg. .vagrant/machines/default/virtualbox/private_key
	IdentityFile string
	//  "yes" || "no"
	IdentitiesOnly string
	LogLevel       string
}

func ParseVagrantSshConfig(r io.Reader) (map[string]*VagrantVmSSHConfig, error) {
	res := make(map[string]*VagrantVmSSHConfig)
	scanner := bufio.NewScanner(r)

	var config *VagrantVmSSHConfig
	for scanner.Scan() {
		line := scanner.Text()
		log.Debug().Msg(line)

		fields := strings.Fields(strings.TrimSpace(line))

		if len(fields) == 2 {
			switch fields[0] {
			case "Host":
				if config != nil {
					res[config.Host] = config
				}
				config = &VagrantVmSSHConfig{}
				config.Host = fields[1]
			case "HostName":
				config.HostName = fields[1]
			case "IdentitiesOnly":
				config.IdentitiesOnly = fields[1]
			case "IdentityFile":
				config.IdentityFile = fields[1]
			case "LogLevel":
				config.LogLevel = fields[1]
			case "PasswordAuthentication":
				config.PasswordAuthentication = fields[1]
			case "Port":
				config.Port, _ = strconv.Atoi(fields[1])
			case "StrictHostKeyChecking":
				config.StrictHostKeyChecking = fields[1]
			case "User":
				config.User = fields[1]
			case "UserKnownHostsFile":
				config.UserKnownHostsFile = fields[1]
			}
		}
	}

	// add the last element
	if config != nil {
		res[config.Host] = config
	}

	return res, nil
}
