#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Emit the set of provider directories under providers/ that changed
# relative to a base ref, or signal that ALL providers must be re-built
# because a shared dependency (SDK, Makefile, root sources, CI config)
# changed.
#
# Inputs (env):
#   BASE_SHA      Optional. The ref to diff against. If unset, falls back
#                 to `git merge-base HEAD origin/main`.
#   GITHUB_OUTPUT Optional. When set, results are appended for GHA jobs.
#
# Outputs (GHA $GITHUB_OUTPUT, and stdout):
#   all=true|false        — true when every provider must be processed
#   providers="a b c"     — space-separated list (empty when all=true)

set -euo pipefail

# Patterns whose change forces a full re-run across every provider.
# Anything outside providers/<name>/ that providers depend on belongs here.
# Note: `providers/[^/]+\.go$` catches every .go file directly under providers/
# (coordinator, runtime, registry, sbom, builtin*, etc.) — they're all shared
# plumbing that every provider relies on.
FORCE_ALL_PATTERN='^(providers-sdk/|providers/core/|providers/[^/]+\.go$|providers\.yaml$|Makefile$|\.github/env$|scripts/changed-providers\.sh$|\.github/workflows/(pr-test-generated-files\.yaml|pr-test-lint\.yml|pr-extended-linting\.yml|reusable-lint-providers\.yml)$|go\.mod$|go\.sum$|[^/]+\.go$)'

base="${BASE_SHA:-}"
if [ -z "$base" ]; then
  git fetch origin main --depth=100 >/dev/null 2>&1 || true
  base=$(git merge-base HEAD origin/main 2>/dev/null || true)
fi

emit() {
  local all="$1"
  local providers="$2"
  echo "all=$all"
  echo "providers=$providers"
  if [ -n "${GITHUB_OUTPUT:-}" ]; then
    {
      echo "all=$all"
      echo "providers=$providers"
    } >> "$GITHUB_OUTPUT"
  fi
}

if [ -z "$base" ]; then
  echo "warn: no base ref available, falling back to all providers" >&2
  emit "true" ""
  exit 0
fi

# Make sure the base commit is reachable locally (shallow clones).
git fetch origin "$base" --depth=1 >/dev/null 2>&1 || true

# Triple-dot diff: changes introduced on HEAD relative to the merge-base
# with `base`. On a push directly to main the merge-base resolves to HEAD,
# producing no files and an empty providers list. The callers treat an
# empty list as "lint/regen all" (see the `[ -z "$CHANGED_PROVIDERS" ]`
# branch in pr-test-generated-files.yaml and the empty-input handling in
# reusable-lint-providers.yml), so this fallback is intentional.
changed_files=$(git diff --name-only "$base"...HEAD)

if printf '%s\n' "$changed_files" | grep -qE "$FORCE_ALL_PATTERN"; then
  emit "true" ""
  exit 0
fi

# Pick up changed providers/<name>/... entries, filter to ones that still exist.
changed=$(printf '%s\n' "$changed_files" \
  | awk -F/ '/^providers\// && NF > 2 { print $2 }' \
  | sort -u \
  | while read -r name; do
      [ -d "providers/$name" ] && echo "$name"
    done \
  | tr '\n' ' ' \
  | sed 's/[[:space:]]*$//')

emit "false" "$changed"
