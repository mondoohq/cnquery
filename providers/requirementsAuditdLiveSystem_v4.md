# Technical Requirements Document: Extend `auditd.rules` Resource with Live Runtime Support
## Architecture Version 4.0 - Hybrid Approach

## Document Purpose
This document specifies requirements for extending the `auditd.rules` resource to support querying both filesystem-based audit rules AND live runtime audit rules from the Linux kernel when running on live systems, with **strict validation** to detect configuration drift.

**Architecture Approach**: Connection-level data source management (v3.0 pattern) with strict logical AND validation (v2.0 behavior)  
**Target Audience**: LLM agents and developers implementing this feature  
**Status**: Draft - ready for review  
**Previous Versions**: 
- v2.0 (resource-level source parameter with strict validation)
- v3.0 (connection-level provider, lenient validation)

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
- **Configuration drift** - a critical security concern

### Key Requirements from v2.0 and v3.0

**From v2.0 (Keep)**:
- ✅ Strict validation - both filesystem and runtime must match
- ✅ Configuration drift detection
- ✅ Logical AND behavior
- ✅ Clear FAILED states when sources differ

**From v3.0 (Keep)**:
- ✅ Connection-level provider pattern
- ✅ Clean separation of concerns
- ✅ Resource simplicity
- ✅ Transparent behavior (no query changes needed)

**What Changes in v4.0**:
- ❌ Runtime as "source of truth" (v3.0 behavior) - too lenient
- ✅ Both sources must match exactly (v2.0 behavior) - security-first

---

## 2. Objectives

### Primary Goal
Extend the OS connection to provide both filesystem and runtime audit rule data when running on live systems, using connection-level abstraction (v3.0 pattern) with strict validation to detect configuration drift (v2.0 behavior).

### Success Criteria
1. ✅ Existing MQL queries continue to work unchanged with same syntax
2. ✅ Automatic capability-based behavior (no user code changes required)
3. ✅ Connection-level data source management (not resource-level)
4. ✅ **Strict validation: both sources must match on live systems**
5. ✅ Clear FAILED states identify source of failures AND mismatches
6. ✅ All current key features preserved
7. ✅ No performance degradation on non-live systems
8. ✅ **Logical AND behavior with strict comparison (security-first)**
9. ✅ Architecture aligned with existing cnquery connection patterns

---

## 3. Architecture Overview

### 3.1: Connection-Level Data Source Pattern (from v3.0)

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
│  - Filesystem rule loading (via resource)                   │
│  - Runtime rule loading (auditctl -l)                       │
│  - STRICT logical AND validation (v2.0 behavior)            │
│  - Error aggregation with drift detection                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│               auditd.rules Resource                          │
│  - Simple data accessor                                      │
│  - Loads filesystem rules (using MQL helpers)               │
│  - Delegates to provider for runtime merge/validation       │
│  - Presents unified view to user                            │
└─────────────────────────────────────────────────────────────┘
```

### 3.2: Key Architectural Principles

1. **Separation of Concerns** (from v3.0)
   - Connection: "How to get data"
   - Resource: "How to present data"
   - Provider: "How to validate and merge data"
   
2. **Strict Validation** (from v2.0)
   - Both sources must match on live systems
   - Configuration drift is a FAILED state
   - Security-first approach
   
3. **No Resource-Level Configuration**
   - No `source` parameter on resource (v3.0)
   - Behavior determined by connection capabilities
   
4. **Transparent Enhancement**
   - Live systems automatically get dual-source with validation
   - Non-live systems get filesystem-only behavior
   - No query syntax changes needed

---

## 4. Functional Requirements

### FR-1: Connection Capability Detection (from v3.0)
**Requirement**: OS connection automatically detects if the system supports live rule querying.

**Implementation**:
```go
type AuditRuleProvider struct {
    connection Connection
    useRuntime bool  // Set based on Capability_RunCommand
}

