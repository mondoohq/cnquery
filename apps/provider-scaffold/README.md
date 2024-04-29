# Provider Scaffold

This tool generates a new provider skeleton for a new provider.

## Pre-requisites

```shell
go install apps/provider-scaffold/provider-scaffold.go
```

## Usage

```shell
provider-scaffold --path providers/your-provider --provider-id your-provider --provider-name "Your Provider" --go-package go.mondoo.com/cnquery/v11/providers/your-provider
```

Now you have a full provider skeleton in `providers/your-provider` that you can start to implement.

Next run `go mod tidy` to update the go.mod file.
```shell
cd providers/your-provider
go mod tidy
```