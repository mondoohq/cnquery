#!/usr/bin/env bash
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Determine base branch for the provider version-bump PR opened after
# an mql release.
#
# When main's own module major matches the incoming release's major,
# the PR opens against main. Otherwise it opens against the v{major}
# support branch (e.g. a v13.35.1 release while main is on v14 -> v13).
#
# Inputs:
#   $1 (required)  incoming release tag (e.g. v13.35.1 or 14.0.0-rc.1)
#   $MAIN_GOMOD    content of main's go.mod. If unset, fetched via
#                  `gh api` using $GH_REPO (owner/repo) and $GH_TOKEN.
#
# Output: prints the base branch to stdout ("main" or "v{major}").
#         Exits non-zero if main's module major cannot be determined.
set -euo pipefail

TAG="${1:?release tag required as first argument}"

V="${TAG#v}"
RELEASE_MAJOR="${V%%.*}"

if [ -z "${MAIN_GOMOD+set}" ]; then
  # MAIN_GOMOD not passed in at all — fetch it. An explicit empty value
  # is respected (and will fail the "Could not extract" check below), so
  # tests can exercise failure paths without touching the network.
  : "${GH_REPO:?GH_REPO or MAIN_GOMOD required}"
  MAIN_GOMOD=$(gh api "/repos/${GH_REPO}/contents/go.mod?ref=main" --jq '.content' | base64 -d)
fi

# module go.mondoo.com/mql/v13  -> 13
MAIN_MAJOR=$(printf '%s' "$MAIN_GOMOD" \
  | grep -oE '^module go\.mondoo\.com/mql/v[0-9]+' \
  | head -1 \
  | grep -oE '[0-9]+$' || true)

if [ -z "$MAIN_MAJOR" ]; then
  echo "Could not extract main's module major from go.mod" >&2
  exit 1
fi

if [ "$RELEASE_MAJOR" = "$MAIN_MAJOR" ]; then
  echo main
else
  echo "v${RELEASE_MAJOR}"
fi
