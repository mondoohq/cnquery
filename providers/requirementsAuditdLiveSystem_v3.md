# Technical Requirements Document: Extend `auditd.rules` Resource with Live Runtime Support
## Architecture Version 3.0 - K8s Provider Pattern

## Document Purpose
This document specifies requirements for extending the `auditd.rules` resource to support querying both filesystem-based audit rules AND live runtime audit rules from the Linux kernel when running on live systems.

**Architecture Approach**: Connection-level data source management (inspired by K8s provider patterns)  
**Target Audience**: LLM agents and developers implementing this feature  
**Status**: Revised architecture - ready for implementation  
**Previous Version**: v2.0 (resource-level source parameter approach)

---

## 1. Background & Context

### Current Behavior
The `auditd.rules` resource currently:
- Reads audit rule configuration files from `/etc/audit/rules.d/` (default)
- Parses `.rules` files to extract three rule types: controls, files, syscalls
- Works on both live and non-live systems (images, snapshots, containers)
- Only reflects what will be loaded at boot, not current runtime state

### Gap
On live systems, the actual active audit rules in the kernel may differ from filesystem configurations due to:
- Runtime modifications via `auditctl` commands
- Temporary rules added by administrators
- Failed rule loads during boot
- Manual rule deletions

### Key Insight from K8s Provider
The K8s provider demonstrates a clean pattern for integrating multiple data sources:
- **Connection abstraction**: Different connection types (API, Manifest, Admission) implement a unified interface
- **Resource simplicity**: Resources don't know about data sources - they just query the connection
- **Single responsibility**: Connection handles data fetching; resource handles data presentation
- **Transparent behavior**: User connects to different sources without changing queries

---

## 2. Objectives

### Primary Goal
Extend the OS connection to provide both filesystem and runtime audit rule data when running on live systems, using connection-level abstraction similar to K8s provider patterns.

### Success Criteria
1. ✅ Existing MQL queries continue to work unchanged with same syntax
2. ✅ Automatic capability-based behavior (no user code changes required)
3. ✅ Connection-level data source management (not resource-level)
4. ✅ Clear FAILED states identify source of failures (filesystem, runtime, or both)
5. ✅ All current key features preserved (see Section 6)
6. ✅ No performance degradation on non-live systems
7. ✅ Logical AND behavior when both sources available
8. ✅ Architecture aligned with existing cnquery connection patterns

---

## 3. Architecture Overview

### 3.1: Connection-Level Data Source Pattern

Following the K8s provider model where different connection types handle different data sources:

```
┌─────────────────────────────────────────────────────────────┐
│                    OS Provider Service                       │
│  - Manages connection lifecycle                             │
│  - Determines live vs non-live capabilities                 │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    OS Connection                             │
│  - Has capabilities: [run-command, filesystem, ...]         │
│  - Provides unified data interface                          │
│  - Handles dual-source logic internally                     │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Audit Rules Data Provider                       │
│  - Capability detection                                      │
│  - Filesystem rule loading                                  │
│  - Runtime rule loading (if capable)                        │
│  - Logical AND evaluation                                   │
│  - Error aggregation                                        │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               auditd.rules Resource                          │
│  - Simple data accessor                                      │
│  - No knowledge of data sources                             │
│  - Queries connection for rule data                         │
│  - Presents unified view to user                            │
└─────────────────────────────────────────────────────────────┘
```

### 3.2: Key Architectural Principles

1. **Separation of Concerns**
   - Connection: "How to get data"
   - Resource: "How to present data"
   
2. **Single Source of Truth**
   - Connection capabilities determine behavior
   - Resource trusts connection's data
   
3. **No Resource-Level Configuration**
   - No `source` parameter on resource
   - Behavior determined by connection type/capabilities
   
4. **Transparent Enhancement**
   - Live systems automatically get dual-source behavior
   - Non-live systems get filesystem-only behavior
   - No query syntax changes needed

---

## 4. Functional Requirements

### FR-1: Connection Capability Detection
**Requirement**: OS connection automatically detects if the system supports live rule querying.

**Implementation**:
```go
// In OS connection initialization
type Connection struct {
    capabilities []Capability
    auditRuleProvider *AuditRuleProvider
}

func (c *Connection) initAuditRuleProvider() {
    hasRunCommand := c.capabilities.Has(Capability_RunCommand)
    c.auditRuleProvider = &AuditRuleProvider{
        connection: c,
        useRuntime: hasRunCommand,
    }
}
```

