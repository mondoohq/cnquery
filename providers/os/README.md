# OS Provider

This provider makes the operating system accessible, including Linux, macOS, Windows, and many more.

It provides connectors for Local OS, SSH, WinRM, and Docker.

## Usage

```shell
mql shell os
```

## Examples

<!-- TODO: add example queries -->

## SSH connection timeout

SSH banner probing, TCP connection establishment, the handshake and authentication
share a 30-second timeout. Set the `ssh_connect_timeout` inventory connection option
or `MONDOO_SSH_CONNECT_TIMEOUT` environment variable to a positive Go duration such
as `60s` to allow slower connections. The inventory option takes precedence. The
banner probe also has a five-second limit. The startup deadline is cleared after
authentication so it does not expire a healthy scan connection.
