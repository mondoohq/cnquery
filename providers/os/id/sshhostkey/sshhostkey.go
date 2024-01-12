// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package sshhostkey

import (
	"os"

	"github.com/cockroachdb/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers/os/connection"
	"go.mondoo.com/cnquery/v10/providers/os/connection/shared"
	"golang.org/x/crypto/ssh"
)

func Detect(t shared.Connection, p *inventory.Platform) ([]string, error) {
	// if we are using an ssh connection we can read the hostkey from the connection
	if sshTransport, ok := t.(*connection.SshConnection); ok {
		identifier, err := sshTransport.PlatformID()
		if err != nil {
			return nil, err
		}
		return []string{identifier}, nil
	}

	// if we are not at the remote system, we try to load the ssh host key from local system
	identifiers := []string{}

	paths := []string{"/etc/ssh/ssh_host_ecdsa_key.pub", "/etc/ssh/ssh_host_ed25519_key.pub", "/etc/ssh/ssh_host_rsa_key.pub"}
	// iterate over paths and read identifier
	for i := range paths {
		hostKeyFilePath := paths[i]
		data, err := os.ReadFile(hostKeyFilePath)
		if os.IsPermission(err) {
			log.Warn().Err(err).Str("hostkey", hostKeyFilePath).Msg("no permission to access ssh hostkey")
			continue
		} else if os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, errors.Wrap(err, "could not read file:"+hostKeyFilePath)
		}
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return nil, errors.Wrap(err, "could not parse public key file:"+hostKeyFilePath)
		}

		identifiers = append(identifiers, connection.PlatformIdentifier(publicKey))
	}

	return identifiers, nil
}
