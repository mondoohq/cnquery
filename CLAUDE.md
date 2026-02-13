# Claude AI Context for cnquery

## Quick Decisions

Use these rules to make fast choices without reading the full guide.

**cnquery vs cnspec?** Resource, field, or data-gathering = cnquery. Policy assertion or vuln check = cnspec.

**Which resource implementation pattern?**
- Have all data from a list API → `CreateResource` with `__id` set
- Need to resolve by ID or expensive fetch → `NewResource` + `init` function
- Linking resources (address→network) → `init` with in-memory cache (avoids N+1)

**`__id` format?**
- AWS → ARN: `llx.StringDataPtr(thing.Arn)`
- OS/files → path: `llx.StringData(path)`
- Nested/composite → join with `/`: `parentArn + "/child/" + childName`

**Code generation after `.lr` change?**
- Fast (provider-only): `make providers/mqlr && ./mqlr generate providers/<p>/resources/<p>.lr --docs-file providers/<p>/resources/<p>.lr.manifest.yaml --dist providers/<p>/resources`
- Full: `make cnquery/generate`

**Build & test cycle?**
- Provider change only → `make providers/build/<p> && make providers/install/<p>`, then `cnquery` binary
- Core + provider change → `go run apps/cnquery/cnquery.go run ...`
- Debug with breakpoints → set `builtin: [<p>]` in `providers.yaml`, `make providers/config && make cnquery/install`

**Where to put tests?**
- OS provider + file parsing → extract to `resources/<name>/` subpackage, TOML mock in `testdata/`
- Other providers → follow existing mock patterns in that provider
- Integration tests are **required** for new resources

**Running a command on a remote system?** Never `os/exec`. Always `CreateResource(runtime, "command", ...)`. See `providers/os/resources/lsblk.go`.

**Error handling?**
- Permission denied → `Is400AccessDeniedError(err)` returns nil, not error
- Temporary failure (rate limit, network) → return actual error
- Single resource inaccessible → log warning, continue with rest

**Pagination?** Always handle it if the API supports it. Use marker/token loop pattern.

**Never edit** `*.lr.go` files — they're generated and will be overwritten.

---

## 1. Project Context

**cnquery** is a cloud-native infrastructure querying tool using **MQL (Mondoo Query Language)** to query resources across cloud accounts (AWS, Azure, GCP), Kubernetes, containers, OS internals, and APIs.

- **cnquery**: Core inventory tool. Resources, MQL, **data gathering**.
- **cnspec**: Security scanner built *on top* of cnquery. **Policy assertions** and vulnerability checks.
- For resource development (adding fields, new assets), you only work within **cnquery**.

## 2. Resource Development Lifecycle

The primary task in this repo is adding or modifying resources. Follow this lifecycle:

### Step 1: Define in `.lr` schema
Resources are defined in `.lr` files (e.g., `providers/aws/resources/aws.lr`). This acts as the GraphQL-like schema.

### Step 2: Generate code
**Crucial:** You must generate Go interfaces after modifying `.lr` files.
```bash
# Fast path (provider-only, recommended):
make providers/mqlr  # if mqlr binary is not there
./mqlr generate providers/aws/resources/aws.lr --docs-file providers/aws/resources/aws.lr.manifest.yaml --dist providers/aws/resources

# Full generation (slow):
make cnquery/generate
```

### Step 3: Implement
Implement the generated interfaces in the provider's Go code. See "Quick Decisions" above for which pattern to use.

**Pattern A: Immediate Mapping (`CreateResource`)**
*Best for:* Listing APIs where you get data immediately.
1.  Call the Cloud API.
2.  Loop over results.
3.  Map to MQL using `CreateResource(runtime, "aws.ec2.instance", map[string]*llx.RawData{...})`.
4.  **Requirement:** Set `__id` (e.g., ARN, UUID) for caching.

