package services

import (
	"github.com/google/go-github/v45/github"
	"go.mondoo.io/mondoo/lumi"
)

func (g *lumiGitGpgSignature) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.gpgSignature/" + sha, nil
}

func newLumiGitGpgSignature(runtime *lumi.Runtime, sha string, a *github.SignatureVerification) (interface{}, error) {
	return runtime.CreateResource("git.gpgSignature",
		"sha", sha,
		"reason", a.GetReason(),
		"verified", a.GetVerified(),
		"payload", a.GetPayload(),
		"signature", a.GetSignature(),
	)
}

func (g *lumiGitCommitAuthor) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.commitAuthor/" + sha, nil
}

func newLumiGitAuthor(runtime *lumi.Runtime, sha string, a *github.CommitAuthor) (interface{}, error) {
	date := a.GetDate()
	return runtime.CreateResource("git.commitAuthor",
		"sha", sha,
		"name", a.GetName(),
		"email", a.GetEmail(),
		"date", &date,
	)
}

func (g *lumiGitCommit) id() (string, error) {
	sha, err := g.Sha()
	if err != nil {
		return "", err
	}
	return "git.commit/" + sha, nil
}

func newLumiGitCommit(runtime *lumi.Runtime, sha string, c *github.Commit) (interface{}, error) {
	// we have to pass-in the sha because the sha is often not set c.GetSHA()
	author, err := newLumiGitAuthor(runtime, sha, c.GetAuthor())
	if err != nil {
		return nil, err
	}

	committer, err := newLumiGitAuthor(runtime, sha, c.GetCommitter())
	if err != nil {
		return nil, err
	}

	signatureVerification, err := newLumiGitGpgSignature(runtime, sha, c.GetVerification())
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