**Behavior**:
- `run-command` capability present → Enable dual-source mode
- `run-command` capability absent → Filesystem-only mode
- Happens transparently during connection initialization

---

### FR-2: Connection-Level Audit Rule Provider
**Requirement**: Create a connection-level provider that abstracts audit rule data sources.

**Interface Design**:
```go
// New: Audit rule data provider (connection-level)
type AuditRuleProvider struct {
    connection    *Connection
    useRuntime    bool
    
    // Lazy loading with dual storage
    filesystemOnce sync.Once
    filesystemData *AuditRuleData
    filesystemErr  error
    
    runtimeOnce    sync.Once
    runtimeData    *AuditRuleData
    runtimeErr     error
}

type AuditRuleData struct {
    Controls []interface{}
    Files    []interface{}
    Syscalls []interface{}
}

// Main method that resources call
func (p *AuditRuleProvider) GetRules(path string) (*AuditRuleData, error) {
    if !p.useRuntime {
        return p.getFilesystemRules(path)
    }
    return p.getBothRules(path)
}
```

**Key Points**:
- Lives on the connection, not the resource
- Encapsulates all dual-source logic
- Resources simply call `connection.GetAuditRules(path)`
- Similar to how K8s resources call `connection.Resources(kind, name, namespace)`

---

### FR-3: Runtime Rule Collection
**Requirement**: When on a live system, execute `auditctl -l` to gather active kernel audit rules.

**Command**: `auditctl -l`

**Implementation Pattern** (following K8s Discovery pattern):
```go
func (p *AuditRuleProvider) loadRuntimeRules() (*AuditRuleData, error) {
    if !p.useRuntime {
        return nil, nil
    }
    
    cmd := p.connection.RunCommand("auditctl", "-l")
    if cmd.ExitStatus != 0 {
        return nil, fmt.Errorf("Failed to load audit rules from runtime: %s", 
            cmd.Stderr)
    }
    
    // Reuse existing parser
    data := &AuditRuleData{}
    parse(cmd.Stdout, &data.Controls, &data.Files, &data.Syscalls, &errors)
    
    if len(errors) > 0 {
        return nil, fmt.Errorf("Failed to parse runtime rules: %v", errors)
    }
    
    return data, nil
}
```

---

### FR-4: Logical AND Evaluation at Connection Level
**Requirement**: When both sources are available, rules must pass checks in BOTH sources.

**Connection-Level Logic**:
```go
func (p *AuditRuleProvider) getBothRules(path string) (*AuditRuleData, error) {
    // Load both sources in parallel
    var wg sync.WaitGroup
    var fsData, rtData *AuditRuleData
    var fsErr, rtErr error
    
    wg.Add(2)
    go func() {
        defer wg.Done()
        fsData, fsErr = p.getFilesystemRules(path)
    }()
    go func() {
        defer wg.Done()
        rtData, rtErr = p.loadRuntimeRules()
    }()
    wg.Wait()
    
    // Logical AND: both must succeed
    if fsErr != nil && rtErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from filesystem and runtime: [filesystem: %v, runtime: %v]", fsErr, rtErr)
    }
    if fsErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from filesystem: %v", fsErr)
    }
    if rtErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from runtime: %v", rtErr)
    }
    
    // Validate sets match (order-agnostic comparison)
    return p.validateAndMerge(fsData, rtData)
}
```

**Evaluation Matrix**:

| Capability | Filesystem | Runtime | Result |
|-----------|-----------|---------|--------|
| Live | ✅ Pass | ✅ Pass | ✅ PASS |
| Live | ✅ Pass | ❌ Fail | ❌ FAILED (runtime) |
| Live | ❌ Fail | ✅ Pass | ❌ FAILED (filesystem) |
| Live | ❌ Fail | ❌ Fail | ❌ FAILED (both) |
| Non-live | ✅ Pass | N/A | ✅ PASS (current behavior) |
| Non-live | ❌ Fail | N/A | ❌ FAILED (filesystem) |

---

### FR-5: Simplified Resource Implementation
**Requirement**: Resource becomes a simple data accessor, delegating to connection.

