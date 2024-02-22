// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"github.com/google/go-github/v59/github"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/plugin"
)

func (g *mqlGitGpgSignature) id() (string, error) {
	return "git.gpgSignature/" + g.Sha.Data, nil
}

func newMqlGitGpgSignature(runtime *plugin.Runtime, sha string, a *github.SignatureVerification) (interface{}, error) {
	return CreateResource(runtime, "git.gpgSignature", map[string]*llx.RawData{
		"sha":       llx.StringData(sha),
		"reason":    llx.StringData(a.GetReason()),
		"verified":  llx.BoolData(a.GetVerified()),
		"payload":   llx.StringData(a.GetPayload()),
		"signature": llx.StringData(a.GetSignature()),
	})
}

func (g *mqlGitCommitAuthor) id() (string, error) {
	return "git.commitAuthor/" + g.Sha.Data, nil
}

func newMqlGitAuthor(runtime *plugin.Runtime, sha string, a *github.CommitAuthor) (interface{}, error) {
	date := a.GetDate()
	return CreateResource(runtime, "git.commitAuthor", map[string]*llx.RawData{
		"sha":   llx.StringData(sha),
		"name":  llx.StringData(a.GetName()),
		"email": llx.StringData(a.GetEmail()),
		"date":  llx.TimeData(date.Time),
	})
}

func (g *mqlGitCommit) id() (string, error) {
	return "git.commit/" + g.Sha.Data, nil
}

func newMqlGitCommit(runtime *plugin.Runtime, sha string, c *github.Commit) (interface{}, error) {
	// we have to pass-in the sha because the sha is often not set c.GetSHA()
	author, err := newMqlGitAuthor(runtime, sha, c.GetAuthor())
	if err != nil {
		return nil, err
	}

	committer, err := newMqlGitAuthor(runtime, sha, c.GetCommitter())
	if err != nil {
		return nil, err
	}

	signatureVerification, err := newMqlGitGpgSignature(runtime, sha, c.GetVerification())
	if err != nil {
		return nil, err
	}

	return CreateResource(runtime, "git.commit", map[string]*llx.RawData{
		"sha":                   llx.StringData(sha),
		"message":               llx.StringData(c.GetMessage()),
		"author":                llx.AnyData(author),
		"committer":             llx.AnyData(committer),
		"signatureVerification": llx.AnyData(signatureVerification),
	})
}