**Pattern B: Lazy Loading (`NewResource` + `init`)**
*Best for:* Resolving references (e.g., `aws.ec2.instance("i-123")`) or expensive calls.
1.  Return a reference: `NewResource(runtime, "aws.ec2.instance", map[string]*llx.RawData{"__id": ...})`.
2.  Implement an `init` function (e.g., `initAwsEc2Instance`) that checks for the `__id`, fetches data on-demand, and populates the resource.

**Pattern C: Cross-References**
*Best for:* Linking resources (e.g., GCP Address -> Network).
*   Use an `init` function to cache all instances and filter in memory to avoid N+1 API calls.

**Patterns to avoid:**
- **Never use `os/exec` or `exec.CommandContext` directly.** Use the `command` resource:
  ```go
  o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
      "command": llx.StringData("lsblk --json --fs"),
  })
  cmd := o.(*mqlCommand)
  if exit := cmd.GetExitcode(); exit.Data != 0 {
      return nil, errors.New("command failed: " + cmd.Stderr.Data)
  }
  output := cmd.Stdout.Data
  ```
  **Why?** Ensures proper execution context, auth, and connection handling across local/SSH/container. See [lsblk.go](providers/os/resources/lsblk.go).

### Step 4: Test (Required)
**Integration tests are required for new resources.** Interactive testing alone is not sufficient.

#### Unit Tests for Parsing Logic
Extract parsing logic into a separate package and write unit tests:
```
providers/os/resources/
├── limits.go                    # MQL resource wiring
└── limits/
    ├── limits.go                # Pure parsing logic (no MQL dependencies)
    ├── limits_test.go           # Unit tests
    └── testdata/
        └── linux.toml           # Mock file data
```
See `logindefs/`, `limits/`, `sshd/` for examples.

#### Integration Tests (OS provider — TOML mocks)
```go
func TestLimitsParser_MainConfig(t *testing.T) {
    conn, err := mock.New(0, &inventory.Asset{}, mock.WithPath("./testdata/linux.toml"))
    require.NoError(t, err)

    f, err := conn.FileSystem().Open("/etc/security/limits.conf")
    require.NoError(t, err)
    defer f.Close()

    content, err := io.ReadAll(f)
    require.NoError(t, err)

    entries := limits.ParseLines("/etc/security/limits.conf", string(content))
    require.Len(t, entries, 6)
}
```

TOML test data format:
```toml
[files."/etc/security/limits.conf"]
content = """# limits.conf content here
* soft nofile 65536
"""

[files."/etc/security/limits.d"]
stat.isdir = true
```

For other providers (AWS, GCP, Azure, etc.), follow existing test patterns in those providers.

#### Interactive Verification
1.  **Install**: `make cnquery/install` (one-time, or when changing cnquery core).
2.  **Provider**: `make providers/build/<provider> && make providers/install/<provider>` (after each provider change).
3.  **Test**: `cnquery run aws -c "aws.ec2.instances { __id, tags }"`

**Note:** Only use `go run apps/cnquery/cnquery.go run ...` when you're also modifying cnquery core code. For provider-only changes, just rebuild/install the provider and use the installed `cnquery` binary.

## 3. Build & Operations

### Prerequisites
*   Go 1.25.0+
*   Protocol Buffers v21+
*   **First time:** `make prep/tools` (installs protolint, mockgen, gotestsum, golangci-lint, copywrite)

### Common Commands
```bash
# Building
make cnquery/build                    # Build cnquery binary
make cnquery/install                  # Install to $GOBIN
make providers/build/aws              # Build specific provider
make providers/install/aws            # Install to ~/.config/mondoo/providers/
make providers/build/aws && make providers/install/aws  # Quick dev cycle

# Testing
make test/go/plain                    # All tests (excludes providers)
make test/lint                        # Linting
make providers/test                   # Test all providers
go test -v ./providers/core/...       # Specific package
go test ./llx -run TestArrayContains  # Specific test

# Code generation
make cnquery/generate                 # Full (slow)
make providers/mqlr                   # Build mqlr tool (fast provider-specific gen)
```

