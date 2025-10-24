# Technical Requirements Document: Extend `auditd.rules` Resource with Live Runtime Support

## Document Purpose
This document specifies requirements for extending the `auditd.rules` resource to support querying both filesystem-based audit rules AND live runtime audit rules from the Linux kernel when running on live systems.

**Target Audience**: LLM agents and developers implementing this feature  
**Status**: Requirements gathering - pending design decisions

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
Extend `auditd.rules` to query live runtime audit rules via `auditctl -l` when running on live systems, while maintaining 100% backward compatibility for non-live systems.

### Success Criteria
1. ✅ Existing queries continue to work unchanged
2. ✅ New capabilities expose runtime vs filesystem rule differences
3. ✅ Clear error messages identify source of failures (filesystem, runtime, or both)
4. ✅ All current key features preserved (see Section 6)
5. ✅ No performance degradation on non-live systems

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
- `true`: System is live → collect both filesystem AND runtime rules
- `false`: System is non-live → collect only filesystem rules (current behavior)

---

### FR-2: Runtime Rule Collection
**Requirement**: When on a live system, execute `auditctl -l` to gather active kernel audit rules.

**Command**: `auditctl -l`

**Sample Output** (needs verification):
```
# Example output format from auditctl -l
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

**⚠️ RESEARCH NEEDED**: Verify exact output format of `auditctl -l` matches filesystem format or document differences.

---

### FR-3: Separate Data Structures
**Requirement**: Maintain distinct storage for filesystem vs runtime rules.

**Rationale**: 
- Enable comparison between declared (filesystem) vs actual (runtime) state
- Support error messages identifying specific sources
- Allow future reconciliation queries

---

### FR-4: Unified Query Interface
**Requirement**: Users should be able to query rules regardless of source.

**⚠️ DESIGN DECISION NEEDED**: See Section 5.1

---

### FR-5: Error Handling & Reporting
**Requirement**: Provide clear, actionable error messages identifying failure source.

**Scenarios**:

| Scenario | Filesystem | Runtime | Error Message |
|----------|-----------|---------|---------------|
| A | ✅ Success | ✅ Success | No error |
| B | ❌ Failed | ✅ Success | "Failed to load audit rules from filesystem: [details]" |
| C | ✅ Success | ❌ Failed | "Failed to load live audit rules via auditctl: [details]" |
| D | ❌ Failed | ❌ Failed | "Failed to load audit rules from both filesystem and runtime: [filesystem: X, runtime: Y]" |
| E | ✅ Success | N/A (non-live) | No error (current behavior) |

**Additional Error Cases**:
- `auditctl` command not found
- `auditctl` requires elevated privileges
- Audit daemon not running
- Parsing errors in runtime output

---

### FR-6: Key Features Preservation
**Requirement**: All existing key features MUST be maintained:

1. ✅ **Automatic categorization** of different rule types (control, file, syscall)
2. ✅ **Structured field parsing** for syscall filters
3. ✅ **Operator parsing** (=, !=, >=, <=, >, <) for field comparisons
4. ✅ **Multiple file support** - reads all `.rules` files in a directory
5. ✅ **Thread-safe** loading with mutex
6. ✅ **Error accumulation** - collects all parsing errors instead of failing fast
7. ✅ **Lazy evaluation** - rules are parsed only when accessed

---

## 4. Non-Functional Requirements

### NFR-1: Performance
- Runtime rule collection should not block filesystem rule collection
- Consider parallel loading with proper synchronization
- Cache runtime rules (lazy load once per query execution)
- No performance impact when `run-command` capability is false

### NFR-2: Backward Compatibility
- Zero breaking changes to existing MQL queries
- Existing tests must continue to pass
- Resource IDs remain stable

### NFR-3: Security
- Handle privilege escalation failures gracefully
- Never execute arbitrary commands (only `auditctl -l`)
- Sanitize any command output before parsing

### NFR-4: Maintainability
- Reuse existing `parseKeyVal()` and parsing logic where possible
- Share test fixtures between filesystem and runtime parsers
- Document any format differences between filesystem and `auditctl -l` output

---

## 5. Design Decisions Required

### 5.1: API Design - How to Expose Runtime vs Filesystem Rules?

**DECISION NEEDED**: Choose one option or propose alternative.

#### **Option A: Separate Properties (Explicit)**
```mql
# Filesystem rules (current behavior)
auditd.rules.files
auditd.rules.syscalls
auditd.rules.controls

# New: Runtime rules
auditd.rules.runtime.files
auditd.rules.runtime.syscalls
auditd.rules.runtime.controls

