package ssh

import (
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

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
