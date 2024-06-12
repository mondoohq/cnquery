# Development

## Build

### Prerequisites

Before building from source, be sure to install:

- [go 1.22.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.22 protobuf`

## Install from source

1. Verify that you have Go 1.22+ installed:

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

3. Build and install on Unix-like systems:

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

`cnquery` uses a plugin mechanism. Each provider has its own go modules. This ensures that dependencies are only used on
the appropriate provider. Since providers are their own binaries, debugging is more complex. To ease debugging, we wrote
a small tool that configures the provider accordingly so that it is compiled into the main binary.

To debug a provider locally with cnquery:

1. Modify the `providers.yaml` in the root folder and add providers you want to test to the `builtin` field. Example:
   ```yaml
   builtin: [aws]
   ```
2. Build and update everything:
   ```bash
   make providers/config
   ```
3. You can now use and debug your code. For example `make cnquery/install` or start a debugger.
4. Once done, please remember to restore `providers.yaml` (or just set back: `builtin: []`) and
   re-run `make providers/config`.

In your favorite IDE use `apps/cnquery/cnquery.go` as main entry point and set the following program
arguments `run aws -c "aws.ec2.instances"` to run the AWS provider with the `aws.ec2.instances` MQL query.

### Remote debug providers

Some providers need to run on specific VMs, e.g., GCP Snapshot scanning.
The `launch.json` already includes a debug target for remote debugging.
You only need to adjust the values to your setup.
Additionally, you need to set up the debugger on the remote VM:

1. Install Go.
2. Install Delve.
3. Change the local config to include the provider you want to debug as builtin (as described above).
4. Copy the source to the remote VM. (`rsync` makes multiple debug session easier.)
5. Allow ingress traffic to the debugger in the firewall.
6. Run the debugger on the remove VM:

  ```
  dlv debug <path>/apps/cnquery/cnquery.go --headless --listen=:12345 -- run gcp snapshot --project-id xyz-123 suse15 -c "asset{ name ids }" --verbose
  ```

To learn more, including other possible ways to remote debug, read:
https://github.com/golang/vscode-go/blob/master/docs/debugging.md

## Update provider versions

Each provider has its own version, which is based on [Semver](https://semver.org/).

It's often easy to forget to update them. We didn't want to auto-update versions and accidentally release them for now,
so you must update versions in order to get the new providers out.

Here's how to make this process as easy as 🥧 :

**Set up the version utility**

In the cnquery repo you can now find the version utility in `providers-sdk/v1/util/version`.

To make working with this utility easier, let's alias it:

```bash
alias version="go run providers-sdk/v1/util/version/version.go"
```

**Check provider versions**

The version utility can check if providers need upgrades. If you use it in `--fast` mode, it doesn't crawl the entire
Git change history but only looks for the first change.

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

The utility automatically detects if providers have no changes since their last version bump. It also counts changes to
all providers that have changed.

If you prefer not to wait, you can use the `--fast` option, which only looks for the first change.

**Update provider versions**

Once you are ready to release providers, you can use the `update` command.

Here is an example showing how the version tool increments and updates all provider versions:

```bash
version update providers/*/
```

Notable options include:

- `--increment` auto-increments either the patch or minor version for you (e.g., `--increment=patch`). Without this
  option you get the interactive CLI.
- `--fast` performs fast change detection (i.e., once a change is found it will create the update).
- `--commit` automatically generates the commit for you and pushes the branch to GitHub.

If you use the `--commit` option, the version utility creates the commit and pushes it back to `origin`:

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

## Use Go workspaces

If you want to develop cnquery, cnspec, and providers at the same time, you can use Go workspaces. This allows you
to use the latest updates from the different repos without having to commit and push changes.

Here is a sample config for `go.work` in the root folder of `cnquery` and `cnspec`:

```
go 1.22

use (
   ./cnquery
   ./cnquery/providers/ansible
   ./cnquery/providers/arista
   ./cnquery/providers/atlassian
   ./cnquery/providers/aws
   ./cnquery/providers/azure
   ./cnquery/providers/cloudformation
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
   ./cnquery/providers/shodan
   ./cnquery/providers/snowflake
   ./cnquery/providers/slack
   ./cnquery/providers/terraform
   ./cnquery/providers/vcd
   ./cnquery/providers/vsphere
   ./cnspec
)
```

## Providers development best practices

The more time we spend building providers, the more we learn how to do better in the future. Here we describe learnings
that will help you get started with providers development.

### Referencing MQL resources

Often we have a top-level MQL resource, which we want to reference in another top-level resource.

For example, GCP networks can be retrieved for a project. That is a top-level resource:

```
// GCP Compute Engine
private gcp.project.computeService {
   // Google Compute Engine VPC network in a project
   networks() []gcp.project.computeService.network
}
```

However, we have a reference to a GCP network in a GCP Compute address. This allows us to quickly navigate to the
network in which an address is created:

```
private gcp.project.computeService.address {
  // Static IP address
  address string

  // Network in which to reserve the address
  network() gcp.project.computeService.network
}
```

The simple way to implement the reference would be to call the GCP API every
time `gcp.project.computeService.address.network` is executed. However, this would generate an excessive amount of API
calls when scanning large GCP projects. If we have 10 addresses, this would mean 10 separate API calls to get the
network, one for each of them.

MQL has powerful caching capabilities that let us achieve the same end result with a single (or fewer) API calls.

First, create an init function for `gcp.project.computeService.network`, which is the resource we are cross-referencing:

```go
func initGcpProjectComputeServiceNetwork(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	// Here we check that the resource isn't fully initialized yet
   if len(args) > 2 {
		return args, nil, nil
	}

	// If no args are set, try reading them from the platform ID
	if len(args) == 0 {
		if ids := getAssetIdentifier(runtime); ids != nil {
			args["name"] = llx.StringData(ids.name)
			args["projectId"] = llx.StringData(ids.project)
		} else {
			return nil, nil, errors.New("no asset identifier found")
		}
	}

   // We create a gcp.project.computeService resource which would allow us to retrieve networks via MQL
	obj, err := CreateResource(runtime, "gcp.project.computeService", map[string]*llx.RawData{
		"projectId": args["projectId"],
	})
	if err != nil {
		return nil, nil, err
	}

   // Cast the resource to the appropriate type
	computeSvc := obj.(*mqlGcpProjectComputeService)
   // List the networks: equivalent to gcp.project.computeService.networks MQL query. This retrieves all networks in the project and caches them in the MQL cache. Consecutive calls to this retrieve the data from the cache and do not execute any API calls.
	networks := computeSvc.GetNetworks()
	if networks.Error != nil {
		return nil, nil, networks.Error
	}

   // Filter the networks in memory by comparing them with the input arguments
	for _, n := range networks.Data {
		network := n.(*mqlGcpProjectComputeServiceNetwork)
		name := network.GetName()
		if name.Error != nil {
			return nil, nil, name.Error
		}
		projectId := network.GetProjectId()
		if projectId.Error != nil {
			return nil, nil, projectId.Error
		}

      // return the resource if found
		if name.Data == args["name"].Value && projectId.Data == args["projectId"].Value {
			return args, network, nil
		}
	}
	return nil, nil, fmt.Errorf("not found")
}
```

Then, we implement the function for retrieving the network for a GCP compute address:

```go
func (g *mqlGcpProjectComputeServiceAddress) network() (*mqlGcpProjectComputeServiceNetwork, error) {
	if g.NetworkUrl.Error != nil {
		return nil, g.NetworkUrl.Error
	}
	networkUrl := g.NetworkUrl.Data

	// Format is https://www.googleapis.com/compute/v1/projects/project1/global/networks/net-1
	params := strings.TrimPrefix(networkUrl, "https://www.googleapis.com/compute/v1/")
	parts := strings.Split(params, "/")
	resId := resourceId{Project: parts[1], Region: parts[2], Name: parts[4]}

   // Use the init function for the resource to find the one that we need
	res, err := CreateResource(g.MqlRuntime, "gcp.project.computeService.network", map[string]*llx.RawData{
		"name":      llx.StringData(resId.Name),
		"projectId": llx.StringData(resId.Project),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectComputeServiceNetwork), nil
}

```

## Contribute changes

### Mark PRs with emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals ⚡ speed 📄 docs
✨⭐🌟🌠 smaller or larger features 🐎 race condition
🌙 MQL 🌈 visual 🟢 fix tests 🎫 auth 🦅 falcon 🐳 container
