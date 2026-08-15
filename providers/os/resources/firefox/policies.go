// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

// Package firefox resolves the enterprise policy configuration a managed
// Firefox installation runs under, and normalizes it into one shape that is
// the same on every platform.
//
// Firefox reads its policy set from a JSON file on all platforms and, on
// Windows, additionally from the registry. The two arrive in very different
// shapes — a nested JSON document on one side, a key/value tree with registry
// types on the other — and this package projects the registry side onto the
// JSON side so a single check works everywhere.
//
// The resolution rules below are not invented; they follow what Firefox itself
// does in toolkit/components/enterprisepolicies. Deviations are called out at
// the point they are made.
package firefox

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Kind names the sort of input an enterprise policy configuration was read
// from.
const (
	KindFile     = "file"
	KindRegistry = "registry"
)

// Source is one input to the effective configuration, before merging.
type Source struct {
	// Kind is KindFile or KindRegistry.
	Kind string
	// Path is the filesystem path or the registry key this configuration was
	// read from.
	Path string
	// Params is the normalized policy set this source declares on its own.
	Params map[string]any
}

// PolicyFileName is the name Firefox gives its policy document wherever it
// looks for one.
const PolicyFileName = "policies.json"

// SystemPolicyFile is the administrator-owned policy file on Linux.
//
// Firefox builds this path as SysConfD/policies/policies.json, where SysConfD
// is /etc/<lowercased application name> (xpcom/io/SpecialSystemDirectory.cpp,
// GetUnixSystemConfigDir). The name comes from the application, not from the
// package: Debian's firefox-esr package sets Name=Firefox and only
// RemotingName=firefox-esr in application.ini, so an ESR install reads
// /etc/firefox as well — verified against a Debian firefox-esr 140.13.0
// installation. There is deliberately no /etc/firefox-esr candidate; that
// directory is not a path Firefox consults.
const SystemPolicyFile = "/etc/firefox/policies/policies.json"

// linuxInstallPrefixes are the installation directories a distribution puts
// Firefox in.
//
// Firefox itself does not use a list. It reads
// dirname(realpath("/proc/self/exe")) + "/distribution/policies.json"
// (XREAppDist, nsXREDirProvider.cpp), so the location follows whichever binary
// is running. Off-host we have no running process to ask, so we probe the
// known layouts instead.
//
// The -esr entries carry the weight here. Debian and Ubuntu ship Firefox as
// the firefox-esr package, installed to /usr/lib/firefox-esr; a list covering
// only /usr/lib/firefox and /usr/lib64/firefox reads the two most widely
// deployed Linux Firefoxes as unconfigured, which is indistinguishable from a
// host where policy was never applied.
var linuxInstallPrefixes = []string{
	"/usr/lib/firefox",
	"/usr/lib64/firefox",
	"/usr/lib/firefox-esr",
	"/usr/lib64/firefox-esr",
	// Debian resolves /usr/lib/firefox-esr/distribution through a symlink to
	// here. Listing the target as well keeps the lookup working on a
	// connection that cannot follow a symlink out of the directory it reads.
	"/usr/share/firefox",
	"/usr/share/firefox-esr",
	"/opt/firefox",
	"/usr/local/lib/firefox",
	"/snap/firefox/current/usr/lib/firefox",
}

// darwinInstallPrefixes are the application bundles whose Resources directory
// Firefox resolves XREAppDist to on macOS.
var darwinInstallPrefixes = []string{
	"/Applications/Firefox.app/Contents/Resources",
}

// windowsInstallPrefixes are the default installation directories on Windows.
var windowsInstallPrefixes = []string{
	`C:\Program Files\Mozilla Firefox`,
	`C:\Program Files (x86)\Mozilla Firefox`,
}

