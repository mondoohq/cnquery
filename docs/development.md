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

If `go` is not installed or an older version exists, follow instructions on [the Go website](https://golang.org/doc/install).

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

Whenever you change resources, providers, or protos, you must generate files for the compiler. To do this, make sure you have the necessary tools installed (such as protobuf):

```bash
make prep
```

Then, whenever you make changes, just run:

```bash
make cnquery/generate
```

This generates and updates all required files for the build. At this point you can `make cnquery/install` again as outlined above.

## Debug providers

In v9 we introduced providers, which split up the providers into individual go modules. This make it more development more lightweight and speedy.

To debug a provider locally with cnquery:

1. Copy the resources json file `providers/TYPE/resources/TYPE.resources.json` to the `providers` top-level dir
2. Add the provider to `providers/builtin.go`

   ```
   awsconf "go.mondoo.com/cnquery/providers/aws/config"
   awsp "go.mondoo.com/cnquery/providers/aws/provider"

   //go:embed aws.resources.json
   var awsInfo []byte

   awsconf.Config.ID: {
       Runtime: &RunningProvider{
       Name:     awsconf.Config.Name,
       ID:       awsconf.Config.ID,
       Plugin:   awsp.Init(),
       Schema:   MustLoadSchema("aws", awsInfo),
       isClosed: false,
       },
       Config: &awsconf.Config,
   },
   ```

3. Change the local provider location in `go.mod`

   `replace go.mondoo.com/cnquery/providers/aws => ./providers/aws`

4. Build the provider after making changes: `make providers/build/aws`
5. Run cnquery like you normally do (`go run apps/cnquery/cnquery.go shell aws`). You should see it note that it is "using builtin provider for X"

## Contribute changes

### Mark PRs with emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals ⚡ speed 📄 docs  
✨⭐🌟🌠 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🟢 fix tests 🎫 auth 🦅 falcon 🐳 container
