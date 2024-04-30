# cnquery

![cnquery light-mode logo](docs/images/cnquery-light.svg#gh-light-mode-only)
![cnquery dark-mode logo](docs/images/cnquery-dark.svg#gh-dark-mode-only)

**Open source, cloud-native asset inventory and discovery**

cnquery is a cloud-native tool for querying your entire infrastructure. It answers thousands of questions about your infrastructure and integrates with over 300 resources across cloud accounts, Kubernetes, containers, services, VMs, APIs, and more.

![cnquery run example](docs/images/cnquery-run.gif)

Here are a few more examples:

```bash
# run a query and print the output
cnquery run -c "ports.listening { port process }"

# execute a query pack on a Docker image and print results as json
cnquery scan docker 14119a -f pack.mql.yaml -j

# open an interactive shell to an aws account
cnquery shell aws
> aws.ec2.instances{*}
```

[:books: To learn more, read the cnquery docs.](https://mondoo.com/docs/cnquery/home/)

## Installation

Install cnquery with our installation script:

**Linux and macOS**

```bash
bash -c "$(curl -sSL https://install.mondoo.com/sh)"
```

**Windows**

```powershell
Set-ExecutionPolicy Unrestricted -Scope Process -Force;
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
iex ((New-Object System.Net.WebClient).DownloadString('https://install.mondoo.com/ps1'));
Install-Mondoo;
```

If you prefer manual installation, you can find the cnquery packages in our [GitHub releases](https://github.com/mondoohq/cnquery/releases).

## Interactive shell

The easiest way to explore cnquery is to use our interactive shell, which has auto-complete to guide you:

```bash
cnquery shell
```

Once inside the shell, you can enter MQL queries like this:

```coffeescript
> asset { name title }
```

To learn more, use the `help` command.

To exit, either press CTRL + D or type `exit`.

You can run the shell against local and remote targets like `k8s`, `aws`, `docker`, and many more. Run `--help` to see a full list of supported providers.

## Run simple queries

To run standalone queries in your shell, use the `run` command:

```bash
cnquery run <TARGET> -c <QUERY>
```

For example, this runs a query against your local system:

```bash
cnquery run -c "services { name running }"
```

For automation, it is often helpful to convert the output to JSON. Use `-j` or `--json`:

```bash
cnquery run local -c "services { * }" -j
```

You can then pipe the output to [jq](https://jqlang.github.io/jq/) or other applications.

## Query packs

You can combine multiple queries into query packs, which can run together. cnquery comes with default [query packs](https://github.com/mondoohq/cnquery-packs) out of the box for most systems. You can run:

```bash
cnquery scan
```

Without specifying anything else, cnquery tries to find and run the default query pack for the given system.

You can specify a query pack that you want to run. Use the `--querypack` argument:

```bash
cnquery scan --querypack incident-response
```

Custom query packs let you bundle queries to meet your specific needs. You can find a simple query pack example in `examples/simple.mql.yaml`. To run it:

```bash
cnquery scan -f examples/example-os.mql.yaml
```

Like all other commands, you can specify different providers like `k8s`, `aws`, `docker`, and many more. Run `--help` to see the full list of supported providers.

![](docs/images/cnquery-scan.gif)

These files can also contain multiple query packs for many different target systems.

## Explore your infrastructure in Mondoo Platform​

To more easily explore your infrastructure, sign up for a free Mondoo Platform account. Mondoo's web-based console allows you to navigate, search, and arrange all of your assets.

Go to [console.mondoo.com](https://console.mondoo.com) to sign up.

To learn about Mondoo Platform, read the [Mondoo Platform docs](https://mondoo.com/docs/platform/home/) or visit [mondoo.com](https://mondoo.com).

## Distribute queries across your infrastructure with private query packs

You can create and share query packs using the Registry in the Mondoo Console. The Registry is a secure, private environment in your account where you store both Mondoo query packs and custom query packs. This lets you use the same query packs for all assets.

To use the Registry:

```bash
cnquery login --token TOKEN
```

Once set up, enable the query packs you want to use to collect your asset's data. For example, you can activate one or more AWS query packs in the Mondoo Console. Then run this command any time to collect the AWS information you need:

```bash
cnquery scan aws
```

To add custom query packs, you can upload them:

```bash
cnquery bundle upload mypack.mql.yaml
```

## Supported targets

| Target                            | Provider                   | Example                                                                                                                                                 |
| --------------------------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Arista EOS                        | `arista`                   | `cnquery shell arista`                                                                                                                                  |
| Atlassian Admin                   | `atlassian`                | `cnquery shell atlassian admin`                                                                                                                         |
| Atlassian Jira                    | `atlassian`                | `cnquery shell atlassian jira`                                                                                                                          |
| Atlassian Confluence              | `atlassian`                | `cnquery shell atlassian confluence`                                                                                                                    |
| Atlassian SCIM                    | `atlassian`                | `cnquery shell atlassian scim DIRECTORYID`                                                                                                              |
| AWS accounts                      | `aws`                      | `cnquery shell aws`                                                                                                                                     |
| AWS EC2 instances                 | `ssh`                      | `cnquery shell ssh user@host`                                                                                                                           |
| AWS EC2 Instance Connect          | `aws ec2 instance-connect` | `cnquery shell aws ec2 instance-connect ec2-user@INSTANCEID`                                                                                            |
| AWS EC2 EBS snapshot              | `aws ec2 ebs snapshot`     | `cnquery shell aws ec2 ebs snapshot SNAPSHOTID`                                                                                                         |
| AWS EC2 EBS volume                | `aws ec2 ebs volume`       | `cnquery shell aws ec2 ebs volume VOLUMEID`                                                                                                             |
| Container images                  | `container`, `docker`      | `cnquery shell container ubuntu:latest`                                                                                                                 |
| Container registries              | `container registry`       | `cnquery shell container registry index.docker.io/library/rockylinux:8 `                                                                                |
| DNS records                       | `host`                     | `cnquery shell host mondoo.com`                                                                                                                         |
| GitHub organizations              | `github org`               | `cnquery shell github org mondoohq`                                                                                                                     |
| GitHub repositories               | `github repo`              | `cnquery shell github repo mondoohq/cnquery`                                                                                                            |
| GitLab groups                     | `gitlab`                   | `cnquery shell gitlab --group mondoohq`                                                                                                                 |
| Google Cloud projects             | `gcp`                      | `cnquery shell gcp`                                                                                                                                     |
| Google Workspace                  | `google-workspace`         | `cnquery shell google-workspace --customer-id CUSTOMER_ID --impersonated-user-email EMAIL --credentials-path JSON_FILE`                                 |
| Kubernetes cluster nodes          | `local`, `ssh`             | `cnquery shell ssh user@host`                                                                                                                           |
| Kubernetes clusters               | `k8s`                      | `cnquery shell k8s`                                                                                                                                     |
| Kubernetes manifests              | `k8s`                      | `cnquery shell k8s manifest.yaml`                                                                                                                       |
| Kubernetes workloads              | `k8s`                      | `cnquery shell k8s --discover pods,deployments`                                                                                                         |
| Linux hosts                       | `local`, `ssh`             | `cnquery shell local` or<br></br>`cnquery shell ssh user@host`                                                                                          |
| macOS hosts                       | `local`, `ssh`             | `cnquery shell local` or<br></br>`cnquery shell ssh user@IP_ADDRESS`                                                                                    |
| Microsoft 365 tenants             | `ms365`                    | `cnquery shell ms365 --tenant-id TENANT_ID --client-id CLIENT_ID --certificate-path PFX_FILE`                                                           |
| Microsoft Azure subscriptions     | `azure`                    | `cnquery shell azure --subscription SUBSCRIPTION_ID`                                                                                                    |
| Microsoft Azure instances         | `ssh`                      | `cnquery shell ssh user@host`                                                                                                                           |
| Okta                              | `okta`                     | `cnquery shell okta --token TOKEN --organization ORGANIZATION`                                                                                          |
| OPC UA                            | `opcua`                    | `cnquery shell opcua`                                                                                                                                   |
| Oracle Cloud Infrastructure (OCI) | `oci`                      | `cnquery shell oci`                                                                                                                                     |
| Running containers                | `docker`                   | `cnquery shell docker CONTAINER_ID`                                                                                                                     |
| Slack                             | `slack`                    | `cnquery shell slack --token TOKEN`                                                                                                                     |
| SSL certificates on websites      | `host`                     | `cnquery shell host mondoo.com`                                                                                                                         |
| Terraform HCL                     | `terraform hcl`            | `cnquery shell terraform <directory> HCL_FILE_OR_PATH`                                                                                                  |
| Terraform plan                    | `terraform plan`           | `cnquery shell terraform plan <plan.json> json`                                                                                                         |
| Terraform state                   | `terraform state`          | `cnquery shell terraform state <state_file>.json`                                                                                                       |
| Vagrant virtual machines          | `vagrant`                  | `cnquery shell vagrant HOST`                                                                                                                            |
| VMware vSphere                    | `vsphere`                  | `cnquery shell vsphere user@domain@host --ask-pass`                                                                                                     |
| Windows hosts                     | `local`, `ssh`, `winrm`    | `cnquery shell local`<br></br>`cnquery shell ssh Administrator@IP_ADDRESS --ask-pass`<br></br>`cnquery shell winrm Administrator@IP_ADDRESS --ask-pass` |

## What's next?

There are so many things cnquery can do! Gather information about your infrastructure, find tool-sprawl across systems, run incident response, and share data with auditors… cnquery is nearly limitless in capabilities.

Explore:

- [cnquery docs](https://mondoo.com/docs/cnquery/home/)
- [Query packs](https://github.com/mondoohq/cnquery-packs)
- [MQL introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL language reference](https://mondoo.com/docs/mql/resources/)
- [cnspec](https://github.com/mondoohq/cnspec), our open source, cloud-native security scanner

## Join the community!

Our goal is to become the API for your entire infrastructure. Join our [community](https://github.com/orgs/mondoohq/discussions) today and let's grow it together!

## Development

See our [development documentation](docs/development.md) for information on building and contributing to cnquery.

## Legal

- **Copyright:** 2018-2024, Mondoo, Inc.
- **License:** BUSL 1.1
- **Authors:** Christoph Hartmann, Dominik Richter