// PolicyFileCandidates returns the policy files to probe on a platform, in the
// order Firefox would consult them. The first one that exists is the one that
// applies: Firefox reads exactly one policy file and never merges two
// (JSONPoliciesProvider._getConfigurationFile returns a single nsIFile, and
// _readData reads only that one), so a candidate later in the list is a
// fallback, not an addition.
//
// platform is "windows", "darwin", or anything else for Linux and the other
// unix-likes.
func PolicyFileCandidates(platform string) []string {
	switch platform {
	case "windows":
		return distributionFiles(windowsInstallPrefixes, `\`)
	case "darwin":
		// macOS has no /etc lookup — the SysConfD branch in
		// _getConfigurationFile is gated on the platform being linux.
		return distributionFiles(darwinInstallPrefixes, "/")
	default:
		// The administrator-owned file wins outright over the install prefix.
		return append([]string{SystemPolicyFile}, distributionFiles(linuxInstallPrefixes, "/")...)
	}
}

func distributionFiles(prefixes []string, sep string) []string {
	res := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		res = append(res, strings.Join([]string{prefix, "distribution", PolicyFileName}, sep))
	}
	return res
}

// policyDocument is the envelope Firefox expects: every policy lives under a
// single top-level "policies" object.
type policyDocument struct {
	Policies map[string]any `json:"policies"`
}

// ParsePolicyFile extracts the policy set from a policies.json document.
//
// It returns a nil map with a nil error when the document is well-formed but
// carries no policies — an empty file, or one whose "policies" object is
// absent or empty. Callers must treat that as "this source contributes
// nothing", which is a resolved answer and not a failure.
//
// A document that is not valid JSON is an error. Firefox itself logs and
// ignores a malformed file, but an audit wants to know the difference between
// "no policy is deployed here" and "a policy is deployed and the browser
// silently threw it away" — the second is a misconfiguration worth surfacing,
// and reporting it as the first would be a false all-clear.
func ParsePolicyFile(content []byte) (map[string]any, error) {
	if len(strings.TrimSpace(string(content))) == 0 {
		return nil, nil
	}

	var doc policyDocument
	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse Firefox policy file: %w", err)
	}
	if len(doc.Policies) == 0 {
		return nil, nil
	}
	return doc.Policies, nil
}

// Merge combines sources into the effective policy set.
//
// sources must be ordered by increasing precedence, so the last entry wins a
// conflicting key. The merge is shallow, at the top-level policy key only,
// which is what Firefox's CombinedProvider does:
//
//	// Combine policies with primaryProvider taking precedence.
//	// We only do this for top level policies.
//	this._policies = primaryProvider._policies;
//	for (let policyName of Object.keys(secondaryProvider.policies)) {
//	  if (!(policyName in this._policies)) {
//	    this._policies[policyName] = secondaryProvider.policies[policyName];
//	  }
//	}
//
// The consequence is worth stating because it surprises people: if the winning
// source sets a policy at all, the losing source's object for that same policy
// is discarded whole rather than merged into it. A registry that sets
// SanitizeOnShutdown\Cookies replaces the file's entire SanitizeOnShutdown
// object, including the keys it does not mention.
//
// Merge returns nil when no source contributes anything, so an unmanaged host
// resolves to a null policy set rather than an empty one.
func Merge(sources []Source) map[string]any {
	var res map[string]any
	for _, source := range sources {
		for name, value := range source.Params {
			if res == nil {
				res = map[string]any{}
			}
			res[name] = value
		}
	}
	return res
}

// Describe names where an effective configuration came from, for the resource's
// `source` field: "file", "registry", "file+registry", or "" when nothing
// contributed.
func Describe(sources []Source) string {
	var file, registry bool
	for _, source := range sources {
		switch source.Kind {
		case KindFile:
			file = true
		case KindRegistry:
			registry = true
		}
	}

	switch {
	case file && registry:
		return KindFile + "+" + KindRegistry
	case file:
		return KindFile
	case registry:
		return KindRegistry
	default:
		return ""
	}
}
