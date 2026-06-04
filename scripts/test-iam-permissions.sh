#!/usr/bin/env bash
#
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# test-iam-permissions.sh — provision a throwaway IAM role/custom-role from the
# permission lists mql generates, so you can confirm every permission string is
# real and grantable against the live cloud.
#
# The permission lists come straight from the provider *.permissions.json files:
#   providers/aws/resources/aws.permissions.json   (902 perms)
#   providers/gcp/resources/gcp.permissions.json   (254 project + 5 org perms)
#
# Requirements: bash, jq, and the relevant authenticated CLI (aws / gcloud).
#
# Usage:
#   scripts/test-iam-permissions.sh aws-validate   # validate AWS actions, creates nothing
#   scripts/test-iam-permissions.sh aws-create     # create role + 7 managed policies
#   scripts/test-iam-permissions.sh aws-delete     # tear the AWS resources back down
#   scripts/test-iam-permissions.sh gcp-create     # create project-level custom role
#   scripts/test-iam-permissions.sh gcp-create-org <ORG_ID>   # create org-level custom role
#   scripts/test-iam-permissions.sh gcp-delete     # delete project-level custom role
#   scripts/test-iam-permissions.sh gcp-delete-org <ORG_ID>   # delete org-level custom role
#
set -euo pipefail

# Resolve repo root from this script's location so it runs from anywhere.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

AWS_PERMS="$REPO_ROOT/providers/aws/resources/aws.permissions.json"
GCP_PERMS="$REPO_ROOT/providers/gcp/resources/gcp.permissions.json"

AWS_ROLE="mql-perms-test"
AWS_POLICY_PREFIX="mql-readonly"
AWS_CHUNK=150          # actions per managed policy (keeps each under the 6,144-char limit)

GCP_ROLE_ID="mqlReadonly"
GCP_ORG_ROLE_ID="mqlReadonlyOrg"

TMP="${TMPDIR:-/tmp}"

