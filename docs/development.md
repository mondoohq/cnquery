# Development

## Building

### Prerequisites

Before building from source, be sure to install:

- [Go 1.19.0+](https://golang.org/dl/)
- [Protocol Buffers v21+](https://github.com/protocolbuffers/protobuf/releases)

On macOS systems with Homebrew, run: `brew install go@1.19 protobuf`

## Installation from source

1. Verify that you have Go 1.19+ installed

    ```
    $ go version
    ```

If `go` is not installed or an older version exists, follow instructions on [the Go website](https://golang.org/doc/install).

2. Clone this repository

   ```sh
   $ git clone https://github.com/mondoohq/cnquery.git
   $ cd cnquery
   ```

3. Build and install

    #### Unix-like systems
    ```sh
    # To install `cnquery` using Go into the $GOBIN directory:
    make cnquery/install
    ```

## Developing cnquery, providers or resources

Whenever you change resources, providers, or protos, you must generate files for the compiler. To do this, make sure you have the necessary tools installed (such as protobuf):

```bash
make prep
```

Then, whenever you make changes, just run:

```bash
make cnquery/generate
```

This generates and updates all required files for the build. At this point you can `make cnquery/install` again as outlined above.

## Contributing Changes

### Marking PRs with Emojis

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals 📄 docs  
✨⭐🌟🎉 smaller or larger features 🐎 race condition  
🌙 MQL 🌈 visual 🍏 fix tests 🎫 auth 🦅 falcon 🐳 container  