func NewAuditRuleProvider(conn Connection) *AuditRuleProvider {
    hasRunCommand := conn.Capabilities().Has(Capability_RunCommand)
    return &AuditRuleProvider{
        connection: conn,
        useRuntime: hasRunCommand,
    }
}
```

**Behavior**:
- `run-command` capability present → Enable dual-source mode with strict validation
- `run-command` capability absent → Filesystem-only mode
- Happens transparently during connection initialization

---

### FR-2: Connection-Level Audit Rule Provider (from v3.0)
**Requirement**: Create a connection-level provider that abstracts audit rule data sources.

**Interface Design**:
```go
type AuditRuleProvider struct {
    connection Connection
    useRuntime bool
    
    // Lazy loading with caching
    runtimeOnce sync.Once
    runtimeData *AuditRuleData
    runtimeErr  error
    
    // Parser function injected from resources
    parser AuditRuleParser
}

type AuditRuleData struct {
    Controls []interface{}
    Files    []interface{}
    Syscalls []interface{}
}

// Main method that resources call
func (p *AuditRuleProvider) GetRules(filesystemData *AuditRuleData) (*AuditRuleData, error) {
    if !p.useRuntime {
        return filesystemData, nil  // Non-live: filesystem only
    }
    return p.validateAndMerge(filesystemData)  // Live: strict validation
}
```

---

### FR-3: Runtime Rule Collection (from v2.0/v3.0)
**Requirement**: When on a live system, execute `auditctl -l` to gather active kernel audit rules.

**Command**: `auditctl -l`

**Implementation**:
```go
func (p *AuditRuleProvider) loadRuntimeRules() (*AuditRuleData, error) {
    if !p.useRuntime {
        return nil, nil
    }
    
    cmd, err := p.connection.RunCommand("auditctl -l")
    if err != nil {
        return nil, fmt.Errorf("failed to execute auditctl: %w", err)
    }
    
    if cmd.ExitStatus != 0 {
        stderr, _ := io.ReadAll(cmd.Stderr)
        return nil, fmt.Errorf("auditctl failed: %s", string(stderr))
    }
    
    stdout, _ := io.ReadAll(cmd.Stdout)
    
    // Parse using injected parser
    return p.parser(string(stdout))
}
```

---

### FR-4: Strict Logical AND Validation (NEW - hybrid of v2.0 and v3.0)
**Requirement**: When both sources are available, they must match exactly. Any divergence is a FAILED state.

**Validation Logic**:
```go
func (p *AuditRuleProvider) validateAndMerge(filesystemData *AuditRuleData) (*AuditRuleData, error) {
    // Load runtime rules
    runtimeData, rtErr := p.getRuntimeRules()
    
    // Handle runtime loading errors
    if rtErr != nil {
        // On live systems (useRuntime == true), ALL runtime errors are FAILED
        // No graceful fallback - runtime must be available and working
        return nil, fmt.Errorf("failed to load runtime rules: %w", rtErr)
    }
    
    // STRICT VALIDATION: Compare actual rule content as sets (not just counts)
    
    // Check controls match (set-based comparison)
    if !rulesMatchAsSet(filesystemData.Controls, runtimeData.Controls) {
        return nil, fmt.Errorf("control rules differ between filesystem and runtime (configuration drift detected)")
    }
    
    // Check files match (set-based comparison)
    if !rulesMatchAsSet(filesystemData.Files, runtimeData.Files) {
        return nil, fmt.Errorf("file rules differ between filesystem and runtime (configuration drift detected)")
    }
    
    // Check syscalls match (set-based comparison)
    if !rulesMatchAsSet(filesystemData.Syscalls, runtimeData.Syscalls) {
        return nil, fmt.Errorf("syscall rules differ between filesystem and runtime (configuration drift detected)")
    }
    
    // Both sources match - return runtime data as it's the current state
    return runtimeData, nil
}

