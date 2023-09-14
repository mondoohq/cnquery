# Development

## Build

### Prerequisites

Before building from source, be sure to install:

- [Go 1.20.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.19 protobuf`

## Install from source

1. Verify that you have Go 1.19+ installed:

    ```
    $ go version
    ```

If `go` is not installed or an older version exists, follow instructions on [the Go website](https://golang.org/doc/install).

2. Clone this repository:

   ```sh
   $ git clone https://github.com/mondoohq/cnquery.git
   $ cd cnquery
   ```

3. Build and install:

    #### Unix-like systems
    ```sh
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

## Develop with beta v9
In v9 we introduced providers, which changes the development process a bit (and make it more lightweight and speedy!)

To test a provider locally:
- copy the resources json file `providers/TYPE/resources/TYPE.resources.json` to the `providers` top-level dir
- add the provider to `providers/builtin.go`
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

- note the local provider location in `go.mod`
    `replace go.mondoo.com/cnquery/providers/aws => ./providers/aws`

- build the provider after making changes: `make providers/build/aws`
- run cnquery like you normally do (`go run apps/cnquery/cnquery.go shell aws`)
- you should see it note that it is "using builtin provider for X"

## Contribute changes

### Mark PRs with emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals ⚡ speed 📄 docs  
✨⭐🌟🌠 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🟢 fix tests 🎫 auth 🦅 falcon 🐳 container  

