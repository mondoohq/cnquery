---
id: cnquery_shell
title: cnquery shell
---

Interactive query shell for MQL

### Synopsis

Allows the interactive exploration of MQL queries

```
cnquery shell [flags]
```

### Options

```
  -c, --command string         MQL query to execute in the shell
  -h, --help                   help for shell
      --platform-id string     Select a specific target asset by providing its platform ID
      --sudo                   Elevate privileges with sudo
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

- [cnquery](cnquery) - cnquery CLI
