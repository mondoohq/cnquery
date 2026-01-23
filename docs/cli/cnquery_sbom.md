---
id: cnquery_sbom
title: cnquery sbom
---

Experimental: Generate a software bill of materials (SBOM) for a given asset

### Synopsis

Generate a software bill of materials (SBOM) for a given asset. The SBOM
is a representation of the asset's software components and their dependencies.

The following formats are supported:

- list (default)
- cnquery-json
- cyclonedx-json
- cyclonedx-xml
- spdx-json
- spdx-tag-value

Note this command is experimental and may change in the future.

```
cnquery sbom [flags]
```

### Options

```
      --annotation stringToString   Add an annotation to the asset (default [])
      --asset-name string           User-override for the asset name
  -h, --help                        help for sbom
  -o, --output string               Set output format: json, cyclonedx-json, cyclonedx-xml, spdx-json, spdx-tag-value, table (default "list")
      --output-target string        Set output target to which the SBOM report will be written
      --sudo                        Elevate privileges with sudo
      --with-cpes                   Generate CPEs for each component
      --with-evidence               Include evidence for each component
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