**Resource Pattern** (following K8s resource pattern):
```go
func (s *mqlAuditdRules) controls(path string) ([]any, error) {
    return s.getRulesByType(path, "controls")
}

func (s *mqlAuditdRules) files(path string) ([]any, error) {
    return s.getRulesByType(path, "files")
}

func (s *mqlAuditdRules) syscalls(path string) ([]any, error) {
    return s.getRulesByType(path, "syscalls")
}

// Internal helper
func (s *mqlAuditdRules) getRulesByType(path string, ruleType string) ([]any, error) {
    // Get connection's audit rule provider
    conn := s.MqlRuntime.Connection.(shared.Connection)
    provider := conn.AuditRuleProvider()
    
    // Let connection handle all data source logic
    data, err := provider.GetRules(path)
    if err != nil {
        return nil, err
    }
    
    // Return appropriate rule type
    switch ruleType {
    case "controls":
        return data.Controls, nil
    case "files":
        return data.Files, nil
    case "syscalls":
        return data.Syscalls, nil
    }
    
    return nil, fmt.Errorf("unknown rule type: %s", ruleType)
}
```

**Schema** (unchanged from current):
```
auditd.rules {
  init(path? string)
  path() string
  controls(path string) []auditd.rule.control
  files(path string) []auditd.rule.file
  syscalls(path string) []auditd.rule.syscall
}
```

**Key Difference from v2.0**:
- ❌ No `source` parameter
- ✅ Connection determines behavior
- ✅ Resource just presents data

---

### FR-6: Investigation and Debugging Support
**Requirement**: Enable users to investigate discrepancies between filesystem and runtime.

**Approach**: Use connection options (similar to K8s context flag)

**Option A - Connection-Level Flag** (Recommended):
```bash
# Default: automatic dual-source on live systems
cnquery shell os

# Force filesystem-only (even on live systems)
cnquery shell os --audit-source=filesystem

# Force runtime-only (fails on non-live systems)
cnquery shell os --audit-source=runtime
```

**Option B - Provider Configuration**:
```go
// In provider connection setup
func (s *Service) ParseCLI(req *plugin.ParseCLIReq) (*plugin.ParseCLIRes, error) {
    conf := &inventory.Config{
        Options: map[string]string{},
    }
    
    if auditSource, ok := req.Flags["audit-source"]; ok {
        conf.Options["audit-source"] = string(auditSource.Value)
    }
    
    return &plugin.ParseCLIRes{Asset: asset}, nil
}
```

**Rationale**:
- Aligns with K8s provider pattern (connection-level configuration)
- Clean separation: connection configuration vs query syntax
- Enables troubleshooting without changing queries
- Similar to `--context` flag in K8s provider

---

### FR-7: Error Handling & FAILED States
**Requirement**: Return FAILED states (not errors) with clear, actionable messages identifying failure source.

**Connection-Level Error Handling**:
```go
func (p *AuditRuleProvider) GetRules(path string) (*AuditRuleData, error) {
    // Errors become FAILED states
    if err != nil {
        // Include source information in error message
        return nil, err  // Connection converts to FAILED state
    }
    return data, nil
}
```

**Error Scenarios**:

| Scenario | Filesystem | Runtime | Message |
|----------|-----------|---------|---------|
| A | ✅ Success | ✅ Success | No message |
| B | ❌ Failed | ✅ Success | "Failed to load audit rules from filesystem: [details]" |
| C | ✅ Success | ❌ Failed | "Failed to load audit rules from runtime: [details]" |
| D | ❌ Failed | ❌ Failed | "Failed to load audit rules from both filesystem and runtime: [filesystem: X, runtime: Y]" |
| E | ✅ Success | N/A (non-live) | No message (current behavior) |

---

### FR-8: Key Features Preservation
**Requirement**: All existing key features MUST be maintained:

1. ✅ **Automatic categorization** of different rule types (control, file, syscall)
2. ✅ **Structured field parsing** for syscall filters
3. ✅ **Operator parsing** (=, !=, >=, <=, >, <) for field comparisons
4. ✅ **Multiple file support** - reads all `.rules` files in a directory
5. ✅ **Thread-safe** loading with mutex
6. ✅ **Error accumulation** - collects all parsing errors instead of failing fast
7. ✅ **Lazy evaluation** - rules are parsed only when accessed
8. ✅ **Set-based comparison** - rule order does not matter, only existence/non-existence