// rulesMatchAsSet performs set-based comparison (order-agnostic)
func rulesMatchAsSet(a, b []interface{}) bool {
    if len(a) != len(b) {
        return false
    }
    
    // Special case: both empty
    if len(a) == 0 {
        return true
    }
    
    // Convert to sets and compare
    setA := makeRuleSet(a)
    setB := makeRuleSet(b)
    
    // Check all elements in A exist in B
    for rule := range setA {
        if !setB[rule] {
            return false
        }
    }
    
    return true
}
```

**Evaluation Matrix**:

| Capability | Filesystem | Runtime | Result |
|-----------|-----------|---------|--------|
| Live | ✅ Pass (5 rules) | ✅ Pass (5 matching rules) | ✅ PASS (content matches) |
| Live | ✅ Pass (5 rules) | ✅ Pass (0 rules) | ❌ FAILED (drift: config not loaded) |
| Live | ✅ Pass (5 rules) | ✅ Pass (7 rules) | ❌ FAILED (drift: extra runtime rules) |
| Live | ✅ Pass (5 rules) | ✅ Pass (5 different rules) | ❌ FAILED (drift: content differs) |
| Live | ❌ Fail (error) | ✅ Pass | ❌ FAILED (filesystem error) |
| Live | ✅ Pass | ❌ Fail (error) | ❌ FAILED (runtime error - no fallback) |
| Live | ✅ Pass | ❌ Fail (cmd not found) | ❌ FAILED (runtime error - no fallback) |
| Live | ✅ Pass | ❌ Fail (permission denied) | ❌ FAILED (runtime error - no fallback) |
| Live | ✅ Pass (0 rules) | ✅ Pass (0 rules) | ✅ PASS (both empty, returns []) |
| Non-live | ✅ Pass | N/A (not checked) | ✅ PASS (filesystem only) |

**Key Differences from v3.0**:
- ❌ v3.0: Returns runtime as source of truth (lenient)
- ❌ v3.0: Graceful fallback for "command not found"
- ✅ v4.0: Validates actual content matches as sets (strict)
- ✅ v4.0: ALL runtime errors are FAILED on live systems (no fallback)

---

### FR-5: Simplified Resource Implementation (from v3.0)
**Requirement**: Resource loads filesystem data and delegates validation to provider.

**Resource Pattern**:
```go
func (s *mqlAuditdRules) files(path string) ([]any, error) {
    // Load filesystem rules using existing helpers
    filesystemData, err := s.loadFilesystemRules(path)
    if err != nil {
        return nil, err
    }
    
    // Get provider from connection
    provider := s.MqlRuntime.Connection.(shared.Connection).AuditRuleProvider()
    provider.SetParser(s.parseAuditRules)
    
    // Provider validates and merges (strict mode)
    data, err := provider.GetRules(filesystemData)
    if err != nil {
        return nil, err  // Returns FAILED state
    }
    
    // Populate TValue fields
    s.Files.Data = data.Files
    s.Files.State = plugin.StateIsSet
    
    return data.Files, nil
}
```

**Schema** (unchanged from v3.0):
```
auditd.rules {
  init(path? string)
  path() string
  controls(path) []auditd.rule.control
  files(path) []auditd.rule.file
  syscalls(path) []auditd.rule.syscall
}
```

---

### FR-6: Error Handling & FAILED States (from v2.0)
**Requirement**: Return FAILED states (not errors) with clear, actionable messages identifying failure source.

**Error Scenarios**:

| Scenario | Filesystem | Runtime | Message |
|----------|-----------|---------|---------|
| A | ✅ Success | ✅ Success (match) | No error |
| B | ❌ Failed | N/A | "Failed to load audit rules from filesystem: [details]" |
| C | ✅ Success | ❌ Failed (error) | "Failed to load runtime rules: [details]" |
| D | ✅ Success | ❌ Failed (cmd not found) | "Failed to load runtime rules: command not found" |
| E | ✅ Success | ❌ Failed (permission) | "Failed to load runtime rules: You must be root" |
| F | ✅ Success (5) | ✅ Success (0) | "Control rules differ between filesystem and runtime (configuration drift detected)" |
| G | ✅ Success (5) | ✅ Success (7) | "File rules differ between filesystem and runtime (configuration drift detected)" |
| H | ✅ Success (5) | ✅ Success (5 different) | "Syscall rules differ between filesystem and runtime (configuration drift detected)" |
| I | ✅ Success (0) | ✅ Success (0) | No error (returns empty array []) |
| J | ✅ Success | N/A (non-live) | No error (filesystem only) |

**Special Cases**:
1. **Command not found** (live system): FAILED - auditd must be installed on live systems
2. **Permission denied** (live system): FAILED with stderr message
3. **Daemon not running** (live system): FAILED with OS error
4. **Configuration drift**: FAILED when content differs (set-based comparison)
5. **Both empty**: Returns empty array [] (valid state, not FAILED)
6. **Non-live system**: Runtime never checked, filesystem only

---

### FR-7: Key Features Preservation (from v2.0/v3.0)
**Requirement**: All existing key features MUST be maintained:

1. ✅ **Automatic categorization** of different rule types (control, file, syscall)
2. ✅ **Structured field parsing** for syscall filters
3. ✅ **Operator parsing** (=, !=, >=, <=, >, <) for field comparisons
4. ✅ **Multiple file support** - reads all `.rules` files in a directory
5. ✅ **Thread-safe** loading with mutex
6. ✅ **Error accumulation** - collects all parsing errors instead of failing fast
7. ✅ **Lazy evaluation** - rules are parsed only when accessed
8. ✅ **Set-based comparison** - rule order does not matter (TODO: content matching)

---

## 5. Non-Functional Requirements

### NFR-1: Performance (from v3.0)
- Parallel loading of filesystem and runtime rules where possible
- Lazy evaluation with sync.Once pattern
- Cache runtime rule data per connection instance
- No performance impact when `run-command` capability is false
- No redundant command executions

### NFR-2: Backward Compatibility (from v2.0/v3.0)
- **100% backward compatible**: Existing MQL queries work unchanged
- No syntax changes required in existing policies
- Behavior enhancement happens transparently at connection level
- Existing tests must continue to pass with same syntax
- Resource IDs remain stable

### NFR-3: Security (from v2.0)
- **Configuration drift detection is primary security feature**
- Handle privilege escalation failures gracefully (return FAILED state)
- Never execute arbitrary commands (only `auditctl -l`)
- Sanitize command output before parsing
- Bubble up OS security errors as-is (e.g., "You must be root")

### NFR-4: Maintainability (from v3.0)
- **Connection-level abstraction** reduces resource complexity
- **Reuse existing parsing logic** - both sources use identical format
- **Clear separation of concerns** - connection handles data, resource presents it
- **Aligned with cnquery patterns** - follows K8s provider architecture

---

## 6. Design Decisions (FINALIZED)

### 6.1: Architecture Pattern ✅ DECISION: Connection-Level Provider (from v3.0)

**Chosen Approach**: K8s-inspired connection abstraction pattern

**Design**:
```
Connection (OS)
  └── AuditRuleProvider
        ├── Capability detection
        ├── Runtime loading (auditctl -l)
        ├── STRICT validation (both must match)
        └── Drift detection

