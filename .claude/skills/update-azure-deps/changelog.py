#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Print Azure SDK CHANGELOG sections between two majors so the agent can spot breaking changes.

Usage:  python3 changelog.py <module_path_without_vN> <from_version> <to_version>

Example:
  python3 changelog.py \\
    github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/operationalinsights/armoperationalinsights \\
    v1.2.0 v2.0.2

Prints the CHANGELOG sections covering every release strictly newer than from_version,
up to and including to_version. Focuses output on "Breaking Changes" subsections.
"""
import re
import sys
import urllib.request


def encode_for_proxy(path: str) -> str:
    return "".join(("!" + c.lower()) if c.isupper() else c for c in path)


def fetch_changelog(module_path: str) -> str:
    """Fetch CHANGELOG.md from the raw GitHub URL.

    The Azure SDK Go repo lives at github.com/Azure/azure-sdk-for-go and the
    module path after `github.com/Azure/azure-sdk-for-go/` is the path inside the repo.
    """
    prefix = "github.com/Azure/azure-sdk-for-go/"
    if not module_path.startswith(prefix):
        raise SystemExit(f"unsupported module: {module_path}")
    sub = module_path[len(prefix):]
    url = f"https://raw.githubusercontent.com/Azure/azure-sdk-for-go/main/{sub}/CHANGELOG.md"
    with urllib.request.urlopen(url, timeout=30) as r:
        return r.read().decode()


def split_sections(text: str):
    """Yield (version_str, body) for each '## <version>' section."""
    parts = re.split(r"^## ", text, flags=re.MULTILINE)
    for p in parts[1:]:
        head, _, body = p.partition("\n")
        m = re.match(r"([0-9][^ ]*)", head)
        if not m:
            continue
        yield m.group(1), body.rstrip()


def cmp_semver(a: str, b: str) -> int:
    """Return -1/0/1 for a vs b. Treat pre-release as less than stable."""
    def parse(v):
        v = v.lstrip("v")
        main, _, pre = v.partition("-")
        nums = []
        for n in main.split("."):
            try:
                nums.append(int(n))
            except ValueError:
                nums.append(0)
        return nums, pre
    na, pa = parse(a)
    nb, pb = parse(b)
    if na != nb:
        return -1 if na < nb else 1
    # Stable > prerelease
    if pa == "" and pb != "":
        return 1
    if pa != "" and pb == "":
        return -1
    if pa != pb:
        return -1 if pa < pb else 1
    return 0


def main(module_path: str, from_ver: str, to_ver: str):
    text = fetch_changelog(module_path)
    sections = list(split_sections(text))
    if not sections:
        raise SystemExit("could not parse CHANGELOG")
    relevant = [
        (v, body) for v, body in sections
        if cmp_semver(v, from_ver) > 0 and cmp_semver(v, to_ver) <= 0
    ]
    if not relevant:
        print(f"# No CHANGELOG entries strictly between {from_ver} and {to_ver}")
        return
    print(f"# {module_path}: {from_ver} -> {to_ver}")
    print(f"# {len(relevant)} release(s) crossed\n")
    for v, body in relevant:
        print(f"## {v}")
        # Surface "### Breaking Changes" first; print Features Added too
        # so the agent can spot deprecations described as new fields.
        print(body)
        print()


if __name__ == "__main__":
    if len(sys.argv) != 4:
        print(f"usage: {sys.argv[0]} <module-path> <from-version> <to-version>", file=sys.stderr)
        sys.exit(2)
    main(sys.argv[1], sys.argv[2], sys.argv[3])
