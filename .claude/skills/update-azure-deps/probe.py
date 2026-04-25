#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Probe proxy.golang.org for latest stable major of every Azure dep in a provider's go.mod.

Usage:  python3 probe.py providers/azure/go.mod

Output (one line per dep):
  UP   <current path>@<current ver> -> <new path>@<new stable ver>
  OK   <path>@<current ver>         (already on latest stable major)
  BETA <path>@<current ver>         (new major exists only as beta/pseudo; current stays)

Exit code 0 always. The caller decides what to do with UP rows.
"""
import json
import re
import sys
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor, as_completed

# Module paths under these prefixes are considered "Azure deps".
AZURE_PREFIXES = ("github.com/Azure/", "github.com/AzureAD/")

# Pseudo-version pattern per https://go.dev/ref/mod#pseudo-versions
PSEUDO_RE = re.compile(r"-\d{14}-[0-9a-f]{12}$")
# Pre-release tag (anything after a dash that isn't a pseudo-version)
PRERELEASE_RE = re.compile(r"-(alpha|beta|rc|pre|dev)")


def is_stable(version: str) -> bool:
    if not version or not version.startswith("v"):
        return False
    if PSEUDO_RE.search(version):
        return False
    if PRERELEASE_RE.search(version):
        return False
    return True


def encode(path: str) -> str:
    """Encode for the Go module proxy: capital letters get prefixed with !."""
    return "".join(("!" + c.lower()) if c.isupper() else c for c in path)


def _fetch(url: str, retries: int = 2):
    """GET url, retrying only on transient (non-404) errors. Returns body bytes or None."""
    for attempt in range(retries):
        try:
            with urllib.request.urlopen(url, timeout=15) as r:
                return r.read()
        except urllib.error.HTTPError as e:
            if e.code == 404 or e.code == 410:
                return None  # path doesn't exist; no point retrying
            if attempt == retries - 1:
                return None
        except Exception:
            if attempt == retries - 1:
                return None
        time.sleep(0.4 * (attempt + 1))
    return None


def fetch_latest(path: str):
    body = _fetch(f"https://proxy.golang.org/{encode(path)}/@latest")
    if not body:
        return None
    try:
        return json.loads(body).get("Version")
    except Exception:
        return None


def fetch_versions(path: str):
    """Fetch all known versions; used to find latest stable when @latest returns a beta."""
    body = _fetch(f"https://proxy.golang.org/{encode(path)}/@v/list")
    if not body:
        return []
    return [v for v in body.decode().splitlines() if v]


def _semver_key(s: str):
    parts = s.lstrip("v").split(".")
    out = []
    for p in parts:
        try:
            out.append((0, int(p)))
        except ValueError:
            out.append((1, p))
    return out


def stable_from_versions(path: str):
    """Return the latest stable version at this path by listing all tags. None if path doesn't exist."""
    versions = [v for v in fetch_versions(path) if is_stable(v)]
    if not versions:
        return None
    versions.sort(key=_semver_key)
    return versions[-1]


def probe_major(path: str):
    """Return (latest_any, latest_stable) for a module path.
    latest_any is the @latest result (could be beta/pseudo). latest_stable is the newest stable.
    Both None if path doesn't exist. Single network round-trip in the common case."""
    latest_any = fetch_latest(path)
    if latest_any is None:
        return None, None  # path doesn't exist
    if is_stable(latest_any):
        return latest_any, latest_any
    # @latest is beta/pseudo; need to scan version list for a stable.
    return latest_any, stable_from_versions(path)


def parse_path(full_path: str):
    """Split 'github.com/foo/bar/v3' -> ('github.com/foo/bar', 3). Returns (base, current_major)."""
    m = re.match(r"^(.+)/v(\d+)$", full_path)
    if m:
        return m.group(1), int(m.group(2))
    return full_path, 1  # v0/v1 has no /vN suffix


def parse_gomod(path: str):
    """Yield (module_path, version) for each Azure dep in go.mod."""
    text = open(path).read()
    # match lines inside require blocks: "<path> <version>"
    for line in text.splitlines():
        line = line.strip().rstrip("// indirect").strip()
        m = re.match(r"^([^\s]+)\s+(v[0-9][^\s]*)", line)
        if not m:
            continue
        mod_path, ver = m.group(1), m.group(2)
        if any(mod_path.startswith(p) for p in AZURE_PREFIXES):
            yield mod_path, ver


PROBE_MAJOR_RANGE = 5  # how many majors above current to check


def probe_one(full_path: str, current_ver: str):
    base, current_major = parse_path(full_path)

    stable_highest = None
    beta_highest = None
    for n in range(current_major + 1, current_major + 1 + PROBE_MAJOR_RANGE):
        latest_any, latest_stbl = probe_major(f"{base}/v{n}")
        if latest_any is None:
            # Azure SDK majors are sequential — first 404 means we've gone past the latest.
            break
        if latest_stbl:
            stable_highest = (n, latest_stbl)
        elif not is_stable(latest_any):
            beta_highest = (n, latest_any)

    if stable_highest:
        n, v = stable_highest
        return ("UP", full_path, current_ver, f"{base}/v{n}", v)
    if beta_highest:
        n, v = beta_highest
        return ("BETA", full_path, current_ver, f"{base}/v{n}", v)

    # Same major: report latest patch within (single fetch, fast path).
    _, latest_patch = probe_major(full_path)
    if latest_patch and latest_patch != current_ver:
        return ("UP", full_path, current_ver, full_path, latest_patch)
    return ("OK", full_path, current_ver, "", "")


def format_row(row):
    status, cur_path, cur_ver, new_path, new_ver = row
    if status == "UP":
        return f"UP   {cur_path}@{cur_ver} -> {new_path}@{new_ver}"
    if status == "BETA":
        return f"BETA {cur_path}@{cur_ver}  (new major only as {new_ver}; staying)"
    return f"OK   {cur_path}@{cur_ver}"


def main(gomod_path: str):
    deps = list(parse_gomod(gomod_path))
    if not deps:
        print(f"no Azure deps found in {gomod_path}", file=sys.stderr)
        return
    print(f"# probing {len(deps)} Azure deps...", file=sys.stderr, flush=True)
    with ThreadPoolExecutor(max_workers=8) as ex:
        futs = {ex.submit(probe_one, p, v): (p, v) for p, v in deps}
        for f in as_completed(futs):
            print(format_row(f.result()), flush=True)


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} <path/to/go.mod>", file=sys.stderr)
        sys.exit(2)
    main(sys.argv[1])
