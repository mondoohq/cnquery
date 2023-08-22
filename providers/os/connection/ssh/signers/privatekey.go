// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package signers

import (
	"encoding/pem"
	"errors"
	"strings"

	"golang.org/x/crypto/ssh"
)

func GetSignerFromPrivateKeyWithPassphrase(pemBytes []byte, passphrase []byte) (ssh.Signer, error) {
	// check if the key is encrypted
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("ssh: no key found")
	}

	var signer ssh.Signer
	var err error
	if strings.Contains(block.Headers["Proc-Type"], "ENCRYPTED") {
		// we may want to support to parse password protected encrypted key
		signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, passphrase)
		if err != nil {
			return nil, err
		}
	} else {
		// parse unencrypted key
		signer, err = ssh.ParsePrivateKey(pemBytes)
		if err != nil {
			return nil, err
		}
	}

	return signer, nil
}
