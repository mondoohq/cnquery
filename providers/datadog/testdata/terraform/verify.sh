#!/bin/bash
# Verify the mql Datadog provider against live infrastructure.
# Prerequisites:
#   - DD_API_KEY and DD_APP_KEY set
#   - cnspec installed with datadog provider
#   - Terraform resources applied (see main.tf)
#
# Usage:
#   cd providers/datadog/testdata/terraform
#   terraform apply -var="dd_api_key=$DD_API_KEY" -var="dd_app_key=$DD_APP_KEY"
#   ./verify.sh
#   terraform destroy -var="dd_api_key=$DD_API_KEY" -var="dd_app_key=$DD_APP_KEY"

set -eo pipefail

if [ -z "${DD_API_KEY:-}" ]; then
  echo "Error: DD_API_KEY is not set"
  exit 1
fi
if [ -z "${DD_APP_KEY:-}" ]; then
  echo "Error: DD_APP_KEY is not set"
  exit 1
fi

RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

pass=0
fail=0

run_query() {
  local desc="$1"
  local query="$2"
  local result
  local exit_code=0
  result=$(cnspec run datadog --api-key "$DD_API_KEY" --app-key "$DD_APP_KEY" -c "$query" 2>&1) || exit_code=$?

  local filtered
  filtered=$(echo "$result" | grep -v "loaded configuration from")

  if [ "$exit_code" -ne 0 ] || echo "$filtered" | grep -qi "error occurred\|panic\|unable to"; then
    echo -e "${RED}FAIL${NC}: $desc"
    echo "  Query: $query"
    echo "$filtered" | head -5 | sed 's/^/  /'
    fail=$((fail + 1))
  else
    echo -e "${GREEN}PASS${NC}: $desc"
    pass=$((pass + 1))
  fi
}

echo "=== mql Datadog Provider Verification ==="
echo ""

# Users
run_query "list users" "datadog.users { id email status }"
run_query "user details" "datadog.users { name handle verified disabled }"

# Roles
run_query "list roles" "datadog.roles { id name userCount }"

# Monitors
run_query "list monitors" "datadog.monitors { id name type overallState }"
run_query "monitor details" "datadog.monitors { query message tags priority }"
run_query "monitor config" "datadog.monitors { notifyNoData options }"

# Dashboards
run_query "list dashboards" "datadog.dashboards { id title layoutType }"
run_query "dashboard details" "datadog.dashboards { description authorHandle isReadOnly }"

# Synthetics
run_query "list synthetics" "datadog.syntheticsTests { publicId name type status }"
run_query "synthetics details" "datadog.syntheticsTests { subtype tags locations config }"

# SLOs
run_query "list SLOs" "datadog.slos { id name type targetThreshold }"
run_query "SLO details" "datadog.slos { description tags timeframe warningThreshold }"

# Log Indexes
run_query "list log indexes" "datadog.logIndexes { name filter numRetentionDays }"
run_query "log index details" "datadog.logIndexes { dailyLimit isRateLimited exclusionFilters }"

# Security Rules
run_query "list security rules" "datadog.securityRules { id name type isEnabled }"
run_query "security rule details" "datadog.securityRules { isDefault tags cases }"

# Downtimes
run_query "list downtimes" "datadog.downtimes { id status scope }"
run_query "downtime details" "datadog.downtimes { message displayTimezone notifyEndStates }"

# API Keys
run_query "list API keys" "datadog.apiKeys { id name last4 }"

# Application Keys
run_query "list app keys" "datadog.applicationKeys { id name last4 scopes }"

# IP Allowlist
run_query "IP allowlist enabled" "datadog.ipAllowlistEnabled"
run_query "IP allowlist entries" "datadog.ipAllowlistEntries"

# AWS Integration
run_query "AWS integrations" "datadog.integrationAwsAccounts { accountId metricsEnabled logsEnabled }"

# Teams
run_query "list teams" "datadog.teams { id name handle userCount }"

# Sensitive Data Scanner
run_query "scanner groups" "datadog.sensitiveDataScannerGroups { id name isEnabled productList }"

# Security Filters
run_query "security filters" "datadog.securityFilters { id name query isEnabled }"

# Security Suppressions
run_query "security suppressions" "datadog.securitySuppressions { id name enabled ruleQuery }"

# Service Accounts
run_query "service accounts" "datadog.serviceAccounts { id email name disabled }"

# Logs Archives
run_query "logs archives" "datadog.logsArchives { id name state destinationType }"

# RUM Applications
run_query "RUM apps" "datadog.rumApplications { id name type isActive }"

# Synthetics Global Variables
run_query "synthetics variables" "datadog.syntheticsGlobalVariables { id name description }"

# Synthetics Private Locations
run_query "synthetics locations" "datadog.syntheticsPrivateLocations { id name }"

echo ""
echo "=== Results: ${pass} passed, ${fail} failed ==="

if [ "$fail" -gt 0 ]; then
  exit 1
fi