### Tips
*   **MCP Tools**: Use the GitHub MCP to check tickets/PRs. Use Notion MCP for internal docs.
*   **Auth**: The environment usually has AWS/Azure CLI tools authenticated. If not, stop and let me know.
*   **Tickets**: If the ticket body contains queries to run in cnquery, make use of them during exploration/dev/testing/verification.
*   **Provider READMEs**: Always check `providers/<provider-name>/README.md` when working with a specific provider.

## 4. Debugging

### Local Provider Debugging (main dev workflow)

Providers normally run as **separate subprocesses** (gRPC via `hashicorp/go-plugin`). Debuggers can't step into subprocess code. Marking a provider as `builtin` in `providers.yaml` compiles it directly into cnquery.

**Workflow:**
1.  **Edit `providers.yaml`**: Add provider to `builtin` (e.g., `builtin: [aws]`).
2.  **Config**: `make providers/config` (generates `builtin_dev.go`).
3.  **Build/Install**: `make cnquery/install`.
4.  **Run/Debug**:
    ```bash
    go run apps/cnquery/cnquery.go run aws -c "aws.ec2.instances"
    # Or use your IDE debugger with entry point: apps/cnquery/cnquery.go
    ```
5.  **Revert**: Clean up `providers.yaml` (set `builtin: []`) and run `make providers/config`.

