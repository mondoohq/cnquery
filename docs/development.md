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

In v9 we introduced provider plugins, which split up the providers into individual go modules. This makes development 
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

### Remote debug providers

Some providers need to run on specific VMs, e.g., GCP Snapshot scanning.
The `launch.json` already includes a debug target for remote debugging.
You only need to adjust the values to your setup.
Additionally, you need to set up the debugger on the remote VM:

- Install go
- Install delve
- Locally change the config to include the provider you want to debug as builtin as described above.
- Copy the source to the remote VM (`rsync` makes multiple debug session easier)
- Allow ingress traffic to the debugger in the firewall.
- Run the debugger on the remove VM:
  ```
  dlv debug <path>/apps/cnquery/cnquery.go --headless --listen=:12345 -- run gcp snapshot --project-id xyz-123 suse15 -c "asset{ name ids }" --verbose
  ```

Further information and possible other ways to remote debug: https://github.com/golang/vscode-go/blob/master/docs/debugging.md

## Update provider versions

Providers each have their own version, which is based on [Semver](https://semver.org/).

It's often easy to forget to update them. We didn't want to auto-update versions and accidentally release them for now, so you'll have to update versions in order to get the new providers out.

Here's how to make this process as easy as 🥧

**Setup**

In the cnquery repo you can now find the version utility in `providers-sdk/v1/util/version`.

To make working with it easier, let's alias it:

```bash
alias version="go run providers-sdk/v1/util/version/version.go"
```

**Version checking**

This utility can check if providers need upgrades. If you use it in `--fast` mode, it won't crawl the entire git change history but only looks for the first change.

```bash
version check providers/*/
```

```
...
crawling git history....
→ no changes provider=opcua version=9.0.1
crawling git history......
→ provider changed changes=2 provider=os version=9.0.1
...
```

It will automatically detect if providers have no changes since their last version bump and count changes that may have happened for those providers that have changed.

If you prefer not to wait, you can use the `--fast` option which will only look for the first change.

**Version update**

Once you are ready to release providers, you can use the `update` command.

Here is an example showing how the version tool will increment and update all provider versions:

```bash
version update providers/*/
```

Notable options include:
- `--increment` will auto-increment either the patch or minor version for you (eg: `--increment=patch`). Without this option you get the interactive CLI.
- `--fast` will do fast change detection (i.e. once a change is found it will create the update)
- `--commit` will automatically generate the commit for you and push the branch to github

If you use the `--commit` option, it will create both the commit and push it back to `origin`:

```bash
version update providers/*/ --increment=patch --commit
```

```
...
→ committed changes for os-9.0.2, slack-9.0.1, terraform-9.0.1, vcd-9.0.1, vsphere-9.0.1
→ running: git push -u origin version/os-9.0.2+slack-9.0.1+terraform-9.0.1+vcd-9.0.1+vsphere-9.0.1
→ updates pushed successfully, open:
	https://github.com/mondoohq/cnquery/compare/version/os-9.0.2+slack-9.0.1+terraform-9.0.1+vcd-9.0.1+vsphere-9.0.1?expand=1
```

The final line of this message is the blueprint for the pull request.

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