# Example usage
auditd.rules.runtime.files.where(path == "/etc/shadow")
```

**Pros**:
- ✅ Explicit and clear what you're querying
- ✅ Easy to compare filesystem vs runtime
- ✅ No ambiguity
- ✅ Backward compatible (existing queries unchanged)

**Cons**:
- ❌ Verbose for simple checks
- ❌ Need to query both sources separately for complete picture

---

#### **Option B: Merged View with Source Metadata**
```mql
# All rules merged with source indicator
auditd.rules.files[] {
  path
  permissions
  keyname
  source  # "filesystem" | "runtime" | "both"
}

# Example usage
auditd.rules.files.where(source == "runtime")
auditd.rules.files.where(source != "filesystem")  # Only in runtime, not in files
```

**Pros**:
- ✅ Concise queries
- ✅ Easy to find discrepancies
- ✅ Single query covers both sources

**Cons**:
- ❌ Complex deduplication logic (what if same rule exists in both?)
- ❌ Harder to implement
- ❌ Potentially confusing default behavior

---

#### **Option C: Both Separate and Merged (Comprehensive)**
```mql
# Separate access
auditd.rules.filesystem.files
auditd.rules.runtime.files

# Merged view
auditd.rules.all.files[] {
  source
  ...
}

# Comparison helpers
auditd.rules.diff.filesystemOnly
auditd.rules.diff.runtimeOnly
auditd.rules.diff.common
```

**Pros**:
- ✅ Maximum flexibility
- ✅ Explicit when needed, convenient when not
- ✅ Built-in diff capabilities

**Cons**:
- ❌ Larger API surface
- ❌ More implementation complexity
- ❌ More testing required

---

#### **Option D: Smart Default with Override**
```mql
# Default: queries runtime if available, falls back to filesystem
auditd.rules.files  # Smart: runtime on live systems, filesystem otherwise

# Explicit when needed
auditd.rules.source("filesystem").files
auditd.rules.source("runtime").files

# Comparison
auditd.rules.source("both").files.where(sources.length == 1)  # Discrepancies
```

**Pros**:
- ✅ Intuitive for most use cases
- ✅ Backward compatible with new behavior
- ✅ Explicit override when needed

**Cons**:
- ❌ "Smart" defaults can be confusing
- ❌ Breaking change in behavior (queries different data on live systems)
- ❌ Complex to reason about

---

**RECOMMENDATION REQUEST**: Which option aligns best with MQL design principles?

---

### 5.2: Discrepancy Handling

**DECISION NEEDED**: When filesystem and runtime rules differ, what should happen?

#### **Option A: Informational Only**
- No errors, just expose both datasets
- User writes queries to detect discrepancies
- Example: `auditd.rules.runtime.files.length != auditd.rules.filesystem.files.length`

#### **Option B: Warning/Advisory**
- Log warnings when discrepancies detected
- Don't fail queries
- Provide helper methods to check alignment

#### **Option C: Configurable Strictness**
- Default: informational
- Strict mode: fail if runtime != filesystem
- Advisory mode: warnings only

**QUESTION**: What's the expected user workflow when rules differ?

---

### 5.3: Command Execution Failure Handling

**DECISION NEEDED**: What happens when `auditctl -l` fails?

#### **Scenario Matrix**:

| Condition | Behavior Option 1 | Behavior Option 2 |
|-----------|------------------|-------------------|
| Command not found | Error only on runtime access | Silently fall back to filesystem |
| Permission denied | Error only on runtime access | Silently fall back to filesystem |
| Daemon not running | Error only on runtime access | Silently fall back to filesystem |
| Parse error | Accumulate errors, partial data | Fail completely |

**QUESTION**: Should failures be fatal or gracefully degrade?

---

### 5.4: Resource ID Calculation

**DECISION NEEDED**: How should resource ID work with dual sources?

Current ID logic:
```go
func (s *mqlAuditdRules) id() (string, error) {
    return s.Path.Data, nil  // Just the path
}
```

**Options**:
1. Keep unchanged (filesystem path only)
2. Include capability indicator: `${path}:${hasRuntime}`
3. Separate resources: `auditd.rules.filesystem` and `auditd.rules.runtime`

**Impact**: Affects caching and query behavior.

---

## 6. Implementation Guidance

### 6.1: Suggested File Structure
```
providers/os/resources/
  auditd.go           # Existing, extend here
  auditd_runtime.go   # New: runtime rule collection
  auditd_common.go    # Shared parsing utilities
  auditd_test.go      # Existing, extend here
  auditd_runtime_test.go  # New: runtime-specific tests
