#!/usr/bin/env python3
# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
"""Find .lr field comments that enumerate values the SDK no longer agrees with.

    python3 enumdrift.py <provider.lr> <extracted-sdk-dir> [--field NAME]

Why this exists: `.lr` comments routinely spell out a field's possible values
("One of Primary, Readonly, Guard, or Temp"). Those lists are a promise to the
person writing a policy -- they decide which `where(state == "...")` clauses get
written -- and nothing keeps them honest when an SDK upgrade adds a state. The
list silently becomes wrong, and a policy that enumerates the documented values
starts missing rows it should match.

The check is a *comparison aid*, not a linter. It pairs each enumerated comment
with the SDK constant blocks whose values overlap, then reports what the SDK has
that the comment does not. Read the pairs before editing: a comment can list a
deliberate subset (the values we actually map), and the matcher can pair the
wrong constant block when several enums share vocabulary. Both are for a human
to judge, which is why the output is a report rather than a diff to apply.

Typical use, after extracting both SDK versions with modfetch.sh:

    python3 enumdrift.py providers/aws/resources/aws.lr mods/<new-sdk-dir>

Restrict to one field while iterating:

    python3 enumdrift.py providers/okta/resources/okta.lr mods/<dir> --field status
"""
import argparse
import os
import re
import sys

# Values enumerated in prose. `.lr` comments write these several ways:
#   One of A, B, or C.        One of "A" or "B".
#   such as A or B            (A, B, or C)
# Captured loosely, then split on separators, because the goal is to notice a
# missing value rather than to parse English.
ENUM_PROSE_RE = re.compile(
    r"(?:one of|such as|either|values?(?:\s+are)?:?)\s+(?P<body>[^.]*)", re.IGNORECASE
)

# A candidate value inside that prose: BAREWORD, "quoted", or `backticked`.
# Enum values in these APIs are overwhelmingly single tokens.
VALUE_RE = re.compile(r'[`"]?([A-Za-z][A-Za-z0-9_.-]{1,60})[`"]?')

# Words that show up inside an enumeration but are not values.
STOPWORDS = {
    "one", "of", "or", "and", "either", "such", "as", "the", "a", "an", "is", "are",
    "value", "values", "for", "to", "in", "on", "when", "which", "that", "this",
    "empty", "null", "unset", "none", "other", "others", "kinds", "etc", "example",
    "instance", "default", "defaults", "set", "not", "it", "its", "be", "by", "with",
}

# type X string  +  const XFoo X = "FOO" / `FOO`
GO_CONST_RE = re.compile(
    r"^\s*(?:const\s+)?([A-Za-z][A-Za-z0-9_]*)\s+([A-Za-z][A-Za-z0-9_.]*)\s*=\s*[`\"]([^`\"]+)[`\"]",
    re.MULTILINE,
)


def lr_fields_with_enums(lr_path: str):
    """Yield (field, line_no, values, comment) for each .lr field whose doc enumerates values."""
    lines = open(lr_path, encoding="utf-8", errors="replace").read().splitlines()
    out = []
    buf = []
    for i, raw in enumerate(lines):
        s = raw.strip()
        if s.startswith("//"):
            buf.append(s.lstrip("/").strip())
            continue
        # A field declaration is `name type` or `name() type`, possibly with tags.
        m = re.match(r"^([a-z][A-Za-z0-9_]*)\s*(?:\(\))?\s*(?:@[^\s]+\s*)*([\[\]A-Za-z]|map)", s)
        if m and buf:
            comment = " ".join(buf)
            vals = extract_enum_values(comment)
            if vals:
                out.append((m.group(1), i + 1, vals, comment))
        if not s.startswith("//"):
            buf = []
    return out


def looks_like_enum_token(v: str) -> bool:
    """Distinguish an API value from an English word caught by the same regex.

    The enumerating clause runs on into prose ("One of A, B, or C, which disables
    the Unity Catalog") and a naive split swallows "disables", "Unity", "Catalog"
    as values. Real values in these SDKs are SCREAMING_SNAKE, camelCase, or
    hyphen/underscore-joined; a bare English word, capitalized or not, is not one.
    """
    if "_" in v or "-" in v or "." in v:
        return True
    if v.isupper() and len(v) > 1:
        return True
    # camelCase / PascalCase with an internal capital: gitHub, bitbucketCloud.
    return any(c.isupper() for c in v[1:])


