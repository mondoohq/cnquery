#!/bin/sh
DEBIAN_FRONTEND=noninteractive apt-get update >/dev/null 2>&1
readlock() { cat /proc/locks | awk '{print $5}' | grep -v ^0 | xargs -I {1} find /proc/{1}/fd -maxdepth 1 -exec readlink {} \; | grep '^/var/lib/dpkg/lock$'; }
while test -n "$(readlock)"; do sleep 1; done
DEBIAN_FRONTEND=noninteractive apt-get upgrade --dry-run