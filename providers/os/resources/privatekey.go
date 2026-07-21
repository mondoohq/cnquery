// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"crypto/dsa" //nolint:staticcheck // DSA is deprecated, but we still detect legacy DSA keys for crypto-posture auditing
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"errors"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"golang.org/x/crypto/ssh"
)

// mqlPrivatekeyInternal caches the parsed PEM key so the independently
// lazy-loaded publicKeyAlgorithm and publicKeyBits accessors don't each
// re-parse the payload. The result is computed inside parseOnce, whose
// happens-before guarantee makes the captured key/err safe to read afterward.
type mqlPrivatekeyInternal struct {
	parseOnce sync.Once
	parsedKey any
	parseErr  error
}

func initPrivatekey(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if x, ok := args["path"]; ok {
		path, ok := x.Value.(string)
		if !ok {
			return nil, nil, errors.New("wrong type for 'path' in privatekey initialization, it must be a string")
		}
		f, err := CreateResource(runtime, "file", map[string]*llx.RawData{
			"path": llx.StringData(path),
		})
		if err != nil {
			return nil, nil, err
		}
		args["file"] = llx.ResourceData(f, "file")
	}

	return args, nil, nil
}

func (r *mqlPrivatekey) id() (string, error) {
	// TODO: use path or hash depending on initialization

	file := r.GetFile()
	if file.Error != nil {
		return "", file.Error
	}
	if file.Data == nil {
		return "", errors.New("no file provided")
	}

	return "privatekey:" + file.Data.Path.Data, nil
}

// parseKey parses the resource's PEM payload into a Go crypto private key once
// and caches the result, so the independently lazy-loaded publicKeyAlgorithm and
// publicKeyBits accessors share a single parse per query.
func (r *mqlPrivatekey) parseKey() (any, error) {
	r.parseOnce.Do(func() {
		r.parsedKey, r.parseErr = r.doParseKey()
	})
	return r.parsedKey, r.parseErr
}

// doParseKey parses the resource's PEM payload into a Go crypto private key.
// Encrypted keys (which require a passphrase to introspect) return a nil key
// and a nil error, letting callers report empty/zero values instead of failing.
func (r *mqlPrivatekey) doParseKey() (any, error) {
	pem := r.GetPem()
	if pem.Error != nil {
		return nil, pem.Error
	}
	if pem.Data == "" {
		return nil, nil
	}

	key, err := ssh.ParseRawPrivateKey([]byte(pem.Data))
	if err != nil {
		// Encrypted keys cannot be introspected without the passphrase; treat
		// them as "unknown" rather than a hard failure.
		var passphraseErr *ssh.PassphraseMissingError
		if errors.As(err, &passphraseErr) {
			return nil, nil
		}
		return nil, err
	}

	return key, nil
}

func (r *mqlPrivatekey) publicKeyAlgorithm() (string, error) {
	key, err := r.parseKey()
	if err != nil {
		return "", err
	}

	switch key.(type) {
	case *rsa.PrivateKey:
		return "RSA", nil
	case *ecdsa.PrivateKey:
		return "ECDSA", nil
	case ed25519.PrivateKey, *ed25519.PrivateKey:
		return "Ed25519", nil
	case *dsa.PrivateKey:
		return "DSA", nil
	default:
		return "", nil
	}
}

func (r *mqlPrivatekey) publicKeyBits() (int64, error) {
	key, err := r.parseKey()
	if err != nil {
		return 0, err
	}

	switch k := key.(type) {
	case *rsa.PrivateKey:
		return int64(k.N.BitLen()), nil
	case *ecdsa.PrivateKey:
		return int64(k.Curve.Params().BitSize), nil
	case ed25519.PrivateKey, *ed25519.PrivateKey:
		return 256, nil
	case *dsa.PrivateKey:
		return int64(k.P.BitLen()), nil
	default:
		return 0, nil
	}
}
