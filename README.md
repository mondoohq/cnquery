# cnquery

Cloud-Native Asset Inventory Framework

`cnquery` is a cloud-native tool for querying your entire fleet. It answers thousands of questions about your infrastructure, and integrates with over 300 resources across cloud accounts, Kubernetes, containers, services, VMs, APIs, and more.

```bash
# run a query and print the output
cnquery run local -c "packages.installed { name version }"

# execute a query pack on a Docker image and print results as json
cnquery explore docker 14119a -f pack.mql.yaml -j

# open an interactive shell to an aws account
cnquery shell aws
> aws.ec2.instances{*}
```

## Installation

Install cnquery with our installation script:

```bash
bash -c "$(curl -sSL https://install.mondoo.com/sh/cnquery)"
```

If you prefer a package, find it on [GitHub releases](https://github.com/mondoohq/cnquery/releases).

## Interactive shell

The easiest way to explore `cnquery` is to use our interactive shell, which has auto-complete to guide you:

```bash
cnquery shell local
```

Once inside the shell, you can enter MQL queries like this:

```coffeescript
> asset { name title }
```

To learn more, use the `help` command. 

To exit, either press CTRL + D or type `exit`.

## Run simple queries

To run standalone queries in your shell, use the `run` command:

```bash
cnquery run <TARGET> -c <QUERY>
```

For example, this runs a query against your local system:

```bash
cnquery run local -c "services.list { name running }"
```

For a list of supported targets, use the `help` command:

```bash
cnquery help run
```

For automation, it is often helpful to convert the output to JSON. Use `-j` or `--json`:

```bash
cnquery run local -c "services.list{*}" -j
```

You can then pipe the output to [jq](https://stedolan.github.io/jq/) or other applications.

## Query packs

You can combine multiple queries into query packs, which can run together. `cnquery` comes with query packs out of the box for most systems. You can simply run:

```bash
cnquery explore local
```

Without specifying anything else, `cnquery` tries to find and run the default query pack for the given system.

You can specify a query pack that you want to run. Use the `--pack` argument:

```bash
cnquery explore local --pack incident-response
```

You can also choose just one query from a query pack. Specify the query ID with the query pack:

```bash
cnquery explore local --pack incident-response --query-id sth-01
```

Custom query packs let you bundle queries to meet your specific needs. You can find a simple query pack example in `examples/simple.mql.yaml`. To run it:

```bash
cnquery explore local -f examples/simple.mql.yaml
```

These files can also contain multiple query packs for many different target systems. For an example, see `examples/multi-target.mql.yaml`.

## Distributing cnqueries across your fleet

You can share query packs across your fleet using the Query Hub.

The Query Hub creates a secure, private environment in your account that stores data about your assets. It makes it very easy for all assets to report on query packs and define custom rules for your fleet.

To use the Query Hub:

```bash
cnquery auth login
```

Once set up, you can collect your asset's data:

```bash
cnquery explore local
```

To add custom query packs, you can upload them:

```bash
cnquery pack upload mypack.mql.yaml
```

## What’s next?

There are so many things `cnquery` can do! Gather information about your fleet, find tool-sprawl across systems, run incident response, and share data with auditors… `cnquery` is nearly limitless in capabilities.

Explore:

- The Query Hub
- [MQL introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL resource packs](https://mondoo.com/docs/references/mql/)
- [cnspec](https://github.com/mondoohq/cnspec), our open source, cloud-native security scanner

Our goal is to become the API for your entire infrastructure. Join our [community](https://github.com/orgs/mondoohq/discussions) today and let’s grow it together!

## Development

See our [Development Documentation](docs/development.md) for information on building and contributing to cnquery.

## Legal

- **Copyright:** 2018-2022, Mondoo Inc, proprietary
- **Authors:** Christoph Hartmann, Dominik Richter
