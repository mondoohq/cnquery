# Claude AI Context for mql

This directory contains information to help Claude AI assistants understand and work effectively with the mql codebase.

## 1. Project Context

**mql** (formerly cnquery) is a cloud-native infrastructure querying tool. It uses **MQL (Mondoo Query Language)** to query over 1,300 resources across cloud accounts (AWS, Azure, GCP), Kubernetes, containers, OS internals, and APIs.

> **v13 Rename:** In v13 the project was renamed from `cnquery` to `mql`. The Go module is `go.mondoo.com/mql/v13`, the CLI binary is `mql`, and all build targets use the `mql/` prefix. The `scan` and `sbom` subcommands have moved to **cnspec**.

### Critical Distinction
*   **mql**: The core inventory tool. Defines resources, implements MQL, and handles **data gathering**.
*   **cnspec**: The security scanning tool built *on top* of mql. It implements **policy assertions**, vulnerability checks, **scanning** (`scan`), and **SBOM generation** (`sbom`).
*   **Rule of Thumb:** For resource development (adding fields, new assets), you only need to work within **mql**.

## 2. Resource Development Guide

The primary task in this repo is adding or modifying resources. Follow this lifecycle:

### Step 1: Definition (`.lr` schema)
Resources are defined in `.lr` files (e.g., `providers/aws/resources/aws.lr`). This acts as the GraphQL-like schema.
*   **Action**: Edit the `.lr` file to add new resources or fields.

### Step 2: Code Generation
**Crucial:** You must generate Go interfaces after modifying `.lr` files.
```bash
# Generate all code (slow)
make mql/generate

# Generate specific provider resources (fast - recommended)
# (if the mqlr binary is not there:)
make providers/mqlr
./mqlr generate providers/aws/resources/aws.lr --dist providers/aws/resources
```

### Step 3: Implementation Strategies
Implement the generated interfaces in the provider's Go code. Use one of these patterns:

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

**Pattern D: Internal Structs for Caching & Cross-References**
*Best for:* Storing data from the creation context that's needed later for lazy-loaded typed resource references.

The code generator detects `mql<ResourceName>Internal` structs and embeds them into the generated resource struct. Use them to cache values needed for computed methods:

```go
type mqlAwsDocumentdbSnapshotInternal struct {
    cacheVpcId    *string
    cacheKmsKeyId *string
}

// In the creator function, after CreateResource:
mqlSnapshot := resource.(*mqlAwsDocumentdbSnapshot)
mqlSnapshot.cacheVpcId = snapshot.VpcId
mqlSnapshot.cacheKmsKeyId = snapshot.KmsKeyId

// Lazy-loaded typed reference method:
func (a *mqlAwsDocumentdbSnapshot) vpc() (*mqlAwsVpc, error) {
    if a.cacheVpcId == nil || *a.cacheVpcId == "" {
        a.Vpc.State = plugin.StateIsNull | plugin.StateIsSet
        return nil, nil
    }
    mqlVpc, err := NewResource(a.MqlRuntime, "aws.vpc",
        map[string]*llx.RawData{"id": llx.StringDataPtr(a.cacheVpcId)})
    if err != nil {
        return nil, err
    }
    return mqlVpc.(*mqlAwsVpc), nil
}
```

**Important:** If you add an Internal struct *after* the first code generation, you must run `./mqlr generate` a **second time** for the generator to detect and embed it.

**`securityGroupIdHandler`**: A reusable embedded struct (defined in `aws_ec2.go`) for converting security group ID lists to typed `[]aws.ec2.securitygroup` references. Embed it in your Internal struct:
```go
type mqlAwsRdsProxyInternal struct {
    securityGroupIdHandler  // provides securityGroups() automatically
    region    string
    accountID string
}
```

**Never hardcode empty/default values for fields the API doesn't return.** If a list API (e.g., `ListDataCatalogs`) returns a summary without certain fields (e.g., `description`, `parameters`), do NOT set them to empty strings or nil maps in `CreateResource`. Instead:
1. Declare them as computed methods in `.lr`: `description() string` (not `description string`)
2. Implement a lazy-loading fetch function that calls the detail API (e.g., `GetDataCatalog`) on demand
3. Cache the results in an Internal struct to avoid repeated API calls

