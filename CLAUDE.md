# Claude AI Context for mql

This directory contains information to help Claude AI assistants understand and work effectively with the mql codebase.

## 1. Project Context

**mql** (formerly cnquery) is a cloud-native infrastructure querying tool. It uses **MQL (Mondoo Query Language)** to query over 1,300 resources across cloud accounts (AWS, Azure, GCP), Kubernetes, containers, OS internals, and APIs.

### Critical Distinction
*   **mql**: The core inventory tool. Defines resources, implements MQL, and handles **data gathering**.
*   **cnspec**: The security scanning tool built *on top* of mql. It implements **policy assertions**, vulnerability checks, **scanning** (`scan`), and **SBOM generation** (`sbom`).
*   **Rule of Thumb:** For resource development (adding fields, new assets), you only need to work within **mql**.

## 2. Resource Development Guide

The primary task in this repo is adding or modifying resources. Follow this lifecycle:

### Step 1: Definition (`.lr` schema)
Resources are defined in `.lr` files (e.g., `providers/aws/resources/aws.lr`). This acts as the GraphQL-like schema.
*   **Action**: Edit the `.lr` file to add new resources or fields.

#### Resource doc-comment format

Every top-level resource (anything users will query directly — including singular records like `aws.ec2.instance` and namespace roots like `aws`) must have a two-part doc-comment immediately above the `resource {` line:

1. **Title.** A simple, technically correct one-line name — a noun phrase, no leading article ("A" / "An"), no trailing verbs like "static analysis" / "configuration analysis". The title is *what the resource is*, not what you do with it. **Must not start with "deprecated"** (enforced) — use `@maturity("deprecated")` instead. **Max 150 characters (enforced).**
2. **Single empty `//` line.**
3. **Description.** Multi-line prose describing what's queryable through the resource — fields, sub-resources, derived predicates, the audits it enables. **Lead with the noun being exposed, not a verb** (do not open with `Examine`/`Iterate`/`Use`). When the resource is keyed by a specific field, mention that field as the selection key with a concrete example. The description gets machine-parsed into the website-rendered resource docs, so favor depth and self-containment over single-line brevity. **Descriptions must not start with "Deprecated." or "Deprecated:" (enforced).** The only accepted leading phrases that contain the word "deprecated" are `Deprecated in favor of ...` and `Deprecated, please use ...`; any other variant must be rewritten so the deprecation notice comes after the resource/field summary.

```
// Apache2 HTTP Server
//
// Server configuration and daemon version, including loaded
// modules, virtual hosts, and directory directives.
apache2 { ... }
```

Selection-keyed example:

```
// Section of the Arista EOS running-config
//
// A single named section of the running-config, for when you need the
// raw text of one configuration block rather than the whole device. The
// `name` field selects the section as it appears in the running-config.
// For example `arista.eos.runningConfig.section(name: "interface Ethernet1")`
// or `... section(name: "router bgp 65001")`. The `content` field returns
// the raw text of that block.
arista.eos.runningConfig.section { ... }
```

**Style rules:**
- Don't reference the parent resource as a navigation hint ("read from the parent device as ...", "iterate from the parent ..."). Just describe what *this* resource examines.
- Don't use developer jargon ("Singleton", "Top-level entry point" is OK if it adds meaning, otherwise drop it). Write user-facing prose.
- Cross-reference sibling resources only when it genuinely helps the reader — e.g., pointing from a raw view (`apache2.conf`) at a richer typed view (`apache2.conf.module`).
- Private resources and pure sub-row types (rows of a parent collection like `*.entry`) typically don't need the two-part form — a single-line comment is fine.
- **No em dashes** anywhere in `.lr` files. Use a period, comma, parentheses, or colon.
- **Reference a field by its bare name** (`serviceAccount`), never the call form (`serviceAccount()`) — the `()` leaks the accessor mechanism.
- **Don't leak implementation mechanism or cross-cloud analogies.** No "typed"/"typed reference", "lazy-loaded"/"fetched on-demand", or "phase 1/phase 2" (describe by function, e.g. key-exchange/data-channel); no "the Azure equivalent of AWS Macie" style analogies. Describe the value on its own terms. (Per-discriminator dict shape lists — the key/value structure that varies by enum sibling — are schema, not mechanism; keep them.)

**Doc-comment shape — applies to both resources AND fields (enforced by the parser):**

