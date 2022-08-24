package ssh

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/vault"
)

func ReadSSHConfig(cc *providers.Config) *providers.Config {
	host := cc.Host

	home, err := homedir.Dir()
	if err != nil {
		log.Debug().Err(err).Msg("ssh> failed to determine user home directory")
		return cc
	}

	sshUserConfigPath := filepath.Join(home, ".ssh", "config")
	f, err := os.Open(sshUserConfigPath)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("ssh> could not read ssh config")
		return cc
	}

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("could not parse ssh config")
		return cc
	}

	// optional step, tries to parse the ssh config to see if additional information
	// is already available
	hostname, err := cfg.Get(host, "HostName")
	if err == nil && len(hostname) > 0 {
		cc.Host = hostname
	}

	if len(cc.Credentials) == 0 {

		user, _ := cfg.Get(host, "User")
		port, err := cfg.Get(host, "Port")
		if err == nil {
			portNum, err := strconv.Atoi(port)
			if err != nil {
				log.Debug().Err(err).Str("file", sshUserConfigPath).Str("port", port).Msg("could not parse ssh port")
			} else {
				cc.Port = int32(portNum)
			}
		}

		entry, err := cfg.Get(host, "IdentityFile")

		// TODO: the ssh_config uses os/home but instead should be use go-homedir, could become a compile issue
		// TODO: the problem is that the lib returns defaults and we cannot properly distingush
		if err == nil && ssh_config.Default("IdentityFile") != entry {
			// commonly ssh config included paths like ~
			expandedPath, err := homedir.Expand(entry)
			if err == nil {
				log.Debug().Str("key", expandedPath).Str("host", host).Msg("ssh> read ssh identity key from ssh config")
				// NOTE: we ignore the error here for now but this should probably been catched earlier anyway
				credential, _ := vault.NewPrivateKeyCredentialFromPath(user, expandedPath, "")
				// apply the option manually
				if credential != nil {
					cc.AddCredential(credential)
				}
			}
		}
	}

	// handle disable of strict hostkey checking:
	// Host *
	// StrictHostKeyChecking no
	entry, err := cfg.Get(host, "StrictHostKeyChecking")
	if err == nil && strings.ToLower(entry) == "no" {
		cc.Insecure = true
	}
	return cc
}

func VerifyConfig(pCfg *providers.Config) error {
	if pCfg.Backend != providers.ProviderType_SSH {
		return providers.ErrProviderTypeDoesNotMatch
	}

	return nil
}