---

## 5. Non-Functional Requirements

### NFR-1: Performance
- Parallel loading of filesystem and runtime rules
- Lazy evaluation with once.Do() pattern
- Cache rule data per connection instance
- No performance impact when `run-command` capability is false
- No redundant command executions

### NFR-2: Backward Compatibility
- **100% backward compatible**: Existing MQL queries work unchanged
- No syntax changes required in existing policies
- Behavior enhancement happens transparently at connection level
- Existing tests must continue to pass with same syntax
- Resource IDs remain stable

### NFR-3: Security
- Handle privilege escalation failures gracefully (return FAILED state)
- Never execute arbitrary commands (only `auditctl -l`)
- Sanitize command output before parsing
- Bubble up OS security errors as-is (e.g., "You must be root")

### NFR-4: Maintainability
- **Connection-level abstraction** reduces resource complexity
- **Reuse existing parsing logic** - both sources use identical format
- **Clear separation of concerns** - connection handles data, resource presents it
- **Aligned with cnquery patterns** - follows K8s provider architecture

---

## 6. Design Decisions (FINALIZED)

### 6.1: Architecture Pattern ✅ DECISION: Connection-Level Data Provider

**Chosen Approach**: K8s-inspired connection abstraction pattern

**Design**:
```
Connection (OS)
  └── AuditRuleProvider
        ├── Capability detection
        ├── Filesystem loading
        ├── Runtime loading (if capable)
        └── Logical AND evaluation

Resource (auditd.rules)
  └── Simple accessor
        └── Calls connection.GetAuditRules()
```

**Rationale**:
- ✅ Aligned with existing cnquery/K8s provider patterns
- ✅ Clean separation of concerns
- ✅ Resource simplicity (no source parameter complexity)
- ✅ Testability (can mock connection provider)
- ✅ Extensibility (easy to add new sources in future)

**Key Difference from v2.0**:
- ❌ **v2.0**: Resource-level `source` parameter, dual-source logic in resource
- ✅ **v3.0**: Connection-level provider, resource is data accessor

---

### 6.2: Configuration Approach ✅ DECISION: Connection Options (Optional)

**Chosen Approach**: Connection-level flags for explicit source control (optional feature)

**Implementation**:
```bash
# Default behavior (automatic)
cnquery shell os
auditd.rules.files  # Dual-source on live, filesystem-only on non-live

# Explicit source selection (for troubleshooting)
cnquery shell os --audit-source=filesystem
cnquery shell os --audit-source=runtime
```

**Rationale**:
- Similar to K8s provider's `--context` flag
- Connection-level configuration, not query-level
- Optional: default behavior works for 99% of use cases
- Enables troubleshooting when needed

---

### 6.3: Discrepancy Handling ✅ DECISION: Logical AND with Set Comparison

**Chosen Approach**: Strict mode - both sources must match (set-based)

**Behavior**:
- Both sources available → Both must succeed
- Sets must match (order-agnostic)
- Any mismatch → FAILED state with details

**Comparison Logic**:
```go
func (p *AuditRuleProvider) validateAndMerge(fs, rt *AuditRuleData) (*AuditRuleData, error) {
    // Set-based comparison for each rule type
    if !rulesMatch(fs.Controls, rt.Controls) {
        return nil, fmt.Errorf("Control rules differ between filesystem and runtime")
    }
    if !rulesMatch(fs.Files, rt.Files) {
        return nil, fmt.Errorf("File rules differ between filesystem and runtime")
    }
    if !rulesMatch(fs.Syscalls, rt.Syscalls) {
        return nil, fmt.Errorf("Syscall rules differ between filesystem and runtime")
    }
    
    return fs, nil  // Return either (they match)
}

func rulesMatch(a, b []interface{}) bool {
    // Set-based comparison (order doesn't matter)
    setA := makeSet(a)
    setB := makeSet(b)
    return setA.Equals(setB)
}
```

---

### 6.4: Resource ID Calculation ✅ DECISION: Unchanged (Path-Based)

**Chosen Approach**: Keep existing ID logic

```go
func (s *mqlAuditdRules) id() (string, error) {
    return s.Path.Data, nil  // Unchanged
}
```

