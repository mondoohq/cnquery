# Technical Requirements Document: Extend `auditd.rules` Resource with Live Runtime Support

## Document Purpose
This document specifies requirements for extending the `auditd.rules` resource to support querying both filesystem-based audit rules AND live runtime audit rules from the Linux kernel when running on live systems.

**Target Audience**: LLM agents and developers implementing this feature  
**Status**: Design decisions finalized - ready for implementation

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

---

## 2. Objectives

### Primary Goal
Extend `auditd.rules` to query live runtime audit rules via `auditctl -l` when running on live systems, while maintaining 100% backward compatibility for existing MQL queries.

### Success Criteria
1. ✅ Existing MQL queries continue to work unchanged with same syntax
2. ✅ Automatic capability-based behavior (no user code changes required)
3. ✅ New `source` parameter enables explicit control when needed
4. ✅ Clear FAILED states identify source of failures (filesystem, runtime, or both)
5. ✅ All current key features preserved (see Section 6)
6. ✅ No performance degradation on non-live systems
7. ✅ Logical AND behavior when both sources available

---

## 3. Functional Requirements

### FR-1: Capability Detection
**Requirement**: Automatically detect if the system supports live rule querying.

**Implementation**:
```mql
# Condition to check
mondoo.capabilities.contains("run-command") == true
```

**Behavior**:
- `true`: System is live → collect both filesystem AND runtime rules, evaluate as logical AND
- `false`: System is non-live → collect only filesystem rules (current behavior)

**Key Point**: This happens transparently without user intervention.

---

### FR-2: Runtime Rule Collection
**Requirement**: When on a live system, execute `auditctl -l` to gather active kernel audit rules.

**Command**: `auditctl -l`

**✅ VERIFIED**: Output format from `auditctl -l` matches filesystem `.rules` format exactly.

**Sample Output**:
```
-D
-b 8192
-w /etc/shadow -p wa -k shadow_changes
-a always,exit -F arch=b64 -S adjtimex -S settimeofday -k time-change
```

**Parsing Requirements**:
- Parse same three rule types: control, file, syscall
- Handle same flag formats as filesystem parser
- Accumulate errors without failing fast
- Skip empty lines and comments
- **Reuse existing parsing logic** - both sources use identical format

---

### FR-3: Unified Resource with Source Awareness
**Requirement**: Single `auditd.rules` resource with transparent dual-source behavior.

**Architecture**:
- Maintain separate internal storage for filesystem vs runtime rules
- Present unified interface to users
- Add optional `source` parameter for explicit control
- Default behavior determined by capabilities

**Resource ID**: Unchanged - uses filesystem path only for stability.

---

### FR-4: Source Parameter API
**Requirement**: Optional `source` parameter for explicit source selection.

**Syntax**:
```mql
# Default: automatic behavior based on capabilities
auditd.rules.files                           # both sources on live systems
auditd.rules.syscalls                        # both sources on live systems

# Explicit source selection when needed
auditd.rules(source: "filesystem").files     # only filesystem
auditd.rules(source: "runtime").files        # only runtime (fails on non-live)
auditd.rules(source: "both").files           # explicit both (default)

# Comparison queries
auditd.rules(source: "filesystem").files.length
auditd.rules(source: "runtime").files.length
```

**Parameter Values**:
- `"both"` (default): Check both sources when available, logical AND evaluation
- `"filesystem"`: Only check filesystem rules
- `"runtime"`: Only check runtime rules (FAILED if not available)

**Backward Compatibility**:
- Existing queries without `source` parameter continue to work
- Behavior automatically upgrades to dual-source on live systems
- No MQL code changes required

---

### FR-5: Logical AND Evaluation (Strict Mode)
**Requirement**: When both sources are available, rules must pass checks in BOTH sources.

**Evaluation Logic**:

