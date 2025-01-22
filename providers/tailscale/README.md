# Tailscale Provider

Use the tailscale provider to query devices, DNS namespaces, and more information about a Tailscale network
known as `tailnet`.

To authenticate using an API access token:

```
cnquery shell tailscale --token <access-token>
```

To authenticate using an OAuth client:

```
cnquery shell tailscale --client-id <id> --client-secret <secret>
```

You can also use the default environment variables `TAILSCALE_OAUTH_CLIENT_ID`, `TAILSCALE_OAUTH_CLIENT_SECRET`,
and `TAILSCALE_TAILNET` to provide your credentials.

If you are using an API access token instead of an OAuth client, use the `TAILSCALE_API_KEY` variable instead.

## Examples

**List all devices in a tailnet**

```shell
cnquery> tailscale.devices()
```

**Show a single device information**

```shell
cnquery> tailscale.device(id: "55161288425123456") {*}
tailscale.device: {
  id: "55161288425123456"
  isExternal: false
  os: "linux"
  created: 2021-06-25 12:34:34 -0700 PDT
  updateAvailable: true
  nodeKey: "nodekey:abc123"
  lastSeen: 2024-03-25 08:01:04 -0700 PDT
  user: "afiune@mondoo.com"
  hostname: "raspberrypi"
  clientVersion: "1.10.0-t766ae6c10-g3e6822772"
  authorized: true
  blocksIncomingConnections: false
  addresses: [
    0: "100.71.181.41"
    1: "abc1:abc1:a1e0:ab12:abc1:cd96:abc1:bf33"
  ]
  keyExpiryDisabled: true
  expires: 2022-08-02 18:55:39 -0700 PDT
  name: "raspberrypi.tail1a4a6.ts.net"
  machineKey: "mkey:abc123"
  tailnetLockKey: ""
  tailnetLockError: ""
}
```

# Advanced Usage

Discover all devices (any computer or mobile device) that joins the tailnet `example.com`.

```shell
cnquery shell tailscale example.com --discover devices
```
