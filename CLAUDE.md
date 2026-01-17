# Claude AI Context for cnquery

This directory contains information to help Claude AI assistants understand and work effectively with the cnquery codebase.

## Project Overview

**cnquery** is an open source, cloud-native asset inventory and discovery tool built by Mondoo. It's designed to query infrastructure across cloud accounts, Kubernetes, containers, services, VMs, APIs, and more.

### Key Technologies
- **Language**: Go (Golang)
- **License**: BUSL 1.1
- **Purpose**: Infrastructure querying and asset discovery
- **Query Language**: MQL (Mondoo Query Language)

## Architecture

### Core Components

1. **Providers** ([providers/](../providers/))
   - Provider implementations for different platforms (AWS, Azure, K8s, OS, etc.)
   - Each provider implements resource types and data collection
   - Located in `providers/os/resources/containers/manager.go` and similar paths

2. **MQL Query Engine**
   - Custom query language for infrastructure
   - Compiles and executes queries against various targets
   - Supports 850+ resources

3. **CLI Interface**
   - Main commands: `shell`, `run`, `scan`
   - Interactive REPL with auto-complete
   - Support for multiple output formats (JSON, YAML, etc.)

4. **Target Support**
   - Local systems, remote SSH/WinRM
   - Cloud platforms (AWS, Azure, GCP, OCI)
   - Containers and Kubernetes
   - SaaS platforms (GitHub, GitLab, Slack, Okta, etc.)

## Development Workflow

### Prerequisites

- **Go 1.25.0+** (install via `brew install go@1.25` on macOS)
- **Protocol Buffers v21+** (install via `brew install protobuf` on macOS)

### Building the Project

```bash
# Initial setup - build pre-req tools
make prep/tools

# Build all providers
make providers

# Install cnquery to $GOBIN
make cnquery/install

# When changing resources, providers, or protos
make prep                    # Install necessary tools
make cnquery/generate        # Generate and update required files
```

### Building Individual Providers

When you update a provider's `.lr` file:

```bash
# Generate go files for a specific provider (e.g., AWS)
make providers/mqlr
./mqlr generate providers/aws/resources/aws.lr \
  --docs-file providers/aws/resources/aws.lr.manifest.yaml \
  --dist providers/aws/resources

# Quick install changed provider
make providers/build/aws && make providers/install/aws
```

See [docs/development.md](../docs/development.md) for detailed build instructions.

### Project Structure

```
cnquery/
├── providers/          # Provider implementations (AWS, OS, K8s, etc.)
├── docs/              # Documentation
├── examples/          # Example query packs
├── cli/               # CLI command implementations
├── motor/             # Connection and transport layer
├── resources/         # MQL resource definitions
└── mqlc/              # MQL compiler
```

## Common Tasks

### Adding New Resources

When adding a new resource to a provider:
1. Define the resource schema in the `.lr` file
2. Implement data collection methods
3. Run `make cnquery/generate` to regenerate files
4. Add tests
5. Update documentation

### Working with Providers

Providers are located in `providers/` and follow a consistent structure:
- `resources/` - Resource definitions (`.lr` files) and implementations
- `config/` - Provider configuration
- `connection/` - Connection handling
- `README.md` - Provider-specific documentation (authentication, examples, prerequisites)

Each provider is a separate Go module with its own dependencies, compiled as a plugin.

**Important**: Many providers have their own README files with valuable information:
- Authentication methods and environment variables
- Prerequisites (e.g., ms365 requires PowerShell)
- Usage examples and common queries
- Troubleshooting guides

Always check `providers/<provider-name>/README.md` when working with a specific provider.

### Debugging Providers

Since providers use a plugin mechanism, debugging requires special configuration:

1. **Local debugging**: Add the provider to `builtin` in `providers.yaml`:
   ```yaml
   builtin: [aws]
   ```
2. Run `make providers/config` to rebuild with the provider built-in
3. Debug using `apps/cnquery/cnquery.go` as the main entry point
4. Restore `providers.yaml` when done: `builtin: []`

Example debug arguments: `run aws -c "aws.ec2.instances"`