**Lazy-loading with double-check locking** (for fields that require a separate API call):
```go
type mqlAwsResourceInternal struct {
    fetched bool
    attrs   map[string]string
    lock    sync.Mutex
}

func (a *mqlAwsResource) fetchAttributes() (map[string]string, error) {
    if a.fetched { return a.attrs, nil }
    a.lock.Lock()
    defer a.lock.Unlock()
    if a.fetched { return a.attrs, nil }
    // ... fetch from API ...
    a.fetched = true
    a.attrs = resp.Attributes
    return a.attrs, nil
}
```
Multiple computed methods can share the same fetch function to batch-load related fields from a single API call.

**Patterns to avoid**
- **Never use `os/exec` or `exec.CommandContext` directly.** Instead, use the `command` resource to delegate execution through the provider system:
  ```go
  // WRONG: Do not do this
  cmd := exec.CommandContext(ctx, "lsblk", "--json", "--fs")
  output, err := cmd.Output()

  // CORRECT: Use the command resource
  o, err := CreateResource(runtime, "command", map[string]*llx.RawData{
      "command": llx.StringData("lsblk --json --fs"),
  })
  if err != nil {
      return nil, err
  }
  cmd := o.(*mqlCommand)
  if exit := cmd.GetExitcode(); exit.Data != 0 {
      return nil, errors.New("command failed: " + cmd.Stderr.Data)
  }
  output := cmd.Stdout.Data
  ```
  **Why?** The `command` resource ensures proper execution context, authentication, connection handling, and works seamlessly across different connection types (local, SSH, container, etc.). See [lsblk.go](providers/os/resources/lsblk.go) for a complete example.

- **Never `return nil, nil` from a singular resource accessor without setting `StateIsNull` first.** When an accessor returns a single resource pointer (e.g., `(*mqlAwsSomeResource, error)`) and the value is legitimately null, you **must** set the field's state before returning. Otherwise the runtime doesn't know the field was resolved and may panic or re-fetch indefinitely.
  ```go
  // WRONG: causes panic — runtime doesn't know the field is null
  func (a *mqlAwsRoute53Record) healthCheck() (*mqlAwsRoute53HealthCheck, error) {
      if healthCheckId == "" {
          return nil, nil
      }
  }

  // CORRECT: mark field as set + null, then return nil
  func (a *mqlAwsRoute53Record) healthCheck() (*mqlAwsRoute53HealthCheck, error) {
      if healthCheckId == "" {
          a.HealthCheck.State = plugin.StateIsSet | plugin.StateIsNull
          return nil, nil
      }
  }
  ```
  This applies to all `return nil, nil` paths: empty/missing IDs, access-denied fallbacks, nil API responses, and unset optional fields (e.g., `!a.Field.IsSet()`). Functions returning slices, maps, or basic types are **not** affected — only singular resource pointer returns.

### Step 4: Verification (Interactive)
Automated tests are rare for MQL resources (thin wrappers). **Interactive testing is standard.**

1.  **Install**: `make mql/install` (one-time, or when changing mql core).
2.  **Provider**: `make providers/build/<provider> && make providers/install/<provider>` (after each provider change).
3.  **Test**: Use your installed `mql` binary directly (e.g., `mql run aws -c "aws.ec2.instances { __id, tags }"`).

**Note:** Only use `go run apps/mql/mql.go run ...` when you're also modifying mql core code (not just the provider). For provider-only changes, just rebuild/install the provider and use your installed mql binary.

## 3. Build & Operations

### Prerequisites
*   Go 1.25.0+
*   Protocol Buffers v21+
*   **Install development tools first:** `make prep/tools` (installs protolint, mockgen, gotestsum, golangci-lint, copywrite)

### Building mql
```bash
# Build all providers and generate code
make providers

# Build the mql binary
make mql/build

# Install mql to $GOBIN
make mql/install

# Build for specific platform
make mql/build/linux
make mql/build/darwin
make mql/build/windows
```

### Working with Providers
```bash
# Build a specific provider
make providers/build/aws
make providers/build/k8s

# Install provider to local config (~/.config/mondoo/providers/) so it can be used by mql
make providers/install/aws

# Build provider for distribution (production build)
make providers/dist

# Quick rebuild and install after changing a provider
make providers/build/aws && make providers/install/aws
```

### Testing Commands
```bash
# Run all tests (excludes providers and integration tests)
make test/go/plain

# Run tests with CI output (generates JUnit XML report)
make test/go/plain-ci

# Run integration tests
make test/integration

# Test all providers
make providers/test

# Run linting
make test/lint

# Extended linting (more comprehensive)
make test/lint/extended

# Race condition detection
make race/go
```