Resource (auditd.rules)
  └── Simple accessor
        ├── Loads filesystem rules
        └── Calls provider for validation
```

**Rationale**:
- ✅ Aligned with existing cnquery/K8s provider patterns
- ✅ Clean separation of concerns
- ✅ Resource simplicity
- ✅ Testability (can mock connection provider)
- ✅ Extensibility

---

### 6.2: Validation Approach ✅ DECISION: Strict Logical AND (from v2.0)

**Chosen Approach**: Both sources must match exactly on live systems

**Behavior**:
- Both sources available → Both must have same count of rules per category
- Any mismatch → FAILED state with drift details
- Command not found → Graceful fallback to filesystem

**Rationale**:
- ✅ **Security-first**: Configuration drift is a critical issue
- ✅ **Compliance**: Audit rules must be enforced as configured
- ✅ **Visibility**: Clear indication when runtime differs from config
- ✅ **Actionable**: Users can investigate and remediate drift

**Example Drift Scenarios**:
1. Filesystem: 5 rules, Runtime: 0 rules → FAILED ("config not loaded into kernel")
2. Filesystem: 10 rules, Runtime: 12 rules → FAILED ("extra runtime rules detected")
3. Filesystem: 5 rules A,B,C,D,E, Runtime: 5 rules A,B,C,X,Y → FAILED ("rule content differs")
4. Filesystem: 0 rules, Runtime: 0 rules → PASS (returns empty array [])

---

### 6.3: Comparison Strategy ✅ DECISION: Set-Based Content Comparison

**Implementation**: Full content comparison as sets (order-agnostic)

```go
func rulesMatchAsSet(a, b []interface{}) bool {
    // Length must match first
    if len(a) != len(b) {
        return false
    }
    
    // Both empty is a match
    if len(a) == 0 {
        return true
    }
    
    // Convert rules to comparable format and check set equality
    setA := makeRuleSet(a)
    setB := makeRuleSet(b)
    
    // All elements in A must exist in B
    for rule := range setA {
        if !setB[rule] {
            return false
        }
    }
    
    return true
}
```

**Rationale**:
- ✅ Content comparison catches all drift scenarios (not just counts)
- ✅ Order doesn't matter (set semantics)
- ✅ More thorough validation for compliance
- ✅ Detects both missing rules AND different rules at same count

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
      audit_provider.go      # AuditRuleProvider implementation
      audit_provider_test.go # Provider tests
  resources/
    auditd.go               # Resource delegates to provider
    auditd_test.go          # Resource tests
    os.lr                   # Schema (no source parameter)
```

