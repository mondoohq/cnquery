// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package plugin

import (
	"net/url"
	"os"

	"github.com/go-git/go-git/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	inventory "go.mondoo.com/cnquery/v11/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v11/providers-sdk/v1/vault"
)

func NewGitClone(asset *inventory.Asset) (string, func(), error) {
	cc := asset.Connections[0]

	if len(cc.Options) == 0 {
		return "", nil, errors.New("missing URLs in options for HCL over Git connection")
	}

	user := ""
	token := ""
	for i := range cc.Credentials {
		cred := cc.Credentials[i]
		if cred.Type == vault.CredentialType_password {
			user = cred.User
			token = string(cred.Secret)
			if token == "" && cred.Password != "" {
				token = string(cred.Password)
			}
		}
	}

	gitUrl := ""

	// If a token is provided, it will be used to clone the repo
	// gitlab: git clone https://oauth2:ACCESS_TOKEN@somegitlab.com/vendor/package.git
	// if sshUrl := cc.Options["ssh-url"]; sshUrl != "" { ... not doing ssh url right now
	if httpUrl := cc.Options["http-url"]; httpUrl != "" {
		u, err := url.Parse(httpUrl)
		if err != nil {
			return "", nil, errors.New("failed to parse url for git repo: " + httpUrl)
		}

		if user != "" && token != "" {
			u.User = url.UserPassword(user, token)
		} else if token != "" {
			u.User = url.User(token)
		}

		gitUrl = u.String()
	}

	if gitUrl == "" {
		return "", nil, errors.New("missing url for git repo " + asset.Name)
	}

	path, closer, err := gitClone(gitUrl)
	if err != nil {
		return "", nil, err
	}
	return path, closer, nil
}

func gitClone(gitUrl string) (string, func(), error) {
	cloneDir, err := os.MkdirTemp(os.TempDir(), "cnquery-git-clone")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create temporary dir for git processing")
	}

	closer := func() {
		log.Info().Str("path", cloneDir).Msg("cleaning up git clone")
		if err = os.RemoveAll(cloneDir); err != nil {
			log.Error().Err(err).Msg("failed to remove temporary dir for git processing")
		}
	}

	// Note: DO NOT leak credentials into logs!!
	var infoUrl string
	if u, err := url.Parse(gitUrl); err == nil {
		if u.User != nil {
			u.User = url.User("_obfuscated_")
		}
		infoUrl = u.String()
	}

	log.Info().Str("url", infoUrl).Str("path", cloneDir).Msg("git clone")
	repo, err := git.PlainClone(cloneDir, false, &git.CloneOptions{
		URL:               gitUrl,
		Progress:          os.Stderr,
		Depth:             1,
		RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
	})
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to clone git repo "+infoUrl)
	}

	ref, err := repo.Head()
	if err != nil {
		closer()
		return "", nil, errors.Wrap(err, "failed to get head of git repo "+infoUrl)
	}

	log.Info().Str("url", infoUrl).Str("path", cloneDir).Str("head", ref.Hash().String()).Msg("finished git clone")

	return cloneDir, closer, nil
}
