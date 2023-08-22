// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package signers

import (
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func GetSignersFromSSHAgent() []ssh.Signer {
	signers := []ssh.Signer{}

	if sshAgentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		log.Debug().Str("socket", os.Getenv("SSH_AUTH_SOCK")).Msg("ssh agent socket found")
		sshAgentClient := agent.NewClient(sshAgentConn)
		sshAgentSigners, err := sshAgentClient.Signers()
		if err == nil && len(sshAgentSigners) == 0 {
			log.Warn().Msg("could not find keys in ssh agent")
		} else if err == nil {
			signers = append(signers, sshAgentSigners...)
		} else {
			log.Error().Err(err).Msg("could not get public keys from ssh agent")
		}
	} else {
		log.Debug().Msg("could not find valid ssh agent authentication")
	}
	return signers
}
