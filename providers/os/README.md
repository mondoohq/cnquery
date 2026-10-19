# OS Provider

This provider makes the operating system accessible, including Linux, macOS, Windows, and many more.

It provides connectors for Local OS, SSH, WinRM, and Docker.

## Usage

```shell
mql shell os
```

## Examples

<!-- TODO: add example queries -->

## Native file searches

Native and archive searches default to a two-minute elapsed budget, at most
1,000,000 visited entries, and at most 100,000 matches. An exhausted budget is
an error. Narrow `from`, `depth`, or `type` to reduce the search scope.

A native search reports an error when its root cannot be read, or when a strict
mount-boundary search cannot read a directory's device metadata. These failures
cannot establish a complete inventory. A root that does not exist still has no
matches.

Cancellation and budget checks run between filesystem operations. A single
kernel-blocked `Stat` or `ReadDir` cannot be interrupted through `io/fs`; a
directory backend can also allocate its listing before the entry budget is
checked.

## SSH connection timeout

SSH banner probing, TCP connection establishment, the handshake and authentication
share a 30-second timeout. Set the `ssh_connect_timeout` inventory connection option
or `MONDOO_SSH_CONNECT_TIMEOUT` environment variable to a positive Go duration such
as `60s` to allow slower connections. The inventory option takes precedence. The
banner probe also has a five-second limit. The startup deadline is cleared after
authentication so it does not expire a healthy scan connection.