Step 3 is the core of the work here (doing the ticket's local dev work). Steps 1-2 wrap the start; steps 4-5 wrap the end.

Use a debugger MCP if available — set breakpoints instead of stdout debugging.

## 5. Architecture

### Component Structure
```
cnquery/
├── cli/                    # CLI commands and execution runtime
├── mql/                    # MQL executor (high-level query interface)
├── mqlc/                   # MQL compiler (parses MQL to bytecode)
├── llx/                    # Low-level execution engine (bytecode VM)
├── providers/              # Provider coordinator and built-in providers
├── providers-sdk/v1/       # SDK for building provider plugins
├── explorer/               # Query bundles, packs, and execution orchestration
├── content/                # Built-in query packs and policies
└── apps/cnquery/           # Main cnquery CLI application
```

### Detailed Query Execution Flow
1. **User Query (MQL string)** → `mqlc.Compile()` (MQL Compiler)
2. **Compiled to `llx.CodeBundle`** (Protobuf-serialized bytecode + metadata)
3. **Wrapped in `explorer.ExecutionQuery`** (execution context)
4. **Executed by `executor.Executor`** (runs bytecode against runtime)
5. **Returns `llx.RawResult`** (typed data + code IDs)
6. **Formatted output** (JSON, YAML, table, etc.)

### Provider System Architecture

**Core Concepts:**
- **Providers** are plugins that connect cnquery to different infrastructure backends (AWS, K8s, Docker, etc.)
- Each provider is a separate Go module with its own dependencies
- Providers communicate with cnquery via gRPC using hashicorp/go-plugin
- The **Core Provider** is always built-in and provides universal resources like `asset`, `time`, `regex`

**Provider Lifecycle:**
1. `providers.Coordinator` spawns provider as subprocess via `exec.Command`
2. gRPC connection established via `hashicorp/go-plugin`
3. Provider implements: `ParseCLI()`, `Connect()`, `GetData()`, `StoreData()`, `Disconnect()`
4. `providers.Runtime` manages active providers for each asset
5. Providers can discover child assets (e.g., K8s discovers pods)

**Resource Data Flow:**
1. Query compiler requests resource field
2. Executor calls `provider.GetData(connection, resource, field, args)`
3. Provider fetches data from backend (cloud API, SSH, etc.)
4. Data converted to `llx.Primitive` → `llx.RawData`
5. Result cached in executor for subsequent access

### MQL, MQLC, and LLX Relationship
- **MQL** (`mql/`): High-level query executor API
- **MQLC** (`mqlc/`): Compiler that parses MQL text and generates bytecode
- **LLX** (`llx/`): Low-level virtual machine that executes bytecode

Think of it as: MQL (like SQL or even better GraphQL) → MQLC (compiler) → LLX (runtime VM)

### Resources and Code Generation
- Resources are defined in `.lr` files (e.g., `aws.lr`, `k8s.lr`)
- The `lr` tool generates Go code from these definitions:
    - Resource structs
    - Schema definitions
    - Data accessor methods
- Generated files: `*.lr.go`, `*.lr.manifest.yaml`, `*.resources.json`

### Resource Caching & __id
**How caching works:**
- Each resource instance has a unique cache key: `resourceName + "\x00" + __id`
- Example: `"aws.ec2.instance\x00arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"`
- The runtime checks cache before fetching: `if x, ok := runtime.Resources.Get(id); ok { return x, nil }`
- Results are cached automatically after first fetch

**Why `__id` matters:**
- Prevents redundant API calls for the same resource
- Enables resource sharing across queries
- Must be unique and stable (ARN, UUID, or composite key)
- If `__id` is empty or duplicate, caching breaks and performance degrades

### Code Generation Dependencies
The build process has several code generation steps:
1. Protocol buffers (`.proto` → `.pb.go`)
2. Resource definitions (`.lr` → `.lr.go`)
3. Provider configurations (`providers.yaml` → `builtin_dev.go`)

Always run `make cnquery/generate` after modifying any of these source files.

### Key Data Structures
When navigating the codebase, these are the critical types you'll encounter:
- **`llx.CodeBundle`** - Compiled MQL bytecode ready for execution
- **`llx.RawData`** - Typed data wrapper for resource fields
- **`llx.RawResult`** - Execution results with type information
- **`plugin.Runtime`** - Execution context linking assets to providers
- **`explorer.Explorer`** - High-level query orchestration
- **`explorer.ExecutionQuery`** - Query wrapped with execution context
- **`providers.Coordinator`** - Manages provider lifecycle and spawning
- **`providers.Runtime`** - Active provider management per asset

## 6. Development Workflow Patterns

See "Local Provider Debugging" above. That pattern should be used for most development (start, bulk of the work and verification, end).

But this also works:

### Adding a New Resource to a Provider
1. Edit the provider's `.lr` file (e.g., `providers/aws/resources/aws.lr`)
2. Run code generation:
   ```bash
   make providers/mqlr # if mqlr is not already available
   ./mqlr generate providers/aws/resources/aws.lr --docs-file providers/aws/resources/aws.lr.manifest.yaml --dist providers/aws/resources
   ```
3. Implement the resource methods in Go
4. Rebuild and install the provider:
   ```bash
   make providers/build/aws && make providers/install/aws
   ```

### Implementing Resource Cross-References
When one resource references another (e.g., GCP address → network), use an init function to cache all instances and filter in memory. This avoids excessive N+1 API calls.

### Provider Version Updates
Use the version utility to check and update provider versions:
```bash
# Set up alias (recommended)
alias version="go run providers-sdk/v1/util/version/version.go"

# Check which providers need version updates
version check providers/*/

# Update provider versions interactively
version update providers/*/

# Auto-increment and commit
version update providers/*/ --increment=patch --commit
```

### Using Go Workspaces for Multi-Repo Development
If developing cnquery alongside cnspec or providers, create a `go.work` file in a parent directory:
```go
go 1.25

use (
   ./cnquery
   ./cnquery/providers/aws
   ./cnquery/providers/k8s
   // add other providers as needed
   ./cnspec
)
```

## 7. Important Implementation Details

### Resource Caching & Performance
- Resource field access is lazy: fields are only fetched when needed
- Results are cached automatically by the executor using `__id` as cache key
- Cross-references should leverage this caching to avoid redundant API calls
- Use `init` functions for expensive operations to enable result sharing across queries

### Provider Connection Management
- Implement proper connection lifecycle: `Connect()`, `GetData()`, `StoreData()`, `Disconnect()`
- Handle authentication failures gracefully with `Is400AccessDeniedError()` checks
- Use connection pooling where possible to optimize API calls
- Implement timeout handling for long-running API operations

### Error Handling Patterns
- Use `Is400AccessDeniedError(err)` for permission issues (returns `nil` result, not error)
- Return actual errors for temporary failures (rate limits, network issues)
- Log warnings for region/permission issues but continue with accessible resources
- Avoid failing entire queries due to single resource access issues

### Pagination Handling
When fetching resources from cloud APIs, **always handle pagination** if the API supports it:
```go
var marker *string
for {
    result, err := svc.DescribeDBParameterGroups(ctx, &rds.DescribeDBParameterGroupsInput{Marker: marker})
    if err != nil {
        return nil, err
    }
    
    for _, item := range result.Items {
        // Process each item
    }
    
    // Check if more pages exist
    if result.Marker == nil {
        break
    }
    marker = result.Marker
}
```

### Resource Field Naming & Constraints
- Properties named "id" or "url" (case insensitive) must be prefixed with "userDefined:" (e.g., "userDefined:URL")
- Time/date fields must use `createdAt`/`updatedAt`/`modifiedAt`/`deletedAt` naming convention — not `createDate`, `modifyTime`, `creation_time`, etc.
- Date fields use expanded format: "date:{property}:start", "date:{property}:end", "date:{property}:is_datetime"
- Place fields split into multiple properties: name, address, latitude, longitude, google_place_id
- Use JavaScript number types for numeric fields, not strings
- Prefer typed resource references over raw ID strings. Instead of a `vpcId string` field, define a `vpc aws.vpc` field that returns the actual resource. This enables MQL traversal (e.g., `aws.ec2.instance.vpc.cidrBlock`) instead of requiring users to manually look up IDs.
- In `.lr.manifest.yaml`, new fields only need `min_mondoo_version` if the resource itself has an older `min_mondoo_version`. If the resource already requires a recent enough version, fields inherit it implicitly.

### Provider Modules & Dependencies
- Each provider in `providers/` has its own `go.mod` for isolation
- Core cnquery has dependencies that providers don't need (and vice versa)
- This keeps provider binaries smaller and dependency trees isolated
- Update provider versions using the version utility to maintain compatibility

### Built-in vs External Providers
- Core provider is always compiled into cnquery (provides universal resources)
- Other providers can be:
    - External plugins (default): separate binaries loaded at runtime via gRPC
    - Built-in (for debugging): compiled into cnquery by modifying `providers.yaml`
- Built-in mode enables easier debugging but requires provider cleanup before commits
- And speaking of debugging: use a debugger mcp if available, so you set breakpoints instead of stdout debugging.

### Code Generation Gotchas
- Always run `make cnquery/generate` after modifying `.lr`, `.proto`, or `providers.yaml` files
- Generated code includes resource structs, schema definitions, and accessor methods
- Never manually edit generated `.lr.go` files - they get overwritten
- Use `make providers/mqlr` for faster provider-specific regeneration

### Testing & Verification
- **Integration tests are required for new resources** - do not skip this step
- For OS provider resources, use TOML-based mock connections (see `limits/`, `logindefs/` for examples)
- For other providers, follow existing test patterns in that provider's codebase
- Extract parsing logic into separate packages for easier unit testing
- If you want to test simple changes, build and install the provider and use cnquery run ....
- Otherwise set it as builtin and use go run ...
- Use `demo.agent.credentials.json` for local development with service accounts
- Verify credentials exist before testing: `~/.aws/credentials`, etc.
- Test error conditions and edge cases during development
- Use `providers-sdk/v1/testutils` for mock providers in unit tests
- Recording/replay system available for reproducible provider tests

### Provider Structure
Each provider follows a standard directory layout:
- **`config/`** - Provider configuration and settings
- **`connection/`** - Connection management and authentication
- **`provider/`** - Provider implementation (ParseCLI, Connect, GetData, etc.)
- **`resources/`** - Resource implementations and .lr files
- **`main.go`** - Provider binary entry point
- **`gen/main.go`** - Generates CLI configuration JSON

### CLI and Output
Key directories for user-facing functionality:
- **`apps/cnquery/cmd/`** - CLI command implementations
- **`cli/shell/`** - Interactive shell with auto-completion
- **`cli/reporter/`** - Output formatting (JSON, CSV, YAML, table, etc.)

**Always use the codebase's patterns.**

## 8. Pre-PR Checklist

When work appears complete, present this checklist to the user for local verification:

### Essential Checks (Run These)
```bash
# 1. Format all Go code with gofmt
gofmt -w .
git diff --exit-code  # Should show no changes if already formatted

# 2. Ensure generated code is up-to-date
make cnquery/generate
git diff --exit-code  # Should show no changes

# 3. Verify go.mod is clean
go mod tidy
git diff go.mod go.sum  # Should show no changes

# 4. Run linting
make test/lint

# 5. Run unit tests
make test/go/plain
```

### Provider-Specific Checks
If you modified a provider:
```bash
# 1. Build and install the provider
make providers/build/<provider> && make providers/install/<provider>

# 2. Interactive verification
cnquery shell <provider>
# Run relevant MQL queries from the ticket

# 3. Run provider tests (if they exist)
go test -v ./providers/<provider>/...
```

### Optional (Performance-Sensitive or Core Changes)
```bash
# Race condition detection (if touching concurrency)
make race/go

# Integration tests (if changing core execution)
make test/integration
```

### Quick Pre-Commit Checklist
- [ ] All Go code is formatted with `gofmt -w .`
- [ ] Generated files are up-to-date (`.lr.go`, `.pb.go`)
- [ ] Linting passes (`make test/lint`)
- [ ] **New resources have integration tests** (required, not optional)
- [ ] Changes work interactively (`cnquery shell <provider>`)
- [ ] `go.mod` is clean (`go mod tidy`)
- [ ] No spelling errors in new comments/docs

**Note:** CI runs comprehensive checks. Run them locally only if you want to verify before pushing or if changing core/performance-critical code.

## 9. Commit Conventions

Use emojis in commit messages (but don't worry about it, since you're NEVER going to commit anything; that's my job):
- 🛑 breaking changes
- 🐛 bugfix
- 🧹 cleanup/internals
- ⚡ speed improvements
- 📄 docs
- ✨⭐🌟🌠 features (smaller to larger)
- 🌈 visual changes
- 🐎 race condition fixes
- 🌙 MQL changes
- 🟢 fix tests
- 🎫 auth
- 🐳 container

## 10. Additional Resources

### External Documentation
- [Official Documentation](https://mondoo.com/docs/cnquery/home/)
- [MQL Introduction](https://mondoohq.github.io/mql-intro/index.html)
- [MQL Language Reference](https://mondoo.com/docs/mql/resources/)
- [GitHub Repository](https://github.com/mondoohq/mql)
- [Community Discussions](https://github.com/orgs/mondoohq/discussions)

### Provider-Specific Documentation
Many providers include detailed README files with authentication, examples, and troubleshooting:
- [ansible](providers/ansible/README.md) - Playbook scanning with query/policy examples
- [ipinfo](providers/ipinfo/README.md) - IP address information and geolocation
- [ms365](providers/ms365/README.md) - Microsoft 365 with PowerShell requirements
- [os](providers/os/README.md) - Operating system provider (Linux, macOS, Windows)
- [shodan](providers/shodan/README.md) - Shodan search engine integration
- [snowflake](providers/snowflake/README.md) - Snowflake data warehouse
- [tailscale](providers/tailscale/README.md) - Tailscale network information

Run `find providers -name "README.md" -type f` to discover all provider documentation.

### Related Projects
- **cnspec**: Cloud-native security scanner built on cnquery
- **Mondoo Platform**: Web-based console for infrastructure exploration

Anticipate needs, offer options when it applies, think in the context of ticket-solution-in-codebase.