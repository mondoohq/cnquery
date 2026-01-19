---
title: Get Started with cnquery
id: cnquery-get-started
sidebar_label: Get Started with cnquery
displayed_sidebar: cnquery
sidebar_position: 2
description: cnquery is Mondoo's open source, cloud-native tool that answers every question about your infrastructure. Install, and get up and running with cnquery.
image: /img/cnquery/mondoo-feature.jpg
---

Welcome to cnquery, an open source project created by [Mondoo](https://mondoo.com)!

-> [Learn about cnquery](/cnquery/cnquery-about)

## Download and install cnquery​

Install cnquery with our installation script:

### Linux and macOS

```bash
bash -c "$(curl -sSL https://install.mondoo.com/sh)"
```

(You can read the [Linux/macOS installation script](https://install.mondoo.com/sh).)

### Windows

```powershell
Set-ExecutionPolicy Unrestricted -Scope Process -Force;
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072;
iex ((New-Object System.Net.WebClient).DownloadString('https://install.mondoo.com/ps1/cnquery'));
Install-Mondoo -Product cnquery;
```

(You can read the [Windows installation script](https://install.mondoo.com/ps1/cnquery).)

### Install manually

Manual installation packages are available on [GitHub releases](https://github.com/mondoohq/cnquery/releases/latest).

<Callout>
If you install cnquery on machines that can't download and install updates (because they're air-gapped or don't give cnquery write access), you must install cnquery providers. To learn more, read [Manage cnquery Providers](/cnquery/providers/).
</Callout>
## Run queries in the cnquery shell​

The easiest way to discover cnquery's capabilities is to use the interactive shell, which has auto-complete to guide you:

```bash
cnquery shell
```

Once inside the shell, you can enter MQL queries. For example, this query returns the name of the current machine and the platform it's running:

```bash
asset { name title }
```

### Get help in the cnquery shell​

To see what information cnquery can retrieve, use the `help` command. These are some examples of how the help can guide you:

| This command...        | Describes the queryable resources for... |
| ---------------------- | ---------------------------------------- |
| `help`                 | All of cnquery                           |
| `help k8s`             | Kubernetes                               |
| `help k8s.statefulset` | Kubernetes Cluster StatefulSets          |
| `help azure`           | Azure                                    |
| `help terraform`       | Terraform                                |

### Exit the cnquery shell​

To exit cnquery shell, either press `Ctrl + D` or type `exit`.

## Run queries in your own shell​

To run standalone queries in your shell, use the cnquery run command:

```bash
cnquery run TARGET -c "QUERY"
```

| For...   | Substitute...                                                           |
| -------- | ----------------------------------------------------------------------- |
| `TARGET` | The asset to query, such as `local` or a transport to a remote machine. |
| `QUERY`  | The MQL query that specifies the information you want.                  |

For example, this command runs a query against your local system. It lists the services installed and whether each service is running:

```bash
cnquery run local -c "services.list { name running }"
```

For a list of supported targets, use the help command:

```bash
cnquery help run
```

## Explore your infrastructure in Mondoo Platform​

To more easily explore your infrastructure, sign up for a Mondoo Platform account. Mondoo's web-based console allows you to navigate, search, and inspect all of your assets.

To get started, [contact Mondoo](https://mondoo.com/contact).

To learn about Mondoo Platform, read the [Mondoo Platform docs](../platform/home) or visit [mondoo.com](https://mondoo.com).

## Learn more​

- To explore cnquery commands, read [CLI Reference](/cnquery/cli/cnquery).
- To explore the capabilities of the MQL language, read the [MQL docs](/mql/resources).
- To learn what technologies cnquery integrates with, read [Supported Query Targets](/cnquery/cnquery-supported).

---
