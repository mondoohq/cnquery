# Development

## Build

### Prerequisites

Before building from source, be sure to install:

- [Go 1.21.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.21 protobuf`

## Install from source

1. Verify that you have Go 1.21+ installed:

   ```
   $ go version
   ```

If `go` is not installed or an older version exists, follow instructions
on [the Go website](https://golang.org/doc/install).

2. Clone this repository:

   ```sh
   $ git clone https://github.com/mondoohq/cnquery.git
   $ cd cnquery
   ```

3. Build and install on Unix-like systems

   ```sh
   # Build all providers
   make providers

   # To install cnquery using Go into the $GOBIN directory:
   make cnquery/install
   ```

## Develop cnquery, providers, or resources

Whenever you change resources, providers, or protos, you must generate files for the compiler. To do this, make sure you
have the necessary tools installed (such as protobuf):

```bash
make prep
```

Then, whenever you make changes, just run:

```bash
make cnquery/generate
```

This generates and updates all required files for the build. At this point you can `make cnquery/install` again as
outlined above.

## Debug providers

In v9 we introduced providers, which split up the providers into individual go modules. This make it more development
more lightweight and speedy.

To debug a provider locally with cnquery:

1. Modify the `providers.yaml` in the root folder and add providers you want to test to the `builtin` field. Example:
   ```yaml
   builtin: [aws]
   ```
2. Build and update everything via:
   ```bash
   make providers/config
   ```
3. You can now use and debug your code. For example `make cnquery/install` or start a debugger.
4. Once done, please remember to restore `providers.yaml` (or just set back: `builtin: []`) and
   re-run `make providers/config`.

## Using go workspaces

In case you want to develop cnquery, cnspec and providers at the same time, you can use go workspaces. This allows you
to use the latest updates from each other without having to commit and push changes.

Here is a sample config for `go.work` in the root folder of `cnquery` and `cnspec`:

```
go 1.21

use (
   ./cnquery
   ./cnquery/providers/arista
   ./cnquery/providers/aws
   ./cnquery/providers/azure
   ./cnquery/providers/equinix
   ./cnquery/providers/gcp
   ./cnquery/providers/github
   ./cnquery/providers/gitlab
   ./cnquery/providers/google-workspace
   ./cnquery/providers/ipmi
   ./cnquery/providers/k8s
   ./cnquery/providers/ms365
   ./cnquery/providers/oci
   ./cnquery/providers/okta
   ./cnquery/providers/opcua
   ./cnquery/providers/slack
   ./cnquery/providers/terraform
   ./cnquery/providers/vcd
   ./cnquery/providers/vsphere
   ./cnspec
)
```

## Contribute changes

### Mark PRs with emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals ⚡ speed 📄 docs  
✨⭐🌟🌠 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🟢 fix tests 🎫 auth 🦅 falcon 🐳 container
