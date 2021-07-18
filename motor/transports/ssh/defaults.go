package ssh

import (
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/mitchellh/go-homedir"
	"go.mondoo.io/mondoo/motor/transports"
)

// ApplyDefaults applies all ssh defaults to the transport. It specifically set
// - default port
// - loads ssh keys from known locations
// - configures ssh agent authentication
func ApplyDefaults(cc *transports.TransportConfig, username string, identityFile string, password string) error {
	// set default port for ssh
	ApplyDefaultConfig(cc)

	// handle credentials cases:
	// if identity file is provided but no password -> private key
	// if identity file is provided with password -> encrypted private key
	// if no identity file is provided but a password -> password
	if identityFile != "" {
		credential, err := transports.NewPrivateKeyCredentialFromPath(username, identityFile, password)
		if err != nil {
			return err
		}
		cc.AddCredential(credential)
	} else if password != "" {
		credential := transports.NewPasswordCredential(username, password)
		cc.AddCredential(credential)
	}

	// add default identities
	ApplyDefaultIdentities(cc, username, password)

	return nil
}

// ApplyDefaultConfig set defaults like the ssh port 22 to the transport configuration
// to cover cases where users have not set those values explicitly
func ApplyDefaultConfig(cc *transports.TransportConfig) *transports.TransportConfig {
	p, err := cc.IntPort()
	// use default port if port is 0
	if err == nil && p <= 0 {
		cc.Port = "22"
	}
	return cc
}

// ApplyDefaultIdentities loads user's ssh identifies from ~/.ssh/
func ApplyDefaultIdentities(cc *transports.TransportConfig, username string, password string) *transports.TransportConfig {
	// ssh config overwrite like: IdentityFile ~/.foo/identity is done in ReadSSHConfig()
	// fallback to default paths 	~/.ssh/id_rsa and ~/.ssh/id_dsa if they exist
	home, err := homedir.Dir()
	if err == nil {
		files := []string{
			filepath.Join(home, ".ssh", "id_rsa"),
			filepath.Join(home, ".ssh", "id_dsa"),
			filepath.Join(home, ".ssh", "id_ed25519"),
			// specific handling for google compute engine, see https://cloud.google.com/compute/docs/instances/connecting-to-instance
			filepath.Join(home, ".ssh", "google_compute_engine"),
		}

		// filter keys by existence
		for i := range files {
			f := files[i]
			_, err := os.Stat(f)
			if err == nil {
				log.Debug().Str("key", f).Msg("load ssh identity")
				credential, err := transports.NewPrivateKeyCredentialFromPath(username, f, password)
				if err != nil {
					log.Warn().Err(err).Str("key", f).Msg("could not load ssh identity")
				} else {
					cc.AddCredential(credential)
				}
			}
		}
	}

	// enable ssh-agent authentication as default
	cc.AddCredential(&transports.Credential{Type: transports.CredentialType_ssh_agent, User: username})

	return cc
}