### Tips
*   **MCP Tools**: Use the GitHub MCP to check tickets/PRs. Use Notion MCP for internal docs. We're going to be talking about tickets and PRs (so that's github mcp), and there's also notion for company-wide docs (focus on Engineering stuff, infra, dev env, etc)
*   **Auth**: The environment usually has AWS/Azure CLI tools authenticated (so you can use them when needed). If they're not present or logged in, stop and let me know so I can setup the provider's needs (tools, auth, whatever)
*   **Tickets**: If the ticket body contains queries to run in mql, make use of them during exploration/dev/testing/verification.
*   **Provider READMEs**: Many providers have detailed README files with authentication methods, prerequisites, usage examples, and troubleshooting. Always check `providers/<provider-name>/README.md` when working with a specific provider.

## 4. Debugging & Profiling

### Local Provider Debugging

Providers run as separate gRPC subprocesses, so debuggers can't step into them. Mark the provider as `builtin` in `providers.yaml` to compile it into the main `mql` binary for in-process debugging:

1. Add to `builtin:` list in `providers.yaml` (e.g. `builtin: [aws]`)
2. `make providers/config` (regenerates `builtin_dev.go`)
3. `make mql/install`
4. `go run apps/mql/mql.go run aws -c "aws.ec2.instances"` (or attach IDE debugger to `apps/mql/mql.go`)
5. Revert: set `builtin: []` and re-run `make providers/config`

The middle step (3) is where the actual ticket work happens; 1–2 and 5 just toggle the mode.

### Remote Debugging
For providers that need to run on specific VMs (e.g., GCP snapshot scanning):
1. Install Go and Delve on the remote VM
2. Configure provider as builtin locally (see "Debugging Providers" above)
3. Copy source to remote VM (use `rsync` for easier iteration)
4. Allow ingress traffic to debugger port in firewall
5. Start debugger on remote VM:
   ```bash
   dlv debug apps/mql/mql.go --headless --listen=:12345 -- run gcp snapshot --project-id xyz-123 --verbose
   ```

## 5. Architecture Deep Dive

