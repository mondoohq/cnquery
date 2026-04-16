# ADR: cnspec SBOM Cataloger for PHP / Composer

**Status:** Proposed
**Date:** 2026-04-17

---

## Context

cnspec needs native SBOM cataloging for PHP/Composer packages to detect dependencies in both local and remote scanning contexts (SSH, WinRM, containers). PHP powers a large portion of the web (WordPress, Laravel, Symfony, Drupal), making it important for SBOM coverage.

PHP uses Composer as its package manager. Dependencies are declared in `composer.json`, resolved versions are recorded in `composer.lock`, and installed package metadata is stored in `vendor/composer/installed.json`. All three files use JSON format.

---

## File Formats

### composer.lock (Lock File)

Contains resolved dependency versions with integrity hashes.

```json
{
  "packages": [
    {
      "name": "monolog/monolog",
      "version": "3.5.0",
      "source": { "type": "git", "url": "..." },
      "dist": { "type": "zip", "url": "...", "shasum": "..." },
      "require": { "php": ">=8.1", "psr/log": "^2.0 || ^3.0" },
      "license": ["MIT"],
      "description": "Sends your logs to files, sockets, ..."
    }
  ],
  "packages-dev": [
    {
      "name": "phpunit/phpunit",
      "version": "10.5.5",
      "license": ["BSD-3-Clause"]
    }
  ]
}
```

**Key fields:**
- `packages`: Production dependencies with resolved versions
- `packages-dev`: Development dependencies
- Each entry: `name` (vendor/package), `version`, `license`, `description`

### composer.json (Manifest)

Declares project metadata and dependency requirements.

```json
{
  "name": "myvendor/myproject",
  "version": "1.0.0",
  "require": {
    "php": ">=8.1",
    "monolog/monolog": "^3.0"
  },
  "require-dev": {
    "phpunit/phpunit": "^10.0"
  }
}
```

### vendor/composer/installed.json (Installed Packages)

Metadata about packages installed on disk. Present in image scans.

```json
{
  "packages": [
    {
      "name": "monolog/monolog",
      "version": "3.5.0",
      "version_normalized": "3.5.0.0",
      "license": ["MIT"],
      "description": "Sends your logs to files, sockets, ..."
    }
  ]
}
```

---

## Parsing Strategy

All three files are standard JSON parsed with `encoding/json`.

### composer.lock Parser (Preferred)
1. Parse `packages` array for production dependencies
2. Parse `packages-dev` array for development dependencies
3. Extract name, version, license, description from each entry

### composer.json Parser
1. Parse `name` and `version` for root project
2. Parse `require` map for production dependencies (skip `php` and `ext-*` entries)
3. Parse `require-dev` map for development dependencies

### installed.json Parser
1. Parse `packages` array for installed package metadata
2. No dev/prod distinction available

---

## Package Identification

### PURL
Format: `pkg:composer/<vendor>/<name>@<version>`

Examples:
- `pkg:composer/monolog/monolog@3.5.0`
- `pkg:composer/symfony/console@6.4.1`

### CPE Generation
Vendor is extracted from the Composer vendor prefix (e.g., `monolog` from `monolog/monolog`).

---

## Dev/Prod Classification

- **composer.lock**: `packages` = production, `packages-dev` = dev
- **composer.json**: `require` = production, `require-dev` = dev
- **installed.json**: No distinction

---

## Remote Scanning Considerations

All files are JSON, typically small (< 500KB). No special handling needed.

Default search paths: `/app`, `/var/www/html`, `/usr/src/app`, `/home/*/app`

---

## Verification Plan

- Fixture-based tests for all 3 parsers
- Interactive: `mql run os -c "php.packages(path: '.') { list { name version purl } }"`
- Container: Scan a PHP container image