def extract_enum_values(comment: str):
    """Pull the enumerated values out of a doc comment, or return an empty set."""
    m = ENUM_PROSE_RE.search(comment)
    if not m:
        return set()
    body = m.group("body")
    vals = set()
    for cand in VALUE_RE.findall(body):
        if cand.lower() in STOPWORDS or not looks_like_enum_token(cand):
            continue
        vals.add(cand)
    return vals if len(vals) >= 2 else set()


def sdk_const_groups(sdk_dir: str):
    """-> {type_name: {value, ...}} for every Go string-const enum in the tree."""
    groups = {}
    for root, dirs, files in os.walk(sdk_dir):
        dirs[:] = [d for d in dirs if d not in ("vendor", "testdata", "examples")]
        for name in files:
            if not name.endswith(".go") or name.endswith("_test.go"):
                continue
            try:
                src = open(os.path.join(root, name), encoding="utf-8", errors="replace").read()
            except OSError:
                continue
            for _const_name, type_name, value in GO_CONST_RE.findall(src):
                groups.setdefault(type_name, set()).add(value)
    return {k: v for k, v in groups.items() if len(v) >= 2}


def best_match(values, groups):
    """Pick the SDK enum whose values overlap the comment's most.

    Scored on two axes. Coverage (how much of the comment the enum accounts for)
    decides, because a comment may deliberately list a subset. Tightness (how
    little of the enum is left over) breaks ties, which matters more than it
    sounds: unrelated enums share vocabulary -- USER, GROUP and SERVICE_PRINCIPAL
    appear in half a dozen types -- so several will cover a short comment
    perfectly and the tightest fit is the likeliest one.
    """
    best, best_score = None, (0.0, 0.0)
    lowered = {v.lower() for v in values}
    for type_name, sdk_vals in groups.items():
        hit = len(lowered & {v.lower() for v in sdk_vals})
        if not hit:
            continue
        score = (hit / len(lowered), hit / len(sdk_vals))
        if score > best_score:
            best, best_score = type_name, score
    return best, best_score[0]


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("lr_file")
    ap.add_argument("sdk_dir")
    ap.add_argument("--field", help="only report this field name")
    ap.add_argument(
        "--min-overlap",
        type=float,
        default=0.5,
        help="fraction of the comment's values an SDK enum must cover to be paired (default: 0.5)",
    )
    args = ap.parse_args()

    fields = lr_fields_with_enums(args.lr_file)
    if args.field:
        fields = [f for f in fields if f[0] == args.field]
    if not fields:
        print("no .lr fields with enumerated values found", file=sys.stderr)
        return

    groups = sdk_const_groups(args.sdk_dir)
    if not groups:
        print(f"no Go string-const enums found under {args.sdk_dir}", file=sys.stderr)
        return

    drifted, checked, unpaired = 0, 0, 0
    for field, line, values, comment in fields:
        match, score = best_match(values, groups)
        if not match or score < args.min_overlap:
            unpaired += 1
            continue
        checked += 1
        sdk_vals = groups[match]
        lowered = {v.lower() for v in values}
        missing = sorted(v for v in sdk_vals if v.lower() not in lowered)
        if not missing:
            continue
        drifted += 1
        print(f"\n{args.lr_file}:{line}  {field}")
        print(f"  comment lists : {', '.join(sorted(values))}")
        print(f"  SDK enum      : {match} ({int(score * 100)}% overlap)")
        print(f"  SDK also has  : {', '.join(missing)}")

    print(
        f"\n# {checked} enumerated field(s) paired with an SDK enum, {drifted} with values "
        f"the comment omits, {unpaired} unpaired (no confident match).",
        file=sys.stderr,
    )
    if drifted:
        print(
            "# Each hit is a candidate, not a verdict. Confirm the paired enum is the one the "
            "field actually carries, and that the omission is not deliberate, before editing.",
            file=sys.stderr,
        )


if __name__ == "__main__":
    main()