### High-Level Component Structure
```
mql/
├── cli/                    # CLI commands and execution runtime
├── mql/                    # MQL executor (high-level query interface)
├── mqlc/                   # MQL compiler (parses MQL to bytecode)
├── llx/                    # Low-level execution engine (bytecode VM)
├── providers/              # Provider coordinator and built-in providers
├── providers-sdk/v1/       # SDK for building provider plugins
├── explorer/               # Query bundles, packs, and execution orchestration
├── content/                # Built-in query packs and policies
└── apps/mql/               # Main mql CLI application
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
- **Providers** are plugins that connect mql to different infrastructure backends (AWS, K8s, Docker, etc.)
- Each provider is a separate Go module with its own dependencies
- Providers communicate with mql via gRPC using hashicorp/go-plugin
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
- Generated files: `*.lr.go`, `*.lr.versions`, `*.resources.json`, `*.permissions.json`

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

**Performance notes:**
- Resource field access is lazy: fields are only fetched when needed
- Cross-references should leverage caching to avoid redundant API calls
- Use `init` functions for expensive operations to enable result sharing across queries

### Code Generation Dependencies
The build process has several code generation steps:
1. Protocol buffers (`.proto` → `.pb.go`)
2. Resource definitions (`.lr` → `.lr.go`)
3. Provider configurations (`providers.yaml` → `builtin_dev.go`)

Always run `make mql/generate` after modifying any of these source files.

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

### LLX Builtin Function Invariants
When implementing or modifying builtin functions in `llx/` (e.g., array/dict operations):
- **Never mutate slices or maps obtained from `arg.Value` or `bind.Value`.** These reference shared runtime data — the runtime caches resolved values and may return the same `RawData` pointer to multiple callers. Mutating them corrupts the original value for any subsequent use. Always copy before modifying:
  ```go
  // WRONG: mutates the argument's backing array in-place
  filters := arg.Value.([]any)
  filters = append(filters[0:j], filters[j+1:]...)  // destroys arg.Value for future callers

  // CORRECT: copy first, then mutate the copy
  argFilters := arg.Value.([]any)
  filters := make([]any, len(argFilters))
  copy(filters, argFilters)
  filters = append(filters[0:j], filters[j+1:]...)  // safe — original untouched
  ```

## 6. Development Workflow Patterns

The primary workflow for provider changes is the "Local Provider Debugging" pattern in Section 4. For resource-only changes, follow Steps 1-4 in Section 2.

### Provider Version Updates
```bash
alias version="go run providers-sdk/v1/util/version/version.go"
version check providers/*/                              # check which need updates
version update providers/*/                              # interactive update
version update providers/*/ --increment=patch --commit   # auto-increment and commit
```

### Go Workspaces for Multi-Repo Development
When developing mql alongside cnspec, create a `go.work` in a parent directory with `use (./mql, ./mql/providers/aws, ./cnspec)` etc.

## 7. Important Implementation Details

### Copyright & License Headers
Every source file (`.go`, `.lr`, `.proto`, etc.) must begin with:
```
// Copyright Mondoo, Inc. 2024, <current year>
// SPDX-License-Identifier: BUSL-1.1
```
The format is `2024, <current year>` (comma-separated, not a dash). The first year (2024) is fixed; the second year should be the current calendar year. This is enforced by the `copywrite` tool installed via `make prep/tools`.

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
- Date fields use expanded format: "date:{property}:start", "date:{property}:end", "date:{property}:is_datetime"
- Place fields split into multiple properties: name, address, latitude, longitude, google_place_id
- Use JavaScript number types for numeric fields, not strings
- **Always use typed resource references over raw ID/ARN strings.** This is critical for good MQL UX:
  - `vpcId string` → `vpc() aws.vpc`
  - `vpcSubnetIds []string` → `subnets() []aws.vpc.subnet`
  - `vpcSecurityGroupIds []string` → `securityGroups() []aws.ec2.securitygroup`
  - `roleArn string` → `iamRole() aws.iam.role` (use `iamRole` not `role` to avoid ambiguity)
  - `kmsMasterKeyId string` → `kmsMasterKey() aws.kms.key`
  - `topicArn string` → `topic() aws.sns.topic`
  - `streamArn string` → `stream() aws.kinesis.stream`
  These enable MQL traversal (e.g., `aws.rds.proxy.vpc.cidrBlock`) instead of requiring manual lookups. Store the raw ID/ARN in a `cache*` field on the Internal struct, then implement the typed method using `NewResource`.
- **GCP: Use typed resource references over raw URL strings.** GCP Compute resources often store references as self-link URLs (e.g., `https://www.googleapis.com/compute/v1/projects/.../networks/my-net`). Always add a typed computed method alongside the raw URL field:
  - `networkUrl string` → `network() gcp.project.computeService.network`
  - `subnetworkUrl string` → `subnetwork() gcp.project.computeService.subnetwork`
  - `routerUrl string` → `router() gcp.project.computeService.router`
  - `sslPolicyUrl string` → `sslPolicy() gcp.project.computeService.sslPolicy`
  - `securityPolicyUrl string` → `securityPolicy() gcp.project.computeService.securityPolicy`
  - `interconnectUrl string` → `interconnect() gcp.project.computeService.interconnect`
  - `vpnGatewayUrl string` → `vpnGateway() gcp.project.computeService.vpnGateway`
- **When to create a sub-resource.** A sub-resource (`aws.foo.bar`) is appropriate only when it meets **one of two bars**:
  1. **Clear ID.** It has a stable unique identifier — an ARN, a name-with-region, or another natural key returned by the API. Synthetic composite IDs like `<parentArn>/leaf` do **not** count.
  2. **Nested typed reference.** It exists to hold typed refs to other modeled resources (e.g. a `computeEnvironmentOrder` sub-resource whose reason to exist is `computeEnvironment() aws.batch.computeEnvironment`).

  If neither applies, **do not create a sub-resource**. Instead:
  - **Flatten scalars into the parent** with a disambiguating prefix. Prefer `linuxMaxSwap int`, `linuxSharedMemorySize int`, `fargatePlatformVersion string` over a single-scalar `linuxParameters`/`fargatePlatformConfiguration` sub-resource.
  - **Use `map[string]string`** for name/value pair lists that would otherwise be a struct with two string fields (e.g. `environmentVariables map[string]string` rather than a `{name, value}` sub-resource — keys of `GPU`/`MEMORY`/`VCPU` work the same way for resource requirements).
  - **Use `[]dict`** for small heterogeneous structs (e.g. Linux devices, tmpfs mounts, evaluate-on-exit conditions) where no individual field warrants its own typed audit query.

  **Why:** A sub-resource has real cost — a generated struct, `__id` stability, serialization, test surface, and a new entry in `.lr.versions` for every field. Creating them for pure data containers bloats the schema without giving auditors new query power. Scalars on the parent are already queryable; a map lets you do `resourceRequirements["GPU"]` without a new resource type.

  **Examples (Batch):**
  - ✓ `aws.batch.jobQueue.computeEnvironmentOrder` — nests typed `computeEnvironment()` ref
  - ✓ `aws.batch.jobDefinition.containerProperties.secret` — `valueFrom` is the ARN (and will eventually resolve to a typed ref)
  - ✗ `{name, value}` env var — use `map[string]string`
  - ✗ `{type, value}` resource requirement — use `map[string]string` keyed by type
  - ✗ `{platformVersion}` single-scalar config — flatten to `fargatePlatformVersion` on parent
  - ✗ `{hostPath, containerPath, permissions}` device / `{containerPath, size, mountOptions}` tmpfs — use `[]dict`
- Every resource and field has an explicit entry in `.lr.versions`. New entries must use the **next patch version** after the provider's current version (e.g., if the provider is at `13.1.1`, new fields should be `13.1.2`). The provider's current version is in `providers/<name>/config/config.go` (look for the `Version` field). Do **not** rely on the highest version in `.lr.versions` — it may be stale from before a major version bump. The `versions` command does this automatically, but verify the result. Existing entries are never overwritten.
- **Match SDK types faithfully:** If an SDK field is `*bool`, use `bool` in `.lr` and `llx.BoolDataPtr()` in Go — don't cast it to `string`. If an SDK enum has only two states (Enabled/Disabled), prefer `bool`. Use `*type` intermediate variables with `llx.*DataPtr` helpers to preserve nil semantics.
- **Consistency with existing fields:** Before adding new fields to a resource, check how its existing fields handle pointers, nil checks, and type conversions. Follow the same pattern.
- **Verify enum values in `.lr` comments:** When listing possible values in field comments, check the SDK/API docs for completeness — don't assume the set is closed.
- **Skip deprecated SDK fields and methods.** Before exposing a proto field or calling a method, check the SDK's `// Deprecated:` comment. Deprecated fields often return empty/zero on modern instances because the data has moved elsewhere (e.g. GCP Memorystore moved `DiscoveryEndpoints`/`PscAutoConnections` into `Endpoints`). Modeling them anyway adds dead schema that looks queryable but never returns data. Same goes for `Get*`/`List*` methods marked deprecated — pick the replacement. If you genuinely need a deprecated field for backward-compat with old API responses, leave a comment explaining why.
- **Deprecating fields and resources — use `@maturity`.** When a field or resource is being kept for backward-compat but should not be used by new audits, mark it with `@maturity("deprecated")` in the `.lr` schema and prefix the field's doc comment with `DEPRECATED:`. The comment must explain what to use instead. Don't use a separate `// Deprecated:` line — `@maturity` is the source of truth and the comment is what users see. Also valid: `@maturity("preview")` for fields whose shape may still change. Examples:
  ```
  // DEPRECATED: use endpointAddress, endpointPort, and endpointHostedZoneId instead
  endpoint @maturity("deprecated") dict
  // DEPRECATED: replaced by aws.foo.bar.newField in 13.20.0
  legacyField @maturity("deprecated") string
  // a whole resource can be deprecated too
  private cloudflare.zone.plan @defaults("name") @maturity("deprecated") { ... }
  ```
  Keep the existing `.lr.versions` entry at its original version — deprecation does not bump the version.

### Provider Modules & Dependencies
- Each provider in `providers/` has its own `go.mod` for isolation
- Core mql has dependencies that providers don't need (and vice versa)
- This keeps provider binaries smaller and dependency trees isolated
- Update provider versions using the version utility to maintain compatibility

### Built-in vs External Providers
- Core provider is always compiled into mql (provides universal resources)
- Other providers can be:
    - External plugins (default): separate binaries loaded at runtime via gRPC
    - Built-in (for debugging): compiled into mql by modifying `providers.yaml`
- Built-in mode enables easier debugging but requires provider cleanup before commits
- And speaking of debugging: use a debugger mcp if available, so you set breakpoints instead of stdout debugging.

### Code Generation Gotchas
- Always run `make mql/generate` after modifying `.lr`, `.proto`, or `providers.yaml` files
- Generated code includes resource structs, schema definitions, and accessor methods
- Never manually edit generated `.lr.go` files - they get overwritten
- Use `make providers/mqlr` for faster provider-specific regeneration
- **Internal structs require a second generation pass.** If you add a new `mql*Internal` struct to a Go file, the first `./mqlr generate` won't embed it (the struct didn't exist when the generator ran). Run `./mqlr generate` again after adding Internal structs.
- **Always commit `*.permissions.json`** when it changes. This file is regenerated by `make providers/build/<provider>` and tracks the IAM permissions each provider requires. Changes to it (e.g., new API calls added, deprecated calls removed) are part of the PR — they're not throwaway build artifacts.

### Testing & Verification
- If you want to test simple changes, build and install the provider and use mql run ....
- Otherwise set it as builtin and use go run ...
- Use `demo.agent.credentials.json` for local development with service accounts
- Verify credentials exist before testing: `~/.aws/credentials`, etc.
- Test error conditions and edge cases during development.
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

### Creating a New Provider
Use the scaffolding tool to generate the provider skeleton:
```bash
go run apps/provider-scaffold/provider-scaffold.go \
  --path providers/your-provider \
  --provider-id your-provider \
  --provider-name "Your Provider"
cd providers/your-provider && go mod tidy
```
The Go package path is derived automatically as `go.mondoo.com/mql/v13/providers/{provider-id}`.
New providers use the ID scheme `go.mondoo.com/mql/providers/{provider-id}` (no version in the ID).

After scaffolding, you must also register the provider in these files:
- **`providers/defaults.go`** — Add a default entry (alphabetically) so the CLI discovers the provider before it's installed
- **`README.md`** — Add a row to the provider table (alphabetically)
- **`DEVELOPMENT.md`** — Add the provider path to the `go.work` provider list (alphabetically)
- **`Makefile`** — Add the provider name to the `PROVIDERS` list (alphabetically)

### CLI and Output
Key directories for user-facing functionality:
- **`apps/mql/cmd/`** - CLI command implementations
- **`cli/shell/`** - Interactive shell with auto-completion
- **`cli/reporter/`** - Output formatting (JSON, CSV, YAML, table, etc.)

**Always use the codebase's patterns.**

## 8. Pre-PR Checklist

When work appears complete, present this checklist to the user for local verification:

### Essential Checks (Run These)
```bash
# 1. Run gofmt on any Go files you changed
gofmt -w <changed .go files>

# 2. Ensure generated code is up-to-date
make mql/generate
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
mql shell <provider>
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
- [ ] `gofmt -w` run on all changed `.go` files
- [ ] Generated files are up-to-date (`.lr.go`, `.pb.go`, `.permissions.json`)
- [ ] Linting passes (`make test/lint`)
- [ ] Changes work interactively (`mql shell <provider>`)
- [ ] `go.mod` is clean (`go mod tidy`)
- [ ] No spelling errors in new comments/docs

**Note:** CI runs comprehensive checks. Run them locally only if you want to verify before pushing or if changing core/performance-critical code.

### Spell Check CI
CI runs [check-spelling/check-spelling](https://github.com/check-spelling/check-spelling) on every PR. It **only checks `.md`, `.lr`, and `.mql.yaml` files** (configured in `.github/actions/spelling/only.txt`).

**How it works:**
- Uses external dictionaries for AWS, Kubernetes, Go, and software terms
- `.github/actions/spelling/expect.txt` — sorted list of allowed custom words (add new terms here)
- `.github/actions/spelling/line_forbidden.patterns` — enforces correct capitalization of product names (e.g., "GitHub" not "Github", "Kubernetes" not "kubernetes")

**Fixing spell check failures:**
- If the word is a legitimate technical term: add it to `expect.txt` (keep sorted)
- If the word appears in a pattern (ARN, hash, URL): add a regex to `patterns.txt`
- If it's a genuine typo: fix the spelling
- Removing words from `expect.txt` is fine as long as the spell check CI job passes

## 9. Commit Conventions

When committing or opening a PR, **start the title with one of these emoji** to mark the change kind:
- 🛑 breaking changes
- 🐛 bugfix
- 🧹 cleanup/internals
- ⚡ speed improvements
- 📄 docs
- ✨⭐🌟🌠 features (smaller to larger; pick by ambition)
- 🌈 visual changes
- 🐎 race condition fixes
- 🌙 MQL changes
- 🟢 fix tests
- 🎫 auth
- 🐳 container

Use these in `git commit -m`, `gh pr create --title`, and any commit-message HEREDOCs. Match recent commits in `git log --oneline` if a change spans multiple kinds — pick the dominant one.

Anticipate needs, offer options when it applies, think in the context of ticket-solution-in-codebase.