**Rationale**:
- Dual-source is connection-level implementation detail
- Resource represents audit configuration at a path
- Stable IDs preserve caching behavior

---

## 7. Implementation Guidance

### 7.1: File Structure

```
providers/os/
  connection/
    shared/
      shared.go              # Add AuditRuleProvider interface
      audit_provider.go      # New: AuditRuleProvider implementation
  resources/
    auditd.go               # Simplify: delegate to connection
    auditd_test.go          # Update: mock connection provider
    os.lr                   # Unchanged
```

### 7.2: Connection Integration

**Add to OS Connection**:
```go
// providers/os/connection/shared/shared.go
type Connection interface {
    plugin.Connection
    // ... existing methods ...
    AuditRuleProvider() *AuditRuleProvider  // New
}

// Implementation
type ConnectionImpl struct {
    // ... existing fields ...
    auditProvider *AuditRuleProvider
}

func (c *ConnectionImpl) AuditRuleProvider() *AuditRuleProvider {
    return c.auditProvider
}
```

### 7.3: Provider Implementation

**New file**: `providers/os/connection/shared/audit_provider.go`

```go
package shared

import (
    "fmt"
    "sync"
)

type AuditRuleProvider struct {
    connection *ConnectionImpl
    useRuntime bool
    
    // Filesystem rules
    filesystemOnce sync.Once
    filesystemData *AuditRuleData
    filesystemErr  error
    
    // Runtime rules
    runtimeOnce sync.Once
    runtimeData *AuditRuleData
    runtimeErr error
}

type AuditRuleData struct {
    Controls []interface{}
    Files    []interface{}
    Syscalls []interface{}
}

func NewAuditRuleProvider(conn *ConnectionImpl) *AuditRuleProvider {
    hasRunCommand := conn.Capabilities().Has(Capability_RunCommand)
    
    return &AuditRuleProvider{
        connection: conn,
        useRuntime: hasRunCommand,
    }
}

func (p *AuditRuleProvider) GetRules(path string) (*AuditRuleData, error) {
    if !p.useRuntime {
        return p.getFilesystemRules(path)
    }
    return p.getBothRules(path)
}

func (p *AuditRuleProvider) getFilesystemRules(path string) (*AuditRuleData, error) {
    p.filesystemOnce.Do(func() {
        p.filesystemData, p.filesystemErr = p.loadFilesystemRules(path)
    })
    return p.filesystemData, p.filesystemErr
}

func (p *AuditRuleProvider) loadFilesystemRules(path string) (*AuditRuleData, error) {
    // Existing filesystem loading logic
    // Read files from path, parse, return data
}

func (p *AuditRuleProvider) loadRuntimeRules() (*AuditRuleData, error) {
    // Execute auditctl -l
    cmd := p.connection.RunCommand("auditctl", "-l")
    if cmd.ExitStatus != 0 {
        return nil, fmt.Errorf("Failed to load audit rules from runtime: %s", cmd.Stderr)
    }
    
    // Parse using existing parser
    data := &AuditRuleData{}
    var errors []interface{}
    parse(cmd.Stdout, &data.Controls, &data.Files, &data.Syscalls, &errors)
    
    if len(errors) > 0 {
        return nil, fmt.Errorf("Failed to parse runtime rules: %v", errors)
    }
    
    return data, nil
}

func (p *AuditRuleProvider) getBothRules(path string) (*AuditRuleData, error) {
    // Load both in parallel
    var wg sync.WaitGroup
    var fsData, rtData *AuditRuleData
    var fsErr, rtErr error
    
    wg.Add(2)
    go func() {
        defer wg.Done()
        fsData, fsErr = p.getFilesystemRules(path)
    }()
    go func() {
        defer wg.Done()
        p.runtimeOnce.Do(func() {
            p.runtimeData, p.runtimeErr = p.loadRuntimeRules()
        })
        rtData, rtErr = p.runtimeData, p.runtimeErr
    }()
    wg.Wait()
    
    // Logical AND evaluation
    if fsErr != nil && rtErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from filesystem and runtime: [filesystem: %v, runtime: %v]", fsErr, rtErr)
    }
    if fsErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from filesystem: %v", fsErr)
    }
    if rtErr != nil {
        return nil, fmt.Errorf("Failed to load audit rules from runtime: %v", rtErr)
    }
    
    // Validate sets match
    return p.validateAndMerge(fsData, rtData)
}

func (p *AuditRuleProvider) validateAndMerge(fs, rt *AuditRuleData) (*AuditRuleData, error) {
    // Set-based comparison
    if !rulesMatch(fs.Controls, rt.Controls) {
        return nil, fmt.Errorf("Control rules differ between filesystem and runtime")
    }
    if !rulesMatch(fs.Files, rt.Files) {
        return nil, fmt.Errorf("File rules differ between filesystem and runtime")
    }
    if !rulesMatch(fs.Syscalls, rt.Syscalls) {
        return nil, fmt.Errorf("Syscall rules differ between filesystem and runtime")
    }
    
    return fs, nil
}

func rulesMatch(a, b []interface{}) bool {
    // Set-based comparison implementation
    // Convert to sets, compare
}
```