| Capability | Source Param | Filesystem | Runtime | Result |
|-----------|--------------|-----------|---------|--------|
| Live | `both` (default) | ✅ Pass | ✅ Pass | ✅ PASS |
| Live | `both` | ✅ Pass | ❌ Fail | ❌ FAILED (runtime) |
| Live | `both` | ❌ Fail | ✅ Pass | ❌ FAILED (filesystem) |
| Live | `both` | ❌ Fail | ❌ Fail | ❌ FAILED (both) |
| Live | `filesystem` | ✅ Pass | N/A | ✅ PASS |
| Live | `runtime` | N/A | ✅ Pass | ✅ PASS |
| Non-live | `both` | ✅ Pass | N/A | ✅ PASS (current behavior) |
| Non-live | `runtime` | N/A | N/A | ❌ FAILED (no runtime capability) |

**FAILED State Messages**:
- FAILED states include source information
- Format: `"Failed to load audit rules from [source]: [details]"`
- Multiple failures: `"Failed to load audit rules from filesystem and runtime: [filesystem: X, runtime: Y]"`

**Rationale**: Strict mode ensures security compliance - if rules are configured but not active, or active but not configured, this is a failure state requiring attention.

---

### FR-6: Error Handling & FAILED States
**Requirement**: Return FAILED states (not errors) with clear, actionable messages identifying failure source.

**✅ VERIFIED**: Error matrix approved.

**Scenarios**:

| Scenario | Filesystem | Runtime | State | Message |
|----------|-----------|---------|-------|---------|
| A | ✅ Success | ✅ Success | PASS | No message |
| B | ❌ Failed | ✅ Success | FAILED | "Failed to load audit rules from filesystem: [details]" |
| C | ✅ Success | ❌ Failed | FAILED | "Failed to load audit rules from runtime: [details]" |
| D | ❌ Failed | ❌ Failed | FAILED | "Failed to load audit rules from both filesystem and runtime: [filesystem: X, runtime: Y]" |
| E | ✅ Success | N/A (non-live) | PASS | No message (current behavior) |

**Runtime Command Failure Handling**:

Think of runtime execution as running `command("auditctl -l")`:
- Command not found → FAILED with OS error message
- Permission denied → FAILED with stderr: `"You must be root to run this program"` (RHEL example)
- Audit daemon not running → FAILED with OS error message
- Parsing errors → FAILED with accumulated parse errors

**Key Principle**: Always return FAILED state (not thrown errors), bubble up OS error messages as-is.

---

### FR-7: Key Features Preservation
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

## 4. Non-Functional Requirements

### NFR-1: Performance
- Runtime rule collection should not block filesystem rule collection
- Consider parallel loading with proper synchronization
- Cache runtime rules (lazy load once per query execution)
- No performance impact when `run-command` capability is false
- No redundant command executions

### NFR-2: Backward Compatibility
- **100% backward compatible**: Existing MQL queries work unchanged
- No syntax changes required in existing policies
- Behavior enhancement happens transparently under the hood
- Existing tests must continue to pass with same syntax
- Resource IDs remain stable (filesystem path only)

### NFR-3: Security
- Handle privilege escalation failures gracefully (return FAILED state)
- Never execute arbitrary commands (only `auditctl -l`)
- Sanitize command output before parsing
- Bubble up OS security errors as-is (e.g., "You must be root")

### NFR-4: Maintainability
- **Reuse existing parsing logic** - both sources use identical format
- Share test fixtures between filesystem and runtime parsers
- Single unified resource reduces API surface
- Clear separation of concerns in internal implementation

---

## 5. Design Decisions (FINALIZED)

### 5.1: API Design ✅ DECISION: Unified Resource with Optional Source Parameter

**Chosen Approach**: Mix of Options B and D with transparent behavior upgrade.

**Design**:
```mql
# Default behavior (automatic based on capabilities)
auditd.rules.files                    # Dual-source on live, filesystem-only on non-live
auditd.rules.syscalls                 # Same automatic behavior

# Explicit source control (when needed)
auditd.rules(source: "filesystem").files     # Only filesystem
auditd.rules(source: "runtime").files        # Only runtime
auditd.rules(source: "both").files           # Explicit both (same as default)
```

