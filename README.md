# cnquery

Welcome to cnquery, the cloud-native asset inventory and query system for your entire fleet!

Here are a few examples of what it can do:

```
# run a query and print the output
cnquery exec -q "packages.installed { name version }"

# run a query pack on a docker image and print results as json
cnquery exec docker 14119a -f pack.mql.yaml -j

# open an interactive shell to an aws account
cnquery shell aws
> ec2.instances{*}
```


## Quick Start

Please ensure you have the latest [Go 1.19.0+](https://golang.org/dl/) and latest [Protocol Buffers](https://github.com/protocolbuffers/protobuf/releases).  

Building:

```bash
# install all dependent tools
make prep 

# build and install cnquery
make build
make install
```

Some files in this repo are auto-generated. Whenever a proto or resource pack is changed, these will need to be rebuilt. Please re-run:

```bash
make cnquery/generate
```

## Development

We love emojis in our commits. These are their meanings:

🛑 breaking 🐛 bugfix 🧹 cleanup/internals 📄 docs
✨⭐🌟🎉 smaller or larger features 🐎 race condition
🌙 MQL 🌈 visual 🍏 fix tests 🎫 auth 🦅 falcon 🐳 container


## Legal

- **Copyright:** 2018-2022, Mondoo Inc, proprietary
- **Authors:** Christoph Hartmann, Dominik Richter