### 7.2: Provider Implementation

**Key Methods**:

```go
// Main validation method
func (p *AuditRuleProvider) GetRules(filesystemData *AuditRuleData) (*AuditRuleData, error) {
    if !p.useRuntime {
        return filesystemData, nil
    }
    return p.validateAndMerge(filesystemData)
}

// Strict validation
func (p *AuditRuleProvider) validateAndMerge(fs *AuditRuleData) (*AuditRuleData, error) {
    rt, err := p.getRuntimeRules()
    if err != nil {
        if isCommandNotFound(err) {
            return fs, nil  // Graceful fallback
        }
        return nil, err
    }
    
    // STRICT: counts must match
    if len(fs.Controls) != len(rt.Controls) {
        return nil, fmt.Errorf("drift detected: controls %d vs %d", 
            len(fs.Controls), len(rt.Controls))
    }
    // ... similar for Files and Syscalls
    
    return rt, nil  // Return runtime as current state
}
```

### 7.3: Resource Implementation

**Simplified Pattern**:

```go
func (s *mqlAuditdRules) files(path string) ([]any, error) {
    // 1. Load filesystem rules (using MQL helpers)
    fsData, err := s.loadFilesystemRules(path)
    if err != nil {
        return nil, err
    }
    
    // 2. Get provider and inject parser
    provider := s.MqlRuntime.Connection.(shared.Connection).AuditRuleProvider()
    provider.SetParser(s.parseAuditRules)
    
    // 3. Provider validates (strict mode)
    data, err := provider.GetRules(fsData)
    if err != nil {
        return nil, err  // FAILED state
    }
    
    // 4. Populate and return
    s.Files.Data = data.Files
    s.Files.State = plugin.StateIsSet
    return data.Files, nil
}
```

---

## 8. Testing Requirements

### Test Scenarios

**TC-1: Non-Live System** ✅
- Input: Filesystem rules, no run-command capability
- Expected: Returns filesystem rules, no runtime check
- Status: PASS

**TC-2: Live System - Perfect Match** ✅
- Input: Filesystem has 5 controls, Runtime has 5 controls (matching)
- Expected: Returns runtime rules
- Status: PASS

**TC-3: Live System - Drift (Runtime Missing)** ❌
- Input: Filesystem has 5 controls, Runtime has 0 controls
- Expected: FAILED with "control rules differ: filesystem has 5, runtime has 0"
- Status: FAILED (drift detected)

**TC-4: Live System - Drift (Extra Runtime)** ❌
- Input: Filesystem has 10 files, Runtime has 12 files
- Expected: FAILED with "file rules differ: filesystem has 10, runtime has 12"
- Status: FAILED (drift detected)

**TC-5: Live System - Runtime Error (Permission)** ❌
- Input: Filesystem OK, auditctl fails with permission denied
- Expected: FAILED with "failed to load runtime rules: You must be root"
- Status: FAILED (runtime error)

**TC-6: Live System - Runtime Error (Command Not Found)** ❌
- Input: Filesystem OK, auditctl command not found
- Expected: FAILED with "failed to load runtime rules: command not found"
- Status: FAILED (runtime error - no fallback on live systems)