### 7.4: Simplified Resource Implementation

**Updated**: `providers/os/resources/auditd.go`

```go
package resources

// Simplified internal structure
type mqlAuditdRulesInternal struct {
    // Just caching, no dual-source complexity
}

func (s *mqlAuditdRules) controls(path string) ([]any, error) {
    data, err := s.getAuditRuleData(path)
    if err != nil {
        return nil, err
    }
    return data.Controls, nil
}

func (s *mqlAuditdRules) files(path string) ([]any, error) {
    data, err := s.getAuditRuleData(path)
    if err != nil {
        return nil, err
    }
    return data.Files, nil
}

func (s *mqlAuditdRules) syscalls(path string) ([]any, error) {
    data, err := s.getAuditRuleData(path)
    if err != nil {
        return nil, err
    }
    return data.Syscalls, nil
}

// Internal helper - delegates to connection
func (s *mqlAuditdRules) getAuditRuleData(path string) (*shared.AuditRuleData, error) {
    conn := s.MqlRuntime.Connection.(shared.Connection)
    provider := conn.AuditRuleProvider()
    return provider.GetRules(path)
}
```

---

## 8. Testing Requirements

### Test Strategy

**Unit Tests** (Connection Provider):
```go
func TestAuditRuleProvider_FilesystemOnly(t *testing.T) {
    // Mock connection without run-command capability
    conn := mockConnection(capabilities: [])
    provider := NewAuditRuleProvider(conn)
    
    data, err := provider.GetRules("/etc/audit/rules.d")
    // Verify only filesystem loaded
}

func TestAuditRuleProvider_DualSource(t *testing.T) {
    // Mock connection with run-command capability
    conn := mockConnection(capabilities: [Capability_RunCommand])
    provider := NewAuditRuleProvider(conn)
    
    data, err := provider.GetRules("/etc/audit/rules.d")
    // Verify both sources loaded and matched
}

func TestAuditRuleProvider_RuntimeMismatch(t *testing.T) {
    // Mock: filesystem has rules, runtime different
    provider := mockProviderWithDifferentSources()
    
    data, err := provider.GetRules("/etc/audit/rules.d")
    // Verify FAILED state returned
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "differ between filesystem and runtime")
}
```

**Integration Tests** (Resource):
```go
func TestAuditdRules_LiveSystem(t *testing.T) {
    // Mock live connection
    runtime := mockRuntimeWithLiveConnection()
    
    rules := &mqlAuditdRules{MqlRuntime: runtime}
    files, err := rules.files("/etc/audit/rules.d")
    
    // Verify data returned (connection handles dual-source)
    assert.NoError(t, err)
    assert.NotEmpty(t, files)
}
```

### Test Coverage

Same test scenarios as v2.0, but connection-level:

- ✅ TC-1: Non-Live System (filesystem-only)
- ✅ TC-2: Live System - Both Sources Match
- ❌ TC-3: Live System - Runtime Missing Rules
- ❌ TC-4: Live System - Filesystem Missing Rules
- ❌ TC-5: Live System - Runtime Command Fails
- ❌ TC-6: Live System - Parse Errors
- ✅ TC-7: Set-Based Comparison (order doesn't matter)
- ✅ TC-8: Performance & Concurrency

---

## 9. Comparison: v2.0 vs v3.0

### Architecture Comparison

| Aspect | v2.0 (Resource-Level) | v3.0 (Connection-Level) |
|--------|----------------------|------------------------|
| **Data Source Logic** | In resource | In connection provider |
| **Source Selection** | `auditd.rules(source: "filesystem")` | Connection options/flags |
| **Resource Complexity** | High (dual-source logic) | Low (simple accessor) |
| **Alignment** | Custom pattern | K8s provider pattern |
| **Testing** | Resource tests | Provider + resource tests |
| **Extensibility** | Add to resource | Add to provider |

### Query Syntax Comparison

**v2.0**:
```mql
# Default
auditd.rules.files

# Explicit source
auditd.rules(source: "filesystem").files
auditd.rules(source: "runtime").files
auditd.rules(source: "both").files
```

**v3.0**:
```mql
# Always same query syntax
auditd.rules.files

# Source selection via connection (if needed)
# cnquery shell os --audit-source=filesystem
```

### Code Complexity Comparison

**v2.0**:
- Resource: ~500 lines (dual-source logic)
- Tests: Resource-focused

**v3.0**:
- Connection Provider: ~300 lines (data logic)
- Resource: ~100 lines (simple accessor)
- Tests: Provider tests + resource tests

---

## 10. Migration from v2.0 to v3.0

### If v2.0 Already Implemented

**Changes Required**:

1. **Remove from Resource** (`auditd.go`):
   - Remove `source` field from schema
   - Remove `initAuditdRules()` source handling
   - Remove dual-source loading logic
   - Simplify to connection delegation

2. **Add to Connection** (`connection/shared/`):
   - Create `AuditRuleProvider`
   - Move all data loading logic
   - Add to connection initialization

3. **Update Tests**:
   - Move logic tests to provider tests
   - Simplify resource tests to delegation tests

**Effort**: Medium (refactoring, not rewriting)

**Benefits**:
- Cleaner architecture
- Better aligned with cnquery patterns
- Easier to extend
- Better testability

---

## 11. Implementation Phases

### Phase 1: Connection Provider Foundation
**Goals**:
- Create `AuditRuleProvider` structure
- Implement filesystem loading
- Implement runtime loading
- Implement logical AND evaluation
- Add to OS connection

**Deliverables**:
- `connection/shared/audit_provider.go`
- Provider unit tests
- Connection integration

---

### Phase 2: Resource Simplification
**Goals**:
- Refactor resource to delegate to connection
- Remove dual-source complexity
- Update resource tests

**Deliverables**:
- Simplified `auditd.go`
- Updated resource tests
- Integration tests

---

### Phase 3: Testing & Documentation
**Goals**:
- Comprehensive test coverage
- Performance validation
- User documentation

**Deliverables**:
- Complete test suite
- Performance benchmarks
- Documentation updates

---

## 12. Success Metrics

### Functional
- ✅ All existing tests pass unchanged
- ✅ Dual-source works on live systems
- ✅ Filesystem-only works on non-live systems
- ✅ FAILED states provide clear messages

### Architectural
- ✅ Resource code < 150 lines (simplified)
- ✅ Connection provider isolated and testable
- ✅ Follows K8s provider patterns
- ✅ Clear separation of concerns

### Performance
- ✅ No regression on non-live systems
- ✅ Parallel loading improves performance
- ✅ Lazy evaluation prevents redundant work

---

## 13. Key Takeaways

### For Implementers
1. **Follow K8s pattern** - connection handles data, resource presents it
2. **Connection provider** - encapsulate all dual-source logic here
3. **Resource simplicity** - just delegate to connection
4. **Test at both levels** - provider tests + resource tests
5. **Reuse parser** - same format, same code

### For Users
1. **No query changes** - same syntax always works
2. **Automatic enhancement** - live systems get dual-source
3. **Transparent behavior** - connection determines data source
4. **Optional control** - connection flags for troubleshooting

### For Reviewers
1. **Architecture aligned** - follows established cnquery patterns
2. **Separation of concerns** - clear boundaries
3. **Testability** - provider isolated and mockable
4. **Backward compatible** - zero breaking changes

---

**Document Version**: 3.0  
**Architecture**: Connection-Level Provider (K8s Pattern)  
**Author**: AI Assistant  
**Date**: 2025-10-24  
**Status**: ✅ Ready for Implementation  
**Supersedes**: v2.0 (Resource-Level Source Parameter)