**Remote debugging** (for VMs): See [docs/development.md](../docs/development.md) for Delve setup instructions.

### Testing

Run provider-specific tests:
```bash
cd providers/<provider-name>
go test ./...
```

### Updating Provider Versions

Providers use semantic versioning. Use the version utility:

```bash
# Alias for convenience
alias version="go run providers-sdk/v1/util/version/version.go"

# Check which providers need version updates
version check providers/*/

# Update versions interactively
version update providers/*/

# Auto-increment patch version and commit
version update providers/*/ --increment=patch --commit --fast
```

### Using Go Workspaces

For simultaneous development on cnquery, cnspec, and providers, create a `go.work` file:

```go
go 1.25

use (
   ./cnquery
   ./cnquery/providers/aws
   ./cnquery/providers/azure
   // ... add other providers as needed
   ./cnspec
)
```

This allows using latest updates across repos without committing/pushing.

## Key Concepts

### MQL (Mondoo Query Language)
- Declarative query language for infrastructure
- Examples:
  ```mql
  # Query running services
  services { name running }

  # Query AWS EC2 instances
  aws.ec2.instances { * }

  # Query with filters
  ports.listening.where(port == 22) { process }
  ```

### Query Packs
- Collections of related queries
- Can be bundled and shared
- Support multiple target systems
- Located in `examples/` directory

### Asset Discovery
- Automatic detection of infrastructure components
- Multi-cloud support
- Container and Kubernetes workload discovery

### Provider Development Best Practices

**Cross-referencing MQL Resources**: When referencing top-level resources from other resources, use init functions and caching to minimize API calls:

1. Create an `init` function for the referenced resource
2. Use `CreateResource()` to leverage MQL caching
3. Filter results in memory rather than making multiple API calls

Example: Instead of calling the GCP API 10 times for 10 addresses to get their networks, call it once to get all networks, cache them, and filter in memory.

See the GCP network/address example in [docs/development.md](../docs/development.md#referencing-mql-resources) for detailed implementation patterns.

## Important Notes

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting

### Security Considerations
- Be cautious with credentials and API tokens
- Follow OWASP guidelines when adding new features
- Validate inputs at system boundaries
- When cross-referencing MQL resources, use caching to minimize API calls

### Dependencies
- Managed via Go modules (`go.mod`)
- Each provider has separate dependencies
- Keep dependencies up to date
- Check for security vulnerabilities

### Performance Monitoring

Debug performance issues using Prometheus and Grafana:

```bash
# Install prometheus (macOS)
brew install prometheus

# Start monitoring stack
make metrics/start

# Run cnquery with metrics enabled
DEBUG=1 cnquery scan local
```

Access Grafana at http://localhost:3000 and connect to Prometheus at `http://host.docker.internal:9009`.

## Resources

- [Official Documentation](https://mondoo.com/docs/cnquery/home/)
- [MQL Introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL Language Reference](https://mondoo.com/docs/mql/resources/)
- [GitHub Repository](https://github.com/mondoohq/cnquery)
- [Community Discussions](https://github.com/orgs/mondoohq/discussions)

## Related Projects

- **cnspec**: Cloud-native security scanner built on cnquery
- **Mondoo Platform**: Web-based console for infrastructure exploration

## Provider-Specific Documentation

Many providers include detailed README files with authentication, examples, and troubleshooting:

- [ansible](../providers/ansible/README.md) - Playbook scanning with query/policy examples
- [ipinfo](../providers/ipinfo/README.md) - IP address information and geolocation
- [ms365](../providers/ms365/README.md) - Microsoft 365 with PowerShell requirements
- [os](../providers/os/README.md) - Operating system provider (Linux, macOS, Windows)
- [shodan](../providers/shodan/README.md) - Shodan search engine integration
- [snowflake](../providers/snowflake/README.md) - Snowflake data warehouse
- [tailscale](../providers/tailscale/README.md) - Tailscale network information

Run `find providers -name "README.md" -type f` to discover all provider documentation.

## Contributing

See [docs/development.md](../docs/development.md) for contribution guidelines.

---

*This context file helps AI assistants understand the cnquery project structure and make appropriate decisions when assisting with development tasks.*