```

### 6.2: Parsing Strategy

**Reuse Existing Parser**:
- The `parse()` method should work for both sources
- Create wrapper: `parseFilesystem()` and `parseRuntime()`
- `parseRuntime()` executes command, feeds output to `parse()`

**Command Execution**:
```go
// Pseudocode
func (s *mqlAuditdRules) loadRuntime() error {
    // Check capability
    if !hasRunCommandCapability() {
        return nil  // Skip silently or set flag
    }
    
    // Execute auditctl -l
    output, err := s.MqlRuntime.ExecCommand("auditctl", "-l")
    if err != nil {
        // Decision point: fail or degrade gracefully?
        return handleCommandError(err)
    }
    
    // Parse using existing logic
    return s.parseRuntime(output, &errors)
}
```

### 6.3: Synchronization Strategy

**Lazy Loading with Dual Lock**:
```go
type mqlAuditdRulesInternal struct {
    filesystemLock sync.Mutex
    filesystemLoaded bool
    filesystemError error
    
    runtimeLock sync.Mutex
    runtimeLoaded bool
    runtimeError error
}
```

**Consideration**: Could potentially load both in parallel for performance.

---

## 7. Testing Requirements

### Test Cases Needed

#### TC-1: Non-Live System (Existing Behavior)
- Mock: `mondoo.capabilities` without "run-command"
- Verify: Only filesystem rules loaded
- Verify: All existing tests pass

#### TC-2: Live System - Both Sources Match
- Mock: Filesystem and `auditctl -l` return identical rules
- Verify: No errors
- Verify: Both sources accessible

#### TC-3: Live System - Sources Differ
- Mock: Different rules in filesystem vs runtime
- Verify: Both accessible separately
- Verify: Discrepancies detectable

#### TC-4: Live System - Runtime Command Fails
- Mock: `auditctl -l` returns error
- Verify: Appropriate error handling per decision 5.3
- Verify: Filesystem rules still accessible

#### TC-5: Live System - Parse Errors
- Mock: Malformed output from `auditctl -l`
- Verify: Errors accumulated
- Verify: Partial data available

#### TC-6: Performance
- Verify: Lazy loading works
- Verify: No redundant executions
- Verify: Thread-safe concurrent access

---

## 8. Open Questions

1. **Output Format**: Does `auditctl -l` output match filesystem format exactly? Need to verify with real system.

2. **Permissions**: What error message should appear when `auditctl` requires sudo but isn't available?

3. **Audit Daemon States**: How to handle various daemon states (stopped, starting, crashed)?

4. **Rule Ordering**: Does order matter when comparing filesystem vs runtime? Should we treat rules as sets or ordered lists?

5. **Backward Compatibility**: Should the default behavior change for live systems, or remain filesystem-only unless explicitly requested?

6. **Resource Naming**: Should we introduce `auditd.rules.runtime` as a new resource or extend existing `auditd.rules`?

---

## 9. Migration Path

### Phase 1: Foundation (Minimal Breaking Change)
- Implement Option A (separate properties)
- Add `auditd.rules.runtime.*` properties
- Keep existing properties unchanged
- Document new capabilities

### Phase 2: Enhancements (Based on Feedback)
- Add comparison helpers if needed
- Implement merged views if desired
- Add convenience methods

### Phase 3: Optimization
- Performance tuning
- Caching strategies
- Error message refinement

---

## 10. Documentation Requirements

### User-Facing Documentation
1. Explain difference between filesystem and runtime rules
2. When to use each data source
3. Common discrepancy scenarios and remediation
4. Query examples for typical use cases

### Developer Documentation
1. Architecture decision records for design choices
2. Parser reuse strategy
3. Testing approach for dual-source scenarios
4. Command execution security considerations

---

## 11. Questions for Clarification

### Critical Decisions Needed:
1. **API Design** (Section 5.1): Which option for exposing filesystem vs runtime rules?
2. **Failure Handling** (Section 5.3): Fatal errors or graceful degradation?
3. **Default Behavior** (Section 8.5): Should queries on live systems default to runtime or filesystem?
4. **Discrepancy Severity** (Section 5.2): Information, warning, or error when sources differ?

### Nice-to-Have Guidance:
5. Preferred naming conventions for new properties/methods
6. Expected query patterns from security policies
7. Priority: correctness vs performance vs backward compatibility

---

## 12. Next Steps

**Before Implementation**:
1. ✅ Review this requirements document
2. ⏳ Make design decisions (Sections 5.1-5.4)
3. ⏳ Research `auditctl -l` output format on real systems
4. ⏳ Define exact API structure in `os.lr`
5. ⏳ Create implementation plan

**Please provide**:
- Decisions on options presented in Section 5
- Answers to open questions in Section 8
- Any additional requirements or constraints
- Approval to proceed with implementation

---

**Document Version**: 1.0  
**Author**: AI Assistant  
**Date**: 2025-10-24  
**Status**: Awaiting Design Decisions