A doc-comment is **either**:

1. **One line** — title only, no description. Use this for fields whose name already says it all (`// Account ID that owns the budget`) or short summaries that fit on one line.
2. **Two parts** — title, then a **single blank `//` separator**, then a multi-line description. Use this whenever you need more than a title (resources almost always; fields when you need to enumerate enum values, give examples, or explain non-obvious semantics).

**There is no third option.** Two contiguous comment lines with no blank `//` between them — the pattern you'd produce when wrapping a long one-liner across two source lines for readability — is **rejected at parse time**. The validator (`lrcore/lr.go: validateDocCommentStructure`) will fail the build with a precise location.

**Title length:** capped at **150 characters** (`lrcore.MaxTitleLength`, rune-counted, enforced at parse time). Descriptions have no cap. If a title doesn't fit, some of what you wrote belongs in the description — use the two-part form.

**Why:** the parser is positional — line 1 → `title`, everything after → `desc`. A wrapped one-liner produces a title truncated mid-clause and an orphaned description fragment; the blank `//` removes the ambiguity. The cap keeps titles usable in CLI tables, auto-complete prompts, and rendered docs.

```
// One-line summary, complete on its own.
foo string

// Title
//
// Multi-line description that can span as many lines as needed.
// The blank `//` above is mandatory whenever the comment has more
// than one line.
bar string
```

**Wrong** (will fail parse):
```
// Title that the author meant as one logical summary
// but accidentally wrapped onto a second line
baz string
```

Fix by either collapsing to one line or inserting a blank `//`. For enum lists, prefer the two-part form:
```
// Budget type
//
// One of COST, USAGE, RI_UTILIZATION, RI_COVERAGE, SAVINGS_PLANS_UTILIZATION, or SAVINGS_PLANS_COVERAGE.
budgetType string
```

### Step 1.5: Typed-reference gate (do this BEFORE you generate code)

**The single most expensive mistake to fix later is shipping a raw ID/URL string where a typed accessor belonged.** Once a `projectId string` field ships, replacing it with `project() openstack.project` is a *breaking change* (version bump + consumer migration) — getting it right first costs one extra accessor. So before codegen, **scan every new field and ask: does this value identify or point at another resource?**

If yes, it MUST be a typed accessor, not a raw string:

| You wrote (stop) | Ship this instead |
|---|---|
| `projectId string` | `project() <provider>.project` |
| `userId string` | `user() <provider>.user` |
| `vpcId string` | `vpc() aws.vpc` |
| `subnetIds []string` | `subnets() []…subnet` |
| `roleArn string` | `iamRole() aws.iam.role` |
| `networkUrl string` (GCP self-link) | `network() …network` |
| `stackId string` | `stack() …stack` |