die() { echo "error: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "required tool '$1' not found in PATH"; }

# ---------------------------------------------------------------------------
# AWS
# ---------------------------------------------------------------------------

# Emit one policy JSON file per chunk; echoes the file paths it wrote.
aws_build_policies() {
  need jq
  [[ -f "$AWS_PERMS" ]] || die "missing $AWS_PERMS"
  local total chunk start n=0
  total=$(jq '.permissions | length' "$AWS_PERMS")
  chunk=$AWS_CHUNK
  rm -f "$TMP"/mql_aws_policy_*.json
  for start in $(seq 0 "$chunk" $((total - 1))); do
    n=$((n + 1))
    jq -c --argjson s "$start" --argjson c "$chunk" \
      '{Version:"2012-10-17",Statement:[{Sid:"mqlReadOnly",Effect:"Allow",Action:.permissions[$s:($s+$c)],Resource:"*"}]}' \
      "$AWS_PERMS" > "$TMP/mql_aws_policy_$n.json"
  done
  echo "$n"
}

aws_validate() {
  need aws
  local count i
  count=$(aws_build_policies)
  echo ">> Validating $count policy chunks with IAM Access Analyzer (creates nothing)..."
  for i in $(seq 1 "$count"); do
    echo "--- chunk $i ---"
    aws accessanalyzer validate-policy \
      --policy-type IDENTITY_POLICY \
      --policy-document "file://$TMP/mql_aws_policy_$i.json" \
      --query 'findings[].{type:findingType,issue:issueCode,detail:findingDetails}' \
      --output table || true
  done
  echo ">> Done. Empty tables above == no findings == all actions valid."
}

aws_create() {
  need aws
  local acct count i arn
  acct=$(aws sts get-caller-identity --query Account --output text)

  cat > "$TMP/mql-trust.json" <<EOF
{"Version":"2012-10-17","Statement":[{"Effect":"Allow",
  "Principal":{"AWS":"arn:aws:iam::${acct}:root"},"Action":"sts:AssumeRole"}]}
EOF

  echo ">> Creating role $AWS_ROLE in account $acct ..."
  aws iam create-role --role-name "$AWS_ROLE" \
    --assume-role-policy-document "file://$TMP/mql-trust.json" >/dev/null

  count=$(aws_build_policies)
  echo ">> Attaching $count managed policies ..."
  for i in $(seq 1 "$count"); do
    arn=$(aws iam create-policy \
            --policy-name "${AWS_POLICY_PREFIX}-$i" \
            --policy-document "file://$TMP/mql_aws_policy_$i.json" \
            --query 'Policy.Arn' --output text)
    aws iam attach-role-policy --role-name "$AWS_ROLE" --policy-arn "$arn"
    echo "   attached $arn"
  done
  echo ">> Created role $AWS_ROLE with $count managed policies."
}

aws_delete() {
  need aws
  local acct count i arn
  acct=$(aws sts get-caller-identity --query Account --output text)
  # Recompute chunk count the same way create did, so we delete exactly what we made.
  count=$(aws_build_policies)
  echo ">> Detaching/deleting $count managed policies ..."
  for i in $(seq 1 "$count"); do
    arn="arn:aws:iam::${acct}:policy/${AWS_POLICY_PREFIX}-$i"
    aws iam detach-role-policy --role-name "$AWS_ROLE" --policy-arn "$arn" 2>/dev/null || true
    aws iam delete-policy --policy-arn "$arn" 2>/dev/null || true
  done
  echo ">> Deleting role $AWS_ROLE ..."
  aws iam delete-role --role-name "$AWS_ROLE" 2>/dev/null || true
  echo ">> AWS teardown complete."
}

# ---------------------------------------------------------------------------
# GCP
# ---------------------------------------------------------------------------

# $1 = jq filter selecting the permission array (.permissions or .org_level_permissions)
# $2 = output yaml path  ;  $3 = role title
gcp_build_role_yaml() {
  need jq
  [[ -f "$GCP_PERMS" ]] || die "missing $GCP_PERMS"
  jq -r --arg title "$3" \
    "\"title: \" + \$title + \"\ndescription: Read-only permissions required by mql\nstage: GA\nincludedPermissions:\n\" + ($1 | map(\"- \" + .) | join(\"\n\"))" \
    "$GCP_PERMS" > "$2"
}

# Partition our project-level permissions against the live custom-role catalog.
# Writes three sorted files under $TMP:
#   gcp_grantable.txt    - in the catalog AND usable in a custom role
#   gcp_notsupported.txt - real permissions GCP only grants via predefined roles
#   gcp_absent.txt       - not testable on this project (API disabled, org-scoped, or typo)
_gcp_partition() {
  local project="$1"
  echo ">> Fetching testable permissions for project $project ..." >&2
  gcloud iam list-testable-permissions \
    "//cloudresourcemanager.googleapis.com/projects/$project" \
    --format=json > "$TMP/gcp_testable.json"

  jq -r '.permissions[]' "$GCP_PERMS" | sort -u > "$TMP/gcp_ours.txt"
  # Supported in custom roles (absent field == SUPPORTED).
  jq -r '.[] | select(.customRolesSupportLevel != "NOT_SUPPORTED") | .name' \
    "$TMP/gcp_testable.json" | sort -u > "$TMP/gcp_supported.txt"
  # Explicitly NOT supported in custom roles.
  jq -r '.[] | select(.customRolesSupportLevel == "NOT_SUPPORTED") | .name' \
    "$TMP/gcp_testable.json" | sort -u > "$TMP/gcp_ns_catalog.txt"

  comm -12 "$TMP/gcp_ours.txt" "$TMP/gcp_supported.txt"  > "$TMP/gcp_grantable.txt"
  comm -12 "$TMP/gcp_ours.txt" "$TMP/gcp_ns_catalog.txt" > "$TMP/gcp_notsupported.txt"
  comm -23 "$TMP/gcp_ours.txt" \
    <(sort -u "$TMP/gcp_supported.txt" "$TMP/gcp_ns_catalog.txt") > "$TMP/gcp_absent.txt"
}

gcp_create() {
  need gcloud; need jq
  local project skipped n
  project=$(gcloud config get-value project 2>/dev/null)
  [[ -n "$project" ]] || die "no active gcloud project (run: gcloud config set project <id>)"
  _gcp_partition "$project"

  skipped=$(cat "$TMP/gcp_notsupported.txt" "$TMP/gcp_absent.txt")
  if [[ -n "$skipped" ]]; then
    n=$(printf '%s\n' "$skipped" | grep -c .)
    echo ">> Skipping $n permission(s) that can't live in a custom role"
    echo "   (NOT_SUPPORTED → use a predefined role; or API not enabled here):"
    printf '%s\n' "$skipped" | sed 's/^/   - /'
  fi

  {
    echo "title: mql-readonly"
    echo "description: Read-only permissions required by mql"
    echo "stage: GA"
    echo "includedPermissions:"
    sed 's/^/- /' "$TMP/gcp_grantable.txt"
  } > "$TMP/mql_gcp_role.yaml"

  echo ">> Creating project-level custom role $GCP_ROLE_ID in $project ($(wc -l < "$TMP/gcp_grantable.txt" | tr -d ' ') perms) ..."
  gcloud iam roles create "$GCP_ROLE_ID" --project="$project" \
    --file="$TMP/mql_gcp_role.yaml"
  echo ">> Done."
}

gcp_create_org() {
  need gcloud
  local org="${1:-}"
  [[ -n "$org" ]] || die "usage: $0 gcp-create-org <ORG_ID>"
  gcp_build_role_yaml ".org_level_permissions" "$TMP/mql_gcp_org_role.yaml" "mql-readonly-org"
  echo ">> Creating org-level custom role $GCP_ORG_ROLE_ID in org $org ..."
  gcloud iam roles create "$GCP_ORG_ROLE_ID" --organization="$org" \
    --file="$TMP/mql_gcp_org_role.yaml"
  echo ">> Done."
}

gcp_delete() {
  need gcloud
  local project
  project=$(gcloud config get-value project 2>/dev/null)
  [[ -n "$project" ]] || die "no active gcloud project"
  echo ">> Deleting project-level custom role $GCP_ROLE_ID in $project ..."
  gcloud iam roles delete "$GCP_ROLE_ID" --project="$project" --quiet
  echo ">> Done (role is soft-deleted; GCP purges it after ~7 days)."
}

gcp_delete_org() {
  need gcloud
  local org="${1:-}"
  [[ -n "$org" ]] || die "usage: $0 gcp-delete-org <ORG_ID>"
  echo ">> Deleting org-level custom role $GCP_ORG_ROLE_ID in org $org ..."
  gcloud iam roles delete "$GCP_ORG_ROLE_ID" --organization="$org" --quiet
  echo ">> Done."
}

# Validate every project-level permission against the live custom-role catalog
# in one pass, instead of discovering rejects one at a time via role-create.
# Splits the results into three buckets so a real bug (a permission GCP has never
# heard of) is not confused with the two benign cases (predefined-role-only, or
# an API that simply is not enabled on this project).
gcp_validate() {
  need gcloud; need jq
  local project ns ab
  project=$(gcloud config get-value project 2>/dev/null)
  [[ -n "$project" ]] || die "no active gcloud project (run: gcloud config set project <id>)"
  _gcp_partition "$project"

  ns=$(cat "$TMP/gcp_notsupported.txt")
  ab=$(cat "$TMP/gcp_absent.txt")

  if [[ -n "$ns" ]]; then
    echo "ℹ️  Required, but NOT_SUPPORTED in custom roles — grant via a predefined role (e.g. roles/<svc>.viewer):"
    sed 's/^/   - /' "$TMP/gcp_notsupported.txt"
  fi
  if [[ -n "$ab" ]]; then
    echo "⚠️  Not in this project's testable catalog — API not enabled here, OR org-scoped, OR a typo. Verify each:"
    sed 's/^/   - /' "$TMP/gcp_absent.txt"
  fi
  if [[ -z "$ns$ab" ]]; then
    echo ">> All $(wc -l < "$TMP/gcp_ours.txt" | tr -d ' ') project-level permissions are grantable in a custom role. ✓"
  else
    echo ">> $(wc -l < "$TMP/gcp_grantable.txt" | tr -d ' ') grantable in a custom role; see notes above for the rest."
  fi
}

# ---------------------------------------------------------------------------

usage() {
  cat >&2 <<'USG'
Usage: test-iam-permissions.sh <command>

AWS:
  aws-validate            validate every AWS action with Access Analyzer (creates nothing)
  aws-create              create role mql-perms-test + 7 managed policies
  aws-delete              tear the AWS role + policies back down

GCP:
  gcp-validate            check all project perms against testable-permissions (creates nothing)
  gcp-create              create project-level custom role mqlReadonly
  gcp-create-org <ORG_ID> create org-level custom role mqlReadonlyOrg
  gcp-delete              delete the project-level custom role
  gcp-delete-org <ORG_ID> delete the org-level custom role
USG
  exit 1
}

case "${1:-}" in
  aws-validate)    aws_validate ;;
  aws-create)      aws_create ;;
  aws-delete)      aws_delete ;;
  gcp-validate)    gcp_validate ;;
  gcp-create)      gcp_create ;;
  gcp-create-org)  gcp_create_org "${2:-}" ;;
  gcp-delete)      gcp_delete ;;
  gcp-delete-org)  gcp_delete_org "${2:-}" ;;
  *)               usage ;;
esac