**Rationale**:
- ✅ 100% backward compatible - no code changes required
- ✅ Automatic enhancement on live systems
- ✅ Explicit control when disambiguation needed
- ✅ Single resource reduces complexity
- ✅ Intuitive default behavior

**Implementation Note**: `source` parameter defaults to `"both"`, capability detection happens internally.

---

### 5.2: Discrepancy Handling ✅ DECISION: Logical AND with Strict Mode

**Chosen Approach**: Option C - Strict Mode (Configurable via source parameter).

**Behavior**:
- **Both sources available + source="both"**: Logical AND - both must pass
- **Only filesystem available**: Current behavior (single source)
- **Explicit source selection**: Only that source evaluated

**When Sources Differ**:
- FAILED state returned
- Message indicates which source(s) failed
- Example: `"Rule exists in filesystem but not in runtime"` (implicit from FAILED state)

**Rationale**:
- Security-first: Misconfiguration or drift is a failure state
- Clear failure attribution via source information
- User can query sources separately for investigation
- Aligns with compliance use cases

---

### 5.3: Command Execution Failure Handling ✅ DECISION: FAILED State with OS Error Messages

**Chosen Approach**: Always return FAILED (not thrown error), bubble up OS messages.

**Behavior Matrix**:

| Condition | State | Message Source |
|-----------|-------|----------------|
| Command not found | FAILED | OS error message |
| Permission denied | FAILED | stderr from auditctl (e.g., "You must be root to run this program") |
| Daemon not running | FAILED | OS error message |
| Parse error | FAILED | Accumulated parse errors |

**Rationale**:
- Consistent with how `command()` resource would behave
- OS error messages are most accurate and actionable
- FAILED state allows queries to continue (not crash)
- Clear distinction from programming errors (which throw)

**Implementation**: Treat runtime collection as if running `command("auditctl -l")`, convert errors to FAILED states.

---

### 5.4: Resource ID Calculation ✅ DECISION: Unchanged (Filesystem Path)

**Chosen Approach**: Keep existing ID logic for stability.

```go
func (s *mqlAuditdRules) id() (string, error) {
    return s.Path.Data, nil  // Unchanged
}
```

**Rationale**:
- Single unified resource `auditd.rules`
- Dual-source behavior is internal implementation detail
- Stable IDs preserve caching behavior
- No breaking changes to query semantics

---

### 5.5: Rule Comparison Strategy ✅ DECISION: Set-Based (Order-Agnostic)

**Chosen Approach**: Treat rules as sets - only existence/non-existence matters.

**Rationale**:
- Rule order in audit configuration doesn't affect equality
- Set-based comparison aligns with security policy intent
- Simplifies diff logic
- Matches how audit system treats rules

**Implementation**: When comparing filesystem vs runtime, use set operations (not ordered list comparison).

---

## 6. Implementation Guidance

### 6.1: File Structure
```
providers/os/resources/
  auditd.go           # Existing, extend here
  auditd_runtime.go   # New: runtime rule collection via auditctl
  auditd_test.go      # Existing, extend here
  auditd_runtime_test.go  # New: runtime-specific tests
```

**Note**: Do NOT create `auditd_common.go` - both sources use identical parsing logic, reuse existing `parse()` method.

---

### 6.2: Parsing Strategy

**Reuse Existing Parser** (formats are identical ✅):
```go
// Existing method works for both sources
func parse(content string, controls, files, syscalls *[]interface{}, errors *[]interface{}) {
    // Existing implementation unchanged
}

// New wrapper for runtime
func (s *mqlAuditdRules) loadRuntime() (*plugin.TValue[[]interface{}], *plugin.TValue[[]interface{}], *plugin.TValue[[]interface{}], error) {
    // Check capability
    if !s.hasRunCommandCapability() {
        return nil, nil, nil, nil  // Return nil, not error
    }
    
    // Execute auditctl -l (like command() resource)
    output, err := s.MqlRuntime.ExecCommand("auditctl", "-l")
    if err != nil {
        // Convert to FAILED state with OS error message
        return nil, nil, nil, s.createFailedState("runtime", err)
    }
    
    // Reuse existing parser
    var controls, files, syscalls, errors []interface{}
    parse(output.String(), &controls, &files, &syscalls, &errors)

    if len(errors) > 0 {
        return nil, nil, nil, s.createFailedState("runtime", accumulatedErrors)
    }

    return controls, files, syscalls, nil
}
```

