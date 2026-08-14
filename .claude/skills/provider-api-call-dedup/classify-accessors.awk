# Copyright Mondoo, Inc. 2024, 2026
# SPDX-License-Identifier: BUSL-1.1
#
# Companion to classify-inits.awk, covering the half it cannot see.
#
#   awk -f classify-accessors.awk providers/<name>/resources/*.go | sort
#
# classify-inits.awk reads init functions. But a typed reference can also be
# resolved by the accessor itself, which builds a client and fetches inline and
# so never enters an init at all. Those are invisible to that script and cannot
# be fixed by anything done at the init level.
#
# This one reads the accessors instead: functions shaped
#
#   func (a *mqlFoo) bar() (*mqlBaz, error)
#
# i.e. returning a single resource, which is what a typed reference looks like.
#
# Output: <bucket>\t<file>:<line>\t<signature>
#
#   INLINE   builds a client and fetches; never calls NewResource. Not reachable
#            from init-level work. Triage by fan-in before doing anything (see
#            below).
#   VIA-NEW  calls NewResource, so it routes through the target's init and the
#            init-level fixes apply.
#   NO-IO    reads what it already has. Fine.
#
# TRIAGE THE INLINE ONES BY FAN-IN. They split into two kinds and only one is
# worth touching:
#
#   Owned sub-object -- a config child belonging to exactly one parent (a
#     database's audit policy, a storage account's management policy). One fetch
#     per parent is the floor and GetOrCompute already memoizes it there, so
#     nothing repeats. Leave it alone.
#
#   Shared resource -- a subnet, a public IP, a firewall policy, a disk.
#     Several resources point at it, so it is fetched once per referrer with
#     nothing deduplicating them. This is the backlog.
#
# Tell them apart by counting references to the target type:
#
#   rg -c '"<target.mql.name>"' providers/<name>/resources/*.go | grep -v _test
#
# In azure the split is roughly 30 owned sub-objects to 12 shared targets, so
# the real backlog is small and specific. Do not report the raw INLINE count as
# though all of it were work.

/^func \(a \*mql[A-Za-z0-9]+\) [a-z][A-Za-z0-9]*\(\) \(\*mql[A-Za-z0-9]+, error\) \{/ {
  inFunc = 1
  sig = $0
  start = FNR
  body = ""
  next
}

inFunc && /^}/ {
  client = (body ~ /New[A-Za-z]*Client\(/)
  newres = (body ~ /NewResource\(/)

  if (newres)      bucket = "VIA-NEW"
  else if (client) bucket = "INLINE"
  else             bucket = "NO-IO"

  printf "%s\t%s:%d\t%s\n", bucket, FILENAME, start, sig
  inFunc = 0
  next
}

inFunc { body = body "\n" $0 }
