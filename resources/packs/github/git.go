package github

import (
	"github.com/google/go-github/v47/github"
	"go.mondoo.com/cnquery/resources"
)

func (g *mqlGitGpgSignature) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.gpgSignature/" + sha, nil
}

func newMqlGitGpgSignature(runtime *resources.Runtime, sha string, a *github.SignatureVerification) (interface{}, error) {
	return runtime.CreateResource("git.gpgSignature",
		"sha", sha,
		"reason", a.GetReason(),
		"verified", a.GetVerified(),
		"payload", a.GetPayload(),
		"signature", a.GetSignature(),
	)
}

func (g *mqlGitCommitAuthor) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.commitAuthor/" + sha, nil
}

func newMqlGitAuthor(runtime *resources.Runtime, sha string, a *github.CommitAuthor) (interface{}, error) {
	date := a.GetDate()
	return runtime.CreateResource("git.commitAuthor",
		"sha", sha,
		"name", a.GetName(),
		"email", a.GetEmail(),
		"date", &date,
	)
}

func (g *mqlGitCommit) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.commit/" + sha, nil
}

func newMqlGitCommit(runtime *resources.Runtime, sha string, c *github.Commit) (interface{}, error) {
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

	return runtime.CreateResource("git.commit",
		"sha", sha,
		"message", c.GetMessage(),
		"author", author,
		"committer", committer,
		"signatureVerification", signatureVerification,
	)
}