**TC-7: Set-Based Comparison - Same Content Different Order** ✅
- Input: Filesystem has [A,B,C], Runtime has [C,A,B] (same rules, different order)
- Expected: PASS (order doesn't matter, content matches)
- Status: PASS

**TC-8: Set-Based Comparison - Different Content Same Count** ❌
- Input: Filesystem has 3 rules [A,B,C], Runtime has 3 rules [A,B,X]
- Expected: FAILED with "rules differ between filesystem and runtime"
- Status: FAILED (drift detected)

**TC-9: Both Empty** ✅
- Input: Filesystem has 0 rules, Runtime has 0 rules
- Expected: Returns empty array []
- Status: PASS (valid state)

---

## 9. Key Differences from Previous Versions

### From v2.0:
**Keep**:
- ✅ Strict validation (both must match)
- ✅ Configuration drift detection
- ✅ Logical AND behavior

**Change**:
- ❌ Resource-level source parameter → ✅ Connection-level provider
- ❌ Complex resource logic → ✅ Simple delegation

### From v3.0:
**Keep**:
- ✅ Connection-level provider pattern
- ✅ Clean architecture
- ✅ Resource simplicity

**Change**:
- ❌ Lenient validation (runtime as truth) → ✅ Strict validation (both must match)
- ❌ Returns runtime when both exist → ✅ Validates counts, returns runtime if match

---

## 10. Success Criteria

### Functional
- ✅ Detects configuration drift (filesystem ≠ runtime)
- ✅ Returns FAILED state with clear drift messages
- ✅ Gracefully handles missing auditctl
- ✅ Works on non-live systems (filesystem only)

### Architectural
- ✅ Connection-level provider (clean separation)
- ✅ Resource < 200 lines (simple delegation)
- ✅ Follows K8s provider patterns

### Security
- ✅ Configuration drift is a FAILED state
- ✅ Missing runtime rules detected
- ✅ Extra runtime rules detected
- ✅ Compliance-friendly (strict mode)

---

## 11. Implementation Phases

### Phase 1: Update Validation Logic
**Tasks**:
- Modify `validateAndMerge()` to be strict (content comparison)
- Implement `rulesMatchAsSet()` for set-based comparison
- Remove all graceful fallbacks on live systems
- Add drift detection error messages
- Handle empty array case (both 0 rules)
- Update tests for new behavior

### Phase 2: Testing
**Tasks**:
- Test drift scenarios (TC-3, TC-4, TC-8)
- Test runtime error handling (TC-5, TC-6 - all FAILED)
- Test set-based comparison (TC-7 - order doesn't matter)
- Test empty array case (TC-9)
- Verify backward compatibility on non-live systems

### Phase 3: Documentation
**Tasks**:
- Update implementation tracking document
- Document drift detection behavior
- Provide troubleshooting guide

---

## 12. Clarifications (RESOLVED)

1. **Graceful fallback scenarios** ✅ RESOLVED:
   - **IF** `run-command` capability is TRUE (live system):
     - Command not found → FAILED
     - Permission denied → FAILED
     - Daemon not running → FAILED
   - **IF** `run-command` capability is FALSE (non-live system):
     - Don't check runtime at all, only filesystem
   - **Key**: No graceful fallback on live systems - runtime failures are FAILED states

2. **Comparison strategy** ✅ RESOLVED:
   - Use actual rule content comparison (set-based)
   - Not just count comparison
   - Order doesn't matter (set semantics)

3. **Zero rules edge case** ✅ RESOLVED:
   - If both filesystem AND runtime have 0 rules → Return empty list/array []
   - This will fail checks like `.any()` but is not an error
   - Empty is a valid state (no audit rules configured)

---

**Document Version**: 4.0  
**Architecture**: Connection-Level Provider with Strict Validation  
**Author**: AI Assistant  
**Date**: 2025-10-24  
**Status**: ✅ FINALIZED - Ready for Implementation  
**Changes from v3.0**: 
- Added strict validation logic (v2.0 behavior) while keeping v3.0 architecture
- Set-based content comparison (not just counts)
- No graceful fallbacks on live systems (all runtime errors are FAILED)
- Empty array handling (both 0 rules returns [])

