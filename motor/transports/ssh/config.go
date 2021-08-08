package ssh

import (
	"errors"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/kevinburke/ssh_config"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/transports"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func ReadSSHConfig(endpoint *transports.TransportConfig) *transports.TransportConfig {
	host := endpoint.Host

	home, err := homedir.Dir()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to determine user home directory")
		return endpoint
	}

	sshUserConfigPath := filepath.Join(home, ".ssh", "config")
	f, err := os.Open(sshUserConfigPath)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("Could not read ssh config")
		return endpoint
	}

	cfg, err := ssh_config.Decode(f)
	if err != nil {
		log.Debug().Err(err).Str("file", sshUserConfigPath).Msg("Could not parse ssh config")
		return endpoint
	}

	// optional step, tries to parse the ssh config to see if additional information
	// is already available
	hostname, err := cfg.Get(host, "HostName")
	if err == nil && len(hostname) > 0 {
		endpoint.Host = hostname
	}

	if len(endpoint.User) == 0 {
		user, err := cfg.Get(host, "User")
		if err == nil {
			endpoint.User = user
		}
	}

	if len(endpoint.Port) == 0 {
		port, err := cfg.Get(host, "Port")
		if err == nil {
			endpoint.Port = port
		}
	}

	if len(endpoint.IdentityFiles) == 0 {
		entry, err := cfg.Get(host, "IdentityFile")
		// TODO: the ssh_config uses os/home but instead should be use go-homedir, could become a compile issue
		// TODO: the problem is that the lib returns defaults and we cannot properly distingush
		if err == nil && ssh_config.Default("IdentityFile") != entry {
			// commonly ssh config included paths like ~
			expanded, err := homedir.Expand(entry)
			if err == nil {
				log.Debug().Str("key", expanded).Str("host", host).Msg("read ssh identity key from ssh config")
				endpoint.IdentityFiles = append(endpoint.IdentityFiles, expanded)
			}
		}
	}

	// handle disable of strict hostkey checking:
	// Host *
	// StrictHostKeyChecking no
	entry, err := cfg.Get(host, "StrictHostKeyChecking")
	if err == nil && strings.ToLower(entry) == "no" {
		endpoint.Insecure = true
	}
	return endpoint
}

func VerifyConfig(endpoint *transports.TransportConfig) error {
	if endpoint.Backend != transports.TransportBackend_CONNECTION_SSH {
		return errors.New("only ssh backend for ssh transport supported")
	}

	_, err := endpoint.IntPort()
	if err != nil {
		return errors.New("port is not a valid number " + endpoint.Port)
	}

	return nil
}

func DefaultConfig(endpoint *transports.TransportConfig) *transports.TransportConfig {
	p, err := endpoint.IntPort()
	// use default port if port is 0
	if err == nil && p <= 0 {
		endpoint.Port = "22"
	}

	if endpoint.User == "" {
		usr, err := user.Current()
		if err != nil {
			log.Warn().Err(err).Msg("could not fallback do current user")
		}
		endpoint.User = usr.Username
	}

	// ssh config overwrite like: IdentityFile ~/.foo/identity is done in ReadSSHConfig()
	// fallback to default paths 	~/.ssh/id_rsa and ~/.ssh/id_dsa if they exist
	home, err := homedir.Dir()
	if err == nil {
		files := []string{
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_dsa"),
			// specific handling for google compute engine, see https://cloud.google.com/compute/docs/instances/connecting-to-instance
			// filepath.Join(home, ".ssh", "google_compute_engine"),
		}

		// filter keys by existence
		for i := range files {
			_, err := os.Stat(files[i])
			if err == nil {
				endpoint.IdentityFiles = append(endpoint.IdentityFiles, files[i])
			}
		}
	}

	return endpoint
}

func KnownHostsCallback() (ssh.HostKeyCallback, error) {
	home, err := homedir.Dir()
	if err != nil {
		log.Debug().Err(err).Msg("Failed to determine user home directory")
		return nil, err
	}

	// load default host keys
	files := []string{
		filepath.Join(home, ".ssh", "known_hosts"),
		// see https://cloud.google.com/compute/docs/instances/connecting-to-instance
		// NOTE: content in that file is structured by compute.instanceid key
		// TODO: we need to keep the instance information during the resolve step
		filepath.Join(home, ".ssh", "google_compute_known_hosts"),
	}

	// filter all files that do not exits
	existentKnownHosts := []string{}
	for i := range files {
		_, err := os.Stat(files[i])
		if err == nil {
			existentKnownHosts = append(existentKnownHosts, files[i])
		}
	}

	return knownhosts.New(existentKnownHosts...)
}