---

### 6.3: Source Parameter Implementation

**Parameter Handling**:
```go
// auditd.rules(source: "filesystem")
// auditd.rules(source: "runtime")
// auditd.rules(source: "both")  // default

func (s *mqlAuditdRules) init(args *resources.Args) (*resources.Args, AuditdRules, error) {
    // Extract source parameter, default to "both"
    source := args.GetString("source", "both")

    // Validate source value
    if source != "filesystem" && source != "runtime" && source != "both" {
        return nil, nil, errors.New("source must be 'filesystem', 'runtime', or 'both'")
    }

    return args, s, nil
}
```

**Evaluation Logic**:
```go
func (s *mqlAuditdRules) files() ([]interface{}, error) {
    source := s.GetSource("both")  // Default

    switch source {
    case "filesystem":
        return s.loadFilesystemFiles()
    case "runtime":
        return s.loadRuntimeFiles()
    case "both":
        return s.loadBothFiles()  // Logical AND
    }
}

func (s *mqlAuditdRules) loadBothFiles() ([]interface{}, error) {
    fsFiles, fsErr := s.loadFilesystemFiles()
    rtFiles, rtErr := s.loadRuntimeFiles()

    // Logical AND: both must succeed
    if fsErr != nil && rtErr != nil {
        return nil, s.createFailedState("both", fsErr, rtErr)
    }
    if fsErr != nil {
        return nil, s.createFailedState("filesystem", fsErr)
    }
    if rtErr != nil {
        return nil, s.createFailedState("runtime", rtErr)
    }

    // Merge and compare as sets
    return s.mergeAndValidate(fsFiles, rtFiles)
}
```

---

### 6.4: Synchronization Strategy

**Lazy Loading with Dual Lock**:
```go
type mqlAuditdRulesInternal struct {
    filesystemLock sync.Mutex
    filesystemLoaded bool
    filesystemData struct {
        controls []interface{}
        files    []interface{}
        syscalls []interface{}
    }
    filesystemError error
    
    runtimeLock sync.Mutex
    runtimeLoaded bool
    runtimeData struct {
        controls []interface{}
        files    []interface{}
        syscalls []interface{}
    }
    runtimeError error
}
```

**Performance Consideration**: Load both in parallel when `source="both"` using goroutines and wait groups.

---

## 7. Testing Requirements

### Test Categories

#### TC-1: Non-Live System Backward Compatibility ✅
**Setup**:
- Mock: `mondoo.capabilities` without "run-command"
- Source: default (both)

**Verify**:
- Only filesystem rules loaded
- All existing tests pass unchanged
- No runtime execution attempted
- Same query syntax works

---

#### TC-2: Live System - Both Sources Match ✅
**Setup**:
- Mock: Filesystem and `auditctl -l` return identical rules
- Source: default (both)

**Verify**:
- No FAILED states
- Both sources loaded successfully
- Logical AND passes (rules match)
- Result equals current behavior

---

#### TC-3: Live System - Runtime Missing Rules ❌
**Setup**:
- Mock: Filesystem has rules, runtime missing some
- Source: default (both)

**Verify**:
- FAILED state returned
- Message indicates runtime source failure
- Can query sources separately for investigation:
  - `auditd.rules(source: "filesystem").files` → works
  - `auditd.rules(source: "runtime").files` → shows missing rules

---

#### TC-4: Live System - Filesystem Missing Rules ❌
**Setup**:
- Mock: Runtime has rules, filesystem missing some
- Source: default (both)

**Verify**:
- FAILED state returned
- Message indicates filesystem source failure
- Clear indication of configuration drift

---

#### TC-5: Live System - Runtime Command Fails ❌
**Setup**:
- Mock: `auditctl -l` returns permission denied error
- Source: default (both)

