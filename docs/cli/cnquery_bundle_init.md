---
id: cnquery_bundle_init
title: cnquery bundle init
---

Create an example query pack

### Synopsis

Create an example query pack that you can use as a starting point. If you don't provide a filename, cnquery uses `example-pack.mql.yaml`

```
cnquery bundle init [path] [flags]
```

### Options

```
  -h, --help   help for init
```

### Options inherited from parent commands

```
      --api-proxy string   Set proxy for communications with Mondoo Platform API
      --auto-update        Enable automatic provider installation and update (default true)
      --config string      Set config file path (default $HOME/.config/mondoo/mondoo.yml)
      --log-level string   Set log level: error, warn, info, debug, trace (default "info")
  -v, --verbose            Enable verbose output
```

### SEE ALSO

- [cnquery bundle](cnquery_bundle) - Create, upload, and validate query packs