Keep the value as a **raw string only when** it is the resource's own `id`, or a genuinely opaque/scalar value that doesn't name a modeled resource (an MD5 hash, an `hostId` opaque digest, a discriminator like `deviceOwner`, a free-form `name`). When in doubt, prefer the typed accessor — store the raw ID/ARN/URL in a `cache*` field on the `Internal` struct and resolve it with `NewResource` (or a shared `resolve*` helper). The detailed naming rules and the `cache*`/`Internal` pattern live in [§6 Resource Field Naming & Constraints](#resource-field-naming--constraints).

**Typed fields are the point of the schema, not a nicety.** They are what makes cross-resource querying possible — `neon.projects { endpoints { branch { protected } } }` or `netlify.dnsZones { site { forceSsl } }` are only expressible because the reference is a resource rather than an ID string. A schema of raw IDs forces the user to do the join by hand, which they cannot do inside a single policy. **Every new provider and every new resource gets this scan, without exception.**

**Model both directions when both are useful.** The gate is usually described as "child points at parent", but the reverse edge is often the one that carries the audit: `deployKey.sites` is what finds a key no site clones with; `project.defaultBranch` is what finds the branch applications actually connect to. If a reverse edge answers a question a policy would ask, add it.

**Resolve references through a cached list, not a per-item lookup.** A `NewResource` call runs the target's `init` **before** the runtime cache is consulted, so resolving a parent per child turns one list into N API calls. When the parent's collection is already fetched (or cheap to fetch once), resolve by scanning that list instead:

```go
// N+1: init runs per endpoint, before the cache is checked
NewResource(runtime, "neon.branch", map[string]*llx.RawData{"id": llx.StringData(branchID)})

// one call: the project's branch list is fetched once and reused
branchByID(runtime, projectID, branchID)   // walks project.GetBranches()
```

This is a manual gate, not a build-time one — there is no linter that catches it for you, so it is on you to run this scan on every new field before `mqlr generate`.

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

> **Only `NewResource` runs a resource's `init`; `CreateResource` skips it.** So an `__id` your `init` would have computed stays empty under `CreateResource`, and parameterized resources (e.g. an access-review / who-can lookup keyed on args) collide in the cache — every query returns the first one's result. Use `NewResource` for anything whose identity is built in `init`.

> **Read the target's `init` before writing a typed accessor.** It defines which arg keys are accepted (`arn`, `id`, or both) and they are **not** uniform: `initAwsEc2Instance` takes `arn` only, `initAwsVpc` takes `arn` or `id`. Passing the wrong key makes `NewResource` return an error — and because most typed accessors log-and-continue on a failed resolution, that surfaces as a **silently empty list**, not a failure. Verify the accessor returns something against real data, or the bug is invisible.

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

**Important:** If you add an Internal struct *after* the first code generation, you must run `./mqlr generate` a **second time** for the generator to detect and embed it. The same applies on **removal**: the stale embed lingers in `.lr.go` (build fails with `undefined: mql<Name>Internal`) until you regenerate.

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
  **Why?** The `command` resource handles execution context, auth, and connection handling across all connection types (local, SSH, container, etc.). See [lsblk.go](providers/os/resources/lsblk.go) for a full example.

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

- **Never let a singular resource's `init` function fall through with `return args, nil, nil` when its lookup found nothing.** Returning no resource and no error tells the runtime to create the resource from whatever partial `args` exist (often just an `arn`), leaving every other field **unset** — not null, unset. Any query touching those fields then crosses the plugin boundary as an empty `DataRes` and surfaces client-side as `llx: encountered a primitive with no type information, coercing to null`, with no attribution. Return a not-found error instead. The same applies to intermediate failures inside the init (e.g., an ARN that fails to parse): return the error, don't fall through to blank-resource creation.
  ```go
  // WRONG: no match found — runtime creates a blank resource with unset fields
  for _, r := range apis.Data {
      if match(r) { return args, r, nil }
  }
  return args, nil, nil

  // CORRECT: report what wasn't found
  return nil, nil, fmt.Errorf("aws.apigatewayv2.api with arn %q not found", wantArn)
  ```
  (`return args, nil, nil` is only correct for the early "args are already complete" fast path, e.g. `if len(args) > 2`.)

- **Resolve discovered assets by ARN, never by asset name.** `getAssetIdentifier` (AWS) returns the asset's validated resource ARN, or `""` when the asset has no parseable `arn:aws:` platform ID (e.g. an account asset). Only inject a non-empty result — an empty `args["arn"]` defeats the init's later `args["arn"] == nil` guard:
  ```go
  if assetArn := getAssetIdentifier(runtime); assetArn != "" {
      args["arn"] = llx.StringData(assetArn)
  }
  ```
  The asset name is a display name (often a `Name` tag), never a resource key — don't use it for lookups. The one exception: when an init's underlying API call is name-driven (e.g. IAM `GetUser`/`GetGroup`) and discovery sets the asset name to the resource's own name, use `getAssetName(runtime)` to inject `args["name"]` instead.

### Step 3.5: Discovery-filter gate (do this whenever you add a discovery target)

**If a resource becomes a discovery target, its lister must consult `conn.Filters`.** Wiring the `case Discovery<Thing>:` branch in `discovery.go` and wiring the filters are separate edits in separate files, and only the first one is what the ticket asks for — so the second is the one that gets forgotten. Adding a discovery target without filter support ships a service that accepts `--filters tag:...` and silently ignores it.

Apply filters in the **lister** (`get<Thing>`), never in `discovery.go`. Discovery reaches assets through the same `GetQueues()`/`GetTopics()` accessor a plain MQL query uses, so one check covers both paths and keeps them consistent.

```go
// gate the tag lookup so it stays lazy when no tag filter is set
if conn.Filters.General.HasTags() {
    tags := /* fetch tags for this resource */
    if conn.Filters.General.IsFilteredOutByTags(tags) {
        continue
    }
}
```

- **Cheap shape** (tags already in hand, or one call): copy `aws_s3.go`.
- **No batch tags endpoint** (one call per resource): use `fetchTagsConcurrently` in `aws.go` — bounded concurrency, per-item failures tolerated. Seed the fetched tags onto the resource so discovery's immediate read doesn't re-fetch, but **only for a tag set you actually read**; publishing an empty map for a resource whose tag call failed reports "no tags" as fact. Use the comma-ok form on the result map: present means read, absent means failed.
- **Region filters need nothing** — `conn.Regions()` already applies them.
- Services with a service-specific filter struct (`conn.Filters.Ecr`, `conn.Filters.S3`) consult that instead; that's correct.

This is a manual gate, not a build-time one. Because the bug is an *absence*, you can't spot it by reading — grep for listers that never mention `conn.Filters` and intersect with the discovery targets:

```bash
grep -rL "conn\.Filters" providers/<provider>/resources/*.go | grep -v _test
grep -n "case Discovery" providers/<provider>/resources/discovery.go
```

### Step 3.6: Pure-Go unit-test gate (do this before you open the PR)

**Every PR ships unit tests for the pure Go logic it adds.** Not the thin wrappers around an API call — those are covered by interactive verification — but everything in the change that computes, parses, or decodes. That is the code where a bug is silent: it compiles, it lints, and it produces a confident wrong answer instead of an error.

Test these:

- **Struct-tag decoding of every API record.** A mistyped `json:"..."` tag yields a zero value, so `blockPublicConnections` reads `false` on a project that blocks them and `passwordProtected` reads `false` on a protected site. Pin each security-relevant tag against a payload shaped like the documented response.
- **Optional-value handling.** An absent pointer must decode to the safe reading (`boolPtr(nil) == false`), and an absent timestamp must stay **null** rather than becoming the zero time, which would report 1 January year 1 as a real date.
- **Derived predicates.** Anything computed rather than reported — `passwordProtected` from whether a password came back, an `isPublic` built from several fields — gets its own table test including the empty and absent cases.
- **Pagination walks and their termination guards.** Use `httptest` and assert both the happy path and the stuck-cursor / repeated-page case. An endpoint that ignores its cursor otherwise multiplies every record up to the page cap.
- **Error classifiers.** `IsForbidden`/`IsNotFound` must match the API's shapes and must **not** match a transport error, or a network blip degrades to a null field and an audit passes on data that was never read.
- **Parsers and helpers.** Config parsers, ID builders, filter predicates, anything with a branch in it.

Skip: functions whose whole body is one SDK call plus a `CreateResource`. There is nothing to assert that the API contract does not already assert.

Put decode tests in `resources/decode_test.go` and client tests in `connection/client_test.go`, following `providers/vercel`. Run `go test ./...` inside `providers/<name>/` before opening the PR.

### Step 4: Verification (Interactive)
Automated tests are rare for MQL resources (thin wrappers). **Interactive testing is standard**, and it complements rather than replaces the unit tests in Step 3.6.

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

### Provider Version Updates
```bash
alias version="go run providers-sdk/v1/util/version/version.go"
version check providers/*/                              # check which need updates
version update providers/*/                              # interactive update
version update providers/*/ --increment=patch --commit   # auto-increment and commit
```

### Go Workspaces for Multi-Repo Development
When developing mql alongside cnspec, create a `go.work` in a parent directory with `use (./mql, ./mql/providers/aws, ./cnspec)` etc.

### Tips
*   **MCP Tools**: Use the GitHub MCP for tickets/PRs, and the Notion MCP for company-wide internal docs (Engineering, infra, dev env, etc.).
*   **Auth**: The environment usually has AWS/Azure CLI tools authenticated. If they're not present or logged in, stop and let me know so I can set up the provider's needs (tools, auth, whatever).
*   **Tickets**: If the ticket body contains queries to run in mql, make use of them during exploration/dev/testing/verification.
*   **Provider READMEs**: Many providers have detailed READMEs (auth methods, prerequisites, usage examples, troubleshooting). Always check `providers/<provider-name>/README.md` when working with a provider.

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
For a provider that must run on a specific VM (e.g. GCP snapshot scanning): configure it as builtin locally (above), `rsync` the source to the VM, install Go + Delve there, open the debugger port in the firewall, then `dlv debug apps/mql/mql.go --headless --listen=:12345 -- run <provider> ...`.

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

### Query Execution & Provider System
- **Flow:** MQL string → `mqlc.Compile()` → `llx.CodeBundle` (protobuf bytecode + metadata) → wrapped in `explorer.ExecutionQuery` → run by `executor.Executor` against the runtime → `llx.RawResult`. Layering: **MQL** (`mql/`, high-level executor, SQL/GraphQL-like) → **MQLC** (`mqlc/`, compiles text to bytecode) → **LLX** (`llx/`, the bytecode VM).
- **Providers** are gRPC plugins (hashicorp/go-plugin), each its own Go module; the **core provider** is always built-in (`asset`, `time`, `regex`). `providers.Coordinator` spawns each as a subprocess; each implements `ParseCLI()`/`Connect()`/`GetData()`/`StoreData()`/`Disconnect()`; `providers.Runtime` manages them per asset, and they can discover child assets (e.g. K8s → pods).
- **Field resolution:** compiler requests a field → `provider.GetData(connection, resource, field, args)` → backend fetch (cloud API, SSH, …) → `llx.Primitive` → `llx.RawData` → cached in the executor.

### Resources and Code Generation
Resources are defined in `.lr` files (e.g., `aws.lr`, `k8s.lr`); the `lr` tool generates Go resource structs, schema definitions, and data accessors from them. Generated files: `*.lr.go`, `*.lr.versions`, `*.resources.json`, `*.permissions.json`.

### Resource Caching & __id
**How caching works:**
- Each resource instance has a unique cache key: `resourceName + "\x00" + __id`
- Example: `"aws.ec2.instance\x00arn:aws:ec2:us-east-1:123456789012:instance/i-1234567890abcdef0"`
- The runtime checks cache before fetching: `if x, ok := runtime.Resources.Get(id); ok { return x, nil }`
- Results are cached automatically after first fetch

**Why `__id` matters:** it prevents redundant API calls and enables resource sharing across queries. It must be unique and stable (ARN, UUID, or composite key); if empty or duplicated, caching breaks and performance degrades.

**Hide synthetic `__id` values; don't expose them as `id` fields.** When a sub-resource's cache key is purely internal — a parent-qualified path like `<parentId>/confidentialCompute`, not something a user would ever query by — do NOT declare `id string` in the `.lr` block. Pass the cache key directly via the magic `"__id"` argument to `CreateResource`, and omit the `id()` Go method:

```go
// schema (.lr): no `id string` field declared
private gcp.project.computeService.instance.confidentialCompute @defaults("enabled instanceType") {
  enabled bool
  instanceType string
}

// creator (.go): pass "__id" directly, no id() func needed
mqlConfCompute, err := CreateResource(runtime, "gcp.project.computeService.instance.confidentialCompute", map[string]*llx.RawData{
    "__id":         llx.StringData(fmt.Sprintf("%d/confidentialCompute", instance.Id)),
    "enabled":      llx.BoolData(...),
    "instanceType": llx.StringData(...),
})
```

Reserve a public `id string` field for resources whose id carries user-meaningful information (an ARN, a GCP resource name, a stable cross-system key) — somewhere a user might write `.where(id == "...")`. Sub-resources whose id is `<parent>/<leaf>` synthetic should hide it. See `gcp.project.binaryAuthorizationControl.policy` for an existing example of this pattern. An existing sibling that exposes such an `id` (e.g. `aws.vpc.routetable.route`) is a pre-existing deviation, not precedent — follow the rule for new resources.

**Performance notes:** Resource field access is lazy (fields fetched only when needed); cross-references should leverage caching to avoid redundant API calls; use `init` functions for expensive operations to enable result sharing across queries.

### Code Generation Dependencies
Three codegen steps feed the build: protobuf (`.proto` → `.pb.go`), resources (`.lr` → `.lr.go`), and provider config (`providers.yaml` → `builtin_dev.go`). Run `make mql/generate` after modifying any of these.

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
- **A `panic` in a builtin or comparator crashes the entire scan.** The executor runs blocks in goroutines, so the panic is unrecoverable — it kills the scan, not just one query. Guard every type assertion on `arg.Value`/`bind.Value` with comma-ok (`x, ok := v.([]any)`) or a `== nil` check; never use the bare `v.(T)` form on runtime data.
- **`null && null` evaluates to `true` in MQL** (three-valued logic). An init-miss stub that leaves its booleans as `StateIsNull` will pass a `{ a && b }` assertion even though nothing was actually verified — set explicit `false` values on such stubs so misses and typos fail.

## 6. Important Implementation Details

### Copyright & License Headers
Every source file (`.go`, `.lr`, `.proto`, etc.) must begin with:
```
// Copyright Mondoo, Inc. 2024, <current year>
// SPDX-License-Identifier: BUSL-1.1
```
The format is `2024, <current year>` (comma-separated, not a dash). The first year (2024) is fixed; the second year should be the current calendar year. This is enforced by the `copywrite` tool installed via `make prep/tools`.

### Provider Connection Management
- Implement the full lifecycle: `Connect()`, `GetData()`, `StoreData()`, `Disconnect()`
- Handle auth failures gracefully with `Is400AccessDeniedError()` checks
- Use connection pooling where possible; add timeout handling for long-running API calls

### Error Handling Patterns
- Use `Is400AccessDeniedError(err)` for permission issues (returns `nil` result, not error)
- Return actual errors for temporary failures (rate limits, network issues)
- Log warnings for region/permission issues but continue with accessible resources
- Avoid failing entire queries due to single resource access issues
- `convert.SliceStrPtrToStr` and `convert.SliceStrPtrToInterface` deref each pointer with no nil guard — they **panic** on a slice containing nil elements. Write a nil-safe loop when the SDK slice may hold nil pointers (they are not drop-in "safe" replacements)

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
- **Always use typed resource references over raw ID/ARN strings.** This is the [Step 1.5 typed-reference gate](#step-15-typed-reference-gate-do-this-before-you-generate-code) you already ran — see its table for the mappings (`vpcId`→`vpc()`, `roleArn`→`iamRole()`, `topicArn`→`topic()`, etc.). They enable MQL traversal (e.g. `aws.rds.proxy.vpc.cidrBlock`); use `iamRole` not `role` to avoid ambiguity. Store the raw ID/ARN in a `cache*` field on the Internal struct, then implement the accessor with `NewResource`. Getting this wrong after ship is a breaking change.
  - **Name accessors `<thing>Ref` / `<things>Refs`** (or a domain word), never `<thing>Typed`.
  - **Deprecate the raw `*Id`/`*Arn` field when a typed ref carries the same value** — it's a useless duplicate. Mark it `@maturity("deprecated")` and give the ref the clean name. Keep the raw field only when it's a list of tokens the ref can't round-trip (then name the accessor `*Refs`).
- **GCP: Use typed resource references over raw URL strings.** GCP Compute resources often store references as self-link URLs (e.g., `https://www.googleapis.com/compute/v1/projects/.../networks/my-net`). Always add a typed computed method alongside the raw URL field — e.g. `networkUrl` → `network()`, and likewise `subnetworkUrl` → `subnetwork()`, `routerUrl`, `sslPolicyUrl`, `securityPolicyUrl`, `interconnectUrl`, `vpnGatewayUrl` (all resolving to `gcp.project.computeService.*`).
- **When to create a sub-resource.** A sub-resource (`aws.foo.bar`) is appropriate only when it meets **one of two bars**:
  1. **Clear ID.** It has a stable unique identifier — an ARN, a name-with-region, or another natural key returned by the API. Synthetic composite IDs like `<parentArn>/leaf` do **not** count.
  2. **Nested typed reference.** It exists to hold typed refs to other modeled resources (e.g. a `computeEnvironmentOrder` sub-resource whose reason to exist is `computeEnvironment() aws.batch.computeEnvironment`).

  If neither applies, **do not create a sub-resource**. Instead:
  - **Flatten scalars into the parent** with a disambiguating prefix. Prefer `linuxMaxSwap int`, `linuxSharedMemorySize int`, `fargatePlatformVersion string` over a single-scalar `linuxParameters`/`fargatePlatformConfiguration` sub-resource.
  - **Use `map[string]string`** for name/value pair lists that would otherwise be a struct with two string fields (e.g. `environmentVariables map[string]string` rather than a `{name, value}` sub-resource — keys of `GPU`/`MEMORY`/`VCPU` work the same way for resource requirements).
  - **Use `[]dict`** for small heterogeneous structs (e.g. Linux devices, tmpfs mounts, evaluate-on-exit conditions) where no individual field warrants its own typed audit query.

  **Why:** A sub-resource has real cost — a generated struct, `__id` stability, serialization, test surface, and a `.lr.versions` entry per field. Creating them for pure data containers bloats the schema without new query power; scalars and maps on the parent are already queryable (e.g. `resourceRequirements["GPU"]`).

  **Examples (Batch):**
  - ✓ `aws.batch.jobQueue.computeEnvironmentOrder` — nests typed `computeEnvironment()` ref
  - ✓ `aws.batch.jobDefinition.containerProperties.secret` — `valueFrom` is the ARN (and will eventually resolve to a typed ref)
  - ✗ `{name, value}` env var — use `map[string]string`
  - ✗ `{type, value}` resource requirement — use `map[string]string` keyed by type
  - ✗ `{platformVersion}` single-scalar config — flatten to `fargatePlatformVersion` on parent
  - ✗ `{hostPath, containerPath, permissions}` device / `{containerPath, size, mountOptions}` tmpfs — use `[]dict`
- Every resource and field has an explicit entry in `.lr.versions`. New entries must use the **next patch version** after the provider's current version (e.g., if the provider is at `13.1.1`, new fields should be `13.1.2`). The provider's current version is in `providers/<name>/config/config.go` (look for the `Version` field). Do **not** rely on the highest version in `.lr.versions` — it may be stale from before a major version bump. The `versions` command does this automatically, but verify the result. Existing entries are never overwritten — which also means **removing a field does not remove its entry**; delete that line by hand.
  - **Exception — brand-new, unreleased provider:** if the provider is being introduced in this PR (its `config.go` `Version` hasn't shipped yet), every `.lr.versions` entry is part of the initial release and equals that version — not `version + 1`. The "next patch" rule applies only to fields added *after* a version has shipped (a new provider at `13.0.0` has all entries at `13.0.0`; a later PR adding a field bumps that one to `13.0.1`).
  - **Don't bump the provider's `Version` in `config.go` in a feature PR** — the release flow handles that separately. You only add the `.lr.versions` entries.
- **`isPublic()` means "reachable from the internet."** It carries that meaning on ~20 resources, so a fleet-wide `isPublic` query has to stay comparable across them. Don't reuse the name for a different kind of "open" (an unrestricted resource policy, a wildcard principal) on a resource that is not internet-facing — pick a specific name like `hasWildcardPolicy()` instead.
- **Match SDK types faithfully:** If an SDK field is `*bool`, use `bool` in `.lr` and `llx.BoolDataPtr()` in Go — don't cast it to `string`. If an SDK enum has only two states (Enabled/Disabled), prefer `bool`. Use `*type` intermediate variables with `llx.*DataPtr` helpers to preserve nil semantics.
- **Never change a shipped field's type** (e.g. `string`→`bool`) — it's customer-breaking. Deprecate the old field and add a new one instead; decline review-bot suggestions to mutate an existing field's type in place.
- **Consistency with existing fields:** Before adding new fields to a resource, check how its existing fields handle pointers, nil checks, and type conversions. Follow the same pattern.
- **Verify enum values in `.lr` comments:** When listing possible values in field comments, check the SDK/API docs for completeness — don't assume the set is closed.
- **Skip deprecated SDK fields and methods.** Check the SDK's `// Deprecated:` comment before exposing a field or calling a method. Deprecated fields often return empty/zero on modern instances because the data moved elsewhere (e.g. GCP Memorystore moved `DiscoveryEndpoints`/`PscAutoConnections` into `Endpoints`) — modeling them adds dead schema. Same for deprecated `Get*`/`List*` methods; pick the replacement. If you genuinely need one for backward-compat, leave a comment explaining why.
- **Deprecating fields and resources — use `@maturity`.** When a field or resource is kept for backward-compat but should not be used by new audits, mark it `@maturity("deprecated")` in the `.lr` schema. The title stays a plain noun phrase (titles starting with "deprecated" are rejected); the deprecation notice and replacement go in the description, which must lead with `Deprecated in favor of ...` or `Deprecated, please use ...` (`Deprecated.` / `Deprecated:` are rejected). Also valid: `@maturity("preview")` for fields whose shape may still change. Examples:
  ```
  // Legacy endpoint dict
  //
  // Deprecated in favor of endpointAddress, endpointPort, and
  // endpointHostedZoneId.
  endpoint @maturity("deprecated") dict

  // Cloudflare zone plan
  //
  // The Cloudflare service plan attached to a zone. Will be
  // removed in the next major release.
  private cloudflare.zone.plan @defaults("name") @maturity("deprecated") { ... }
  ```
  Keep the existing `.lr.versions` entry at its original version — deprecation does not bump the version.

### Provider Modules & Dependencies
- Each provider in `providers/` has its own `go.mod` for isolation, keeping binaries smaller and dependency trees separate (core mql has deps providers don't need, and vice versa)

### Built-in vs External Providers
- Core provider is always compiled into mql (provides universal resources)
- Others default to external plugins (separate binaries loaded at runtime via gRPC); for debugging they can be made built-in via `providers.yaml` — requires cleanup before commits
- For debugging, use a debugger mcp if available, so you set breakpoints instead of stdout debugging.

### Code Generation Gotchas
- Never manually edit generated `.lr.go` files - they get overwritten
- Use `make providers/mqlr` for faster provider-specific regeneration (see [Step 3 Pattern D](#step-3-implementation-strategies) for the Internal-struct second-pass gotcha)
- **A resource named `<parent>.<field>` queried by its dotted path becomes an empty husk.** Fingerprint: `cannot convert primitive with NO type information`. Don't give a resource the same name as the dotted path used to reach it.
- **A list field whose path equals its element resource type compiles as a single resource, not a list.** Fingerprint: `is not a list type`. Use a plural field name over a singular element resource (field `policies` → resource `...policy`).
- **Always commit `*.permissions.json`** when it changes. This file is regenerated by `make providers/build/<provider>` and tracks the IAM permissions each provider requires. Changes to it (e.g., new API calls added, deprecated calls removed) are part of the PR — they're not throwaway build artifacts.
  - A new SDK client's `Get*` permissions only reach `aws.permissions.json` if the client is mapped in `awsConnectionMethodToService` (add an `awsServiceNameOverrides` entry too when the IAM prefix ≠ SDK package name). For GCP, a `Get<Resource>` gRPC method singularizes to a non-existent IAM permission — add a `gcpPermissionOverrides` entry for the correct plural form. Miss either and the perms silently drop from the manifest (GCP surfaces only as a CI-side `TestGCPPermissionsMatchValidatedList` failure).

### Testing & Verification
(Build/install vs. builtin `go run` is covered in [Step 4: Verification](#step-4-verification-interactive).)
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
See [DEVELOPMENT.md → Creating a new provider](DEVELOPMENT.md#creating-a-new-provider) for the scaffolding command and the four files to register the new provider in.

### CLI and Output
Key directories for user-facing functionality:
- **`apps/mql/cmd/`** - CLI command implementations
- **`cli/shell/`** - Interactive shell with auto-completion
- **`cli/reporter/`** - Output formatting (JSON, CSV, YAML, table, etc.)

**Always use the codebase's patterns.**

## 7. Pre-PR Checklist

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
- [ ] `go.mod` is clean — run `go mod tidy` **inside `providers/<name>/`**, not the repo root, or new SDK deps stay `// indirect`
- [ ] Every new field that names another resource is a typed accessor ([Step 1.5](#step-15-typed-reference-gate-do-this-before-you-generate-code))
- [ ] Pure Go logic added by the change has unit tests ([Step 3.6](#step-36-pure-go-unit-test-gate-do-this-before-you-open-the-pr))
- [ ] No spelling errors in new comments/docs

**Note:** CI runs comprehensive checks. Run them locally only if you want to verify before pushing or if changing core/performance-critical code.

### Spell Check CI
CI runs [`crate-ci/typos`](https://github.com/crate-ci/typos) on every PR (config: `_typos.toml` at the repo root). Fix genuine typos; add false positives (identifiers, product names) to `_typos.toml`.

## 8. Commit Conventions

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

**No Claude session IDs in git.** Keep the Claude Code session identifier out of the repo and the GitHub UI: commit messages must not carry a `Claude-Session:` trailer or a `claude.ai/code/session_…` URL, and PR bodies must not include the session URL. This overrides any harness/environment default that would append them. (The `Co-Authored-By: Claude …` trailer is fine — this rule is only about the session identifier.)

Anticipate needs, offer options when it applies, think in the context of ticket-solution-in-codebase.