**Verify**:
- FAILED state (not thrown error)
- Message includes stderr: "You must be root to run this program"
- Filesystem rules still accessible via `source: "filesystem"`

---

#### TC-6: Live System - Parse Errors in Runtime Output ❌
**Setup**:
- Mock: Malformed output from `auditctl -l`
- Source: default (both)

**Verify**:
- FAILED state with accumulated parse errors
- Partial data not used (strict mode)
- Clear error messages identify malformed lines

---

#### TC-7: Explicit Source Selection ✅
**Setup**:
- Live system with both sources available

**Verify**:
- `source: "filesystem"` only loads filesystem
- `source: "runtime"` only loads runtime
- `source: "both"` performs logical AND
- Non-live system with `source: "runtime"` returns FAILED

---

#### TC-8: Set-Based Comparison ✅
**Setup**:
- Filesystem: rules in order A, B, C
- Runtime: rules in order C, A, B (different order, same content)

**Verify**:
- Logical AND passes (order doesn't matter)
- No FAILED state
- Set comparison treats as equal

---

#### TC-9: Performance & Concurrency ✅
**Setup**:
- Concurrent access to rules from multiple goroutines

**Verify**:
- Lazy loading executes only once per source
- No redundant `auditctl -l` executions
- Thread-safe concurrent reads
- Both sources loaded in parallel when possible

---

## 8. Answers to Open Questions

### ✅ All Questions Resolved

1. **Output Format**: ✅ VERIFIED - `auditctl -l` output matches filesystem format exactly. Reuse existing parser.

2. **Permissions**: ✅ DECIDED - Bubble up OS error message as-is (e.g., "You must be root to run this program"). Return FAILED state, not thrown error.

3. **Audit Daemon States**: ✅ DECIDED - All daemon issues (stopped, crashed, etc.) result in FAILED state with OS error message bubbled up.

4. **Rule Ordering**: ✅ DECIDED - Order does NOT matter. Treat as sets, only existence/non-existence evaluated.

5. **Backward Compatibility**: ✅ DECIDED - 100% backward compatible. Default behavior automatically upgrades on live systems. No MQL code changes required.

6. **Resource Naming**: ✅ DECIDED - Keep single unified resource `auditd.rules`. Dual-source behavior is under-the-hood. Add `source` parameter (defaults to `"both"`).

---

## 9. Implementation Phases

### Phase 1: Core Functionality
**Goals**:
- Implement capability detection
- Add runtime rule collection via `auditctl -l`
- Implement `source` parameter
- Implement logical AND evaluation for `source="both"`
- Return FAILED states (not errors)
- Reuse existing parser

**Deliverables**:
- Extended `auditd.go` with dual-source support
- New `auditd_runtime.go` for command execution
- Updated schema with `source` parameter
- Core test suite (TC-1 through TC-6)

---

### Phase 2: Testing & Validation
**Goals**:
- Comprehensive test coverage all scenarios
- Backward compatibility validation
- Performance benchmarking
- Set-based comparison verification

**Deliverables**:
- Complete test suite (TC-1 through TC-9)
- Performance tests
- Integration tests with real `auditctl` output
- Documentation updates

---

### Phase 3: Documentation & Polish
**Goals**:
- User documentation
- Migration guide (spoiler: no migration needed!)
- Error message refinement
- Code review feedback incorporation

**Deliverables**:
- Updated resource documentation
- Example queries for common patterns
- Troubleshooting guide
- Release notes

---

## 10. Documentation Requirements

### User-Facing Documentation

#### Conceptual Guide
1. **What's New**: Automatic runtime rule checking on live systems
2. **How It Works**: Capability-based transparent enhancement
3. **When to Use `source` Parameter**: Disambiguation and troubleshooting scenarios
4. **Logical AND Behavior**: Why both sources must match

#### Query Examples
```mql
# Basic usage (automatic enhancement, no code changes)
auditd.rules.files.where(path == "/etc/shadow")
auditd.rules.syscalls.where(keys.contains("time-change"))

# Troubleshooting discrepancies
auditd.rules(source: "filesystem").files.length != auditd.rules(source: "runtime").files.length

# Explicit source for investigation
auditd.rules(source: "runtime").files.where(path == "/etc/passwd") {
  path
  permissions
  keyname
}

# Filesystem-only queries (e.g., for non-root scans)
auditd.rules(source: "filesystem").files
```

#### Common Scenarios
1. **Drift Detection**: Runtime rules differ from configuration
2. **Permission Issues**: Running without root access
3. **Audit Daemon Down**: Filesystem check still works
4. **Compliance Validation**: Both sources must match

---

### Developer Documentation

#### Architecture Decision Records
1. **ADR-001**: Unified resource with transparent dual-source behavior
2. **ADR-002**: Logical AND evaluation for strict compliance
3. **ADR-003**: FAILED states instead of thrown errors
4. **ADR-004**: Set-based rule comparison (order-agnostic)

#### Implementation Notes
1. **Parser Reuse**: Both sources use identical format, leverage existing `parse()` method
2. **Command Execution**: Treat as `command()` resource pattern
3. **Thread Safety**: Dual-lock strategy for concurrent access
4. **Performance**: Parallel loading when `source="both"`

#### Security Considerations
1. **Privilege Handling**: Graceful degradation when `auditctl` requires root
2. **Command Injection**: Only execute `auditctl -l` (no user input)
3. **Error Leakage**: Bubble up OS errors as-is for actionability

---

## 11. Success Metrics

### Functional Validation
- ✅ All existing tests pass unchanged
- ✅ New tests cover all scenarios (TC-1 through TC-9)
- ✅ Real-world `auditctl -l` output parses correctly
- ✅ FAILED states provide actionable error messages

### Non-Functional Validation
- ✅ Zero breaking changes (100% backward compatible)
- ✅ No performance regression on non-live systems
- ✅ Parallel loading improves performance on live systems
- ✅ Thread-safe under concurrent access

### User Experience
- ✅ Existing MQL queries work without modification
- ✅ Error messages clearly identify failure source
- ✅ `source` parameter provides intuitive control
- ✅ Documentation covers common troubleshooting scenarios

---

## 12. Implementation Checklist

### Prerequisites
- [x] Requirements document reviewed and approved
- [x] Design decisions finalized
- [x] `auditctl -l` format verified as identical to filesystem
- [x] All open questions answered

### Implementation Tasks
- [ ] Add `source` parameter to `auditd.rules` resource schema
- [ ] Implement capability detection (`run-command`)
- [ ] Create `auditd_runtime.go` with `auditctl -l` execution
- [ ] Implement FAILED state creation and handling
- [ ] Add logical AND evaluation for `source="both"`
- [ ] Implement set-based rule comparison
- [ ] Add parallel loading with synchronization
- [ ] Write comprehensive test suite (TC-1 through TC-9)
- [ ] Update user documentation with examples
- [ ] Create developer documentation (ADRs)
- [ ] Performance benchmarking
- [ ] Code review and refinement

---

## 13. Key Takeaways

### For Implementers
1. **Reuse existing parser** - formats are identical
2. **Think `command()` resource** - for runtime execution pattern
3. **FAILED not ERROR** - for operational failures
4. **Sets not lists** - for rule comparison
5. **Backward compatibility is paramount** - existing queries must work unchanged

### For Users
1. **No action required** - automatic enhancement on live systems
2. **Same queries work** - syntax unchanged
3. **Use `source` parameter** - when troubleshooting or investigating discrepancies
4. **Logical AND is strict** - both sources must match for pass
5. **FAILED states are informative** - include source and OS error details

### For Reviewers
1. **Zero breaking changes** - verify all existing tests pass
2. **Format verified identical** - `auditctl -l` == filesystem format
3. **Design decisions finalized** - all options chosen and documented
4. **Security-first approach** - strict mode catches drift
5. **Clear error attribution** - FAILED states identify source

---

**Document Version**: 2.0
**Author**: AI Assistant  
**Date**: 2025-10-24  
**Status**: ✅ Ready for Implementation
**Approvals**: Design decisions finalized, format verified, requirements complete
