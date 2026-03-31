#!/usr/bin/env python3
"""Validate GCP permissions in a permissions JSON file against the IAM API.

Uses the queryTestablePermissions API to fetch all real GCP IAM permissions,
then compares them against the permissions listed in a gcp.permissions.json file.

Because some permissions are only testable at certain resource levels (e.g.
resourcemanager.folders.* at the org level, not the project level), the script
queries both project and organization scopes.

Requirements:
  - gcloud CLI installed and authenticated
  - A GCP project (and optionally an organization) to query permissions against

Usage:
  python3 validate_permissions.py <permissions.json> [--project PROJECT_ID] [--org ORG_ID]

Example:
  python3 validate_permissions.py providers/gcp/resources/gcp.permissions.json
  python3 validate_permissions.py gcp.permissions.json --project my-project --org 123456789
"""

import argparse
import json
import subprocess
import sys
import urllib.error
import urllib.request
from difflib import get_close_matches


def get_access_token():
    """Get an access token from gcloud."""
    result = subprocess.run(
        ["gcloud", "auth", "print-access-token"],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        print("Error: Failed to get access token. Run 'gcloud auth login' first.", file=sys.stderr)
        sys.exit(1)
    return result.stdout.strip()


def get_default_project():
    """Get the default gcloud project."""
    result = subprocess.run(
        ["gcloud", "config", "get-value", "project"],
        capture_output=True, text=True,
    )
    if result.returncode != 0 or not result.stdout.strip():
        return None
    return result.stdout.strip()


def get_org_id():
    """Get the first organization ID the caller has access to."""
    result = subprocess.run(
        ["gcloud", "organizations", "list", "--format=value(ID)", "--limit=1"],
        capture_output=True, text=True,
    )
    if result.returncode != 0 or not result.stdout.strip():
        return None
    return result.stdout.strip().split("\n")[0]


def fetch_testable_permissions(full_resource_name, token):
    """Fetch all testable permissions for a given resource from the IAM API."""
    url = "https://iam.googleapis.com/v1/permissions:queryTestablePermissions"

    all_permissions = []
    page_token = None

    while True:
        body = {
            "fullResourceName": full_resource_name,
            "pageSize": 1000,
        }
        if page_token:
            body["pageToken"] = page_token

        req = urllib.request.Request(
            url,
            data=json.dumps(body).encode(),
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
            },
            method="POST",
        )

        try:
            with urllib.request.urlopen(req) as resp:
                data = json.loads(resp.read())
        except urllib.error.HTTPError as e:
            error_body = e.read().decode()
            print(f"  Warning: API request failed for {full_resource_name} ({e.code}): {error_body}", file=sys.stderr)
            return all_permissions

        permissions = data.get("permissions", [])
        for p in permissions:
            all_permissions.append(p["name"])

        page_token = data.get("nextPageToken")
        if not page_token:
            break

    return all_permissions


def fetch_all_permissions(project_id, org_id=None):
    """Fetch all testable permissions from project and org scopes."""
    token = get_access_token()

    # Always query project-level permissions
    project_resource = f"//cloudresourcemanager.googleapis.com/projects/{project_id}"
    print(f"  Querying project-level permissions ({project_id})...", file=sys.stderr)
    all_permissions = fetch_testable_permissions(project_resource, token)
    print(f"    Got {len(all_permissions)} permissions", file=sys.stderr)

    # Query org-level permissions to catch folder/org-scoped permissions
    if org_id:
        org_resource = f"//cloudresourcemanager.googleapis.com/organizations/{org_id}"
        print(f"  Querying org-level permissions ({org_id})...", file=sys.stderr)
        org_permissions = fetch_testable_permissions(org_resource, token)
        print(f"    Got {len(org_permissions)} permissions", file=sys.stderr)
        all_permissions.extend(org_permissions)

    return all_permissions


def load_permissions_json(path):
    """Load the permissions JSON file and return the permissions list."""
    with open(path) as f:
        data = json.load(f)
    return data.get("permissions", [])


def find_suggestions(permission, valid_set, n=3):
    """Find similar valid permissions as suggestions for an invalid one."""
    parts = permission.split(".")
    if len(parts) < 2:
        return []

    service = parts[0]

    # Try close matches within the same service first
    same_service = [p for p in valid_set if p.startswith(service + ".")]
    matches = get_close_matches(permission, same_service, n=n, cutoff=0.6)
    if matches:
        return matches

    # Broaden to all permissions
    return get_close_matches(permission, list(valid_set), n=n, cutoff=0.5)


def main():
    parser = argparse.ArgumentParser(
        description="Validate GCP permissions against the IAM API.",
    )
    parser.add_argument(
        "permissions_file",
        help="Path to the gcp.permissions.json file",
    )
    parser.add_argument(
        "--project",
        help="GCP project ID (default: gcloud default project)",
    )
    parser.add_argument(
        "--org",
        help="GCP organization ID (default: auto-detected from gcloud)",
    )
    parser.add_argument(
        "--dump-valid",
        metavar="FILE",
        help="Write all valid GCP permissions to a file (one per line)",
    )
    args = parser.parse_args()

    # Resolve project
    project_id = args.project or get_default_project()
    if not project_id:
        print("Error: No project specified and no default gcloud project set.", file=sys.stderr)
        print("Use --project PROJECT_ID or run 'gcloud config set project PROJECT_ID'.", file=sys.stderr)
        sys.exit(1)

    # Resolve org
    org_id = args.org
    if not org_id:
        print("Auto-detecting organization ID...", file=sys.stderr)
        org_id = get_org_id()
        if org_id:
            print(f"  Found organization: {org_id}", file=sys.stderr)
        else:
            print("  No organization found. Org-level permissions (resourcemanager.folders.*, etc.) may show as invalid.", file=sys.stderr)

    # Load our permissions
    print(f"Loading permissions from {args.permissions_file}...")
    our_permissions = load_permissions_json(args.permissions_file)
    print(f"  Found {len(our_permissions)} permissions in file.")

    # Fetch all valid GCP permissions
    print(f"Fetching all testable GCP permissions...")
    all_permissions = fetch_all_permissions(project_id, org_id)
    valid_set = set(all_permissions)
    print(f"  Total unique valid permissions: {len(valid_set)}")

    if args.dump_valid:
        with open(args.dump_valid, "w") as f:
            for p in sorted(valid_set):
                f.write(p + "\n")
        print(f"  Wrote valid permissions to {args.dump_valid}")

    # Compare
    invalid = []
    valid = []
    for perm in our_permissions:
        if perm in valid_set:
            valid.append(perm)
        else:
            invalid.append(perm)

    # Report
    print()
    print(f"Results: {len(valid)} valid, {len(invalid)} invalid out of {len(our_permissions)} total")
    print()

    if invalid:
        print("INVALID PERMISSIONS:")
        print("-" * 70)
        for perm in invalid:
            suggestions = find_suggestions(perm, valid_set)
            print(f"  {perm}")
            if suggestions:
                print(f"    Did you mean: {', '.join(suggestions)}")
            else:
                print(f"    No close matches found")
        print()
    else:
        print("All permissions are valid!")

    return 1 if invalid else 0


if __name__ == "__main__":
    sys.exit(main())
