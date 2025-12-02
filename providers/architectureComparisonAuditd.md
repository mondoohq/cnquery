# Architecture Comparison: auditd.rules Extension Approaches
## v2.0 (Resource-Level) vs v3.0 (Connection-Level)

**Document Purpose**: Detailed comparison of architectural approaches for extending `auditd.rules` with live runtime support

**Date**: 2025-10-24  
**Status**: Analysis Complete

---

## Executive Summary

### v2.0 Approach: Resource-Level Source Parameter
- **Pattern**: Resource manages dual data sources internally
- **API**: `auditd.rules(source: "filesystem")` parameter
- **Pros**: Self-contained, explicit control
- **Cons**: Not aligned with cnquery patterns, complex resource logic

### v3.0 Approach: Connection-Level Provider
- **Pattern**: Connection abstracts data sources (K8s-inspired)
- **API**: No query changes, connection determines behavior
- **Pros**: Aligned with cnquery patterns, clean separation
- **Cons**: Requires connection-level changes

### Recommendation: **v3.0 (Connection-Level Provider)**

**Rationale**:
1. Aligns with established cnquery/K8s provider architecture
2. Better separation of concerns
3. More maintainable and extensible
4. Cleaner resource implementation
5. Better testability

---

## Detailed Comparison

### 1. Architectural Patterns

#### v2.0: Resource-Level Source Management

```
┌─────────────────────────────────────────┐
│          auditd.rules Resource          │
│  ┌───────────────────────────────────┐  │
│  │  Source Parameter Handling        │  │
│  │  - Validate source value          │  │
│  │  - Determine which sources to load│  │
│  │  - Dispatch to loaders            │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  Dual-Source Storage              │  │
│  │  - filesystemData                 │  │
│  │  - runtimeData                    │  │
│  │  - filesystemErr / runtimeErr     │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  Loading Logic                    │  │
│  │  - loadFilesystemRules()          │  │
│  │  - loadRuntimeRules()             │  │
│  │  - loadBothSources()              │  │
│  └───────────────────────────────────┘  │
│  ┌───────────────────────────────────┐  │
│  │  Logical AND Evaluation           │  │
│  │  - Merge sources                  │  │
│  │  - Validate consistency           │  │
│  │  - Return FAILED if mismatch      │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**Characteristics**:
- ⚠️ Resource has multiple responsibilities
- ⚠️ Complex internal state management
- ✅ Self-contained (no connection changes)
- ⚠️ Custom pattern not seen elsewhere

---

#### v3.0: Connection-Level Provider

```
┌─────────────────────────────────────────┐
│         OS Connection                    │
│  ┌───────────────────────────────────┐  │
│  │  AuditRuleProvider                │  │
│  │  ┌─────────────────────────────┐  │  │
│  │  │  Capability Detection       │  │  │
│  │  │  - hasRunCommand?           │  │  │
│  │  │  - Set useRuntime flag      │  │  │
│  │  └─────────────────────────────┘  │  │
│  │  ┌─────────────────────────────┐  │  │
│  │  │  Dual-Source Storage        │  │  │
│  │  │  - filesystemData           │  │  │
│  │  │  - runtimeData              │  │  │
│  │  │  - Lazy loading (once.Do)   │  │  │
│  │  └─────────────────────────────┘  │  │
│  │  ┌─────────────────────────────┐  │  │
│  │  │  GetRules(path)             │  │  │
│  │  │  - Auto-select sources      │  │  │
│  │  │  - Load & validate          │  │  │
│  │  │  - Return unified data      │  │  │
│  │  └─────────────────────────────┘  │  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
                   │
                   │ GetRules()
                   ▼
┌─────────────────────────────────────────┐
│      auditd.rules Resource               │
│  ┌───────────────────────────────────┐  │
│  │  Simple Delegation                │  │
│  │  - controls() → provider.GetRules()│  │
│  │  - files() → provider.GetRules()   │  │
│  │  - syscalls() → provider.GetRules()│  │
│  └───────────────────────────────────┘  │
└─────────────────────────────────────────┘
```

**Characteristics**:
- ✅ Single responsibility per component
- ✅ Connection owns data fetching
- ✅ Resource owns data presentation
- ✅ Follows K8s provider pattern

---

### 2. Code Complexity

#### v2.0 Code Size

**Resource** (`auditd.go`):
```
Lines of Code: ~500
Responsibilities:
  - Schema definition
  - Source parameter validation (50 lines)
  - Dual-source storage management (100 lines)
  - Filesystem loading (150 lines)
  - Runtime loading (100 lines)
  - Logical AND evaluation (50 lines)
  - Rule merging/comparison (50 lines)
```

**Total**: ~500 lines in resource

---

#### v3.0 Code Size

**Connection Provider** (`connection/shared/audit_provider.go`):
```
Lines of Code: ~300
Responsibilities:
  - Capability detection (20 lines)
  - Dual-source storage (50 lines)
  - Filesystem loading (100 lines)
  - Runtime loading (80 lines)
  - Logical AND evaluation (50 lines)
```

**Resource** (`auditd.go`):
```
Lines of Code: ~100
Responsibilities:
  - Schema definition (unchanged)
  - Delegation to connection (50 lines)
```

**Total**: ~400 lines (300 provider + 100 resource)

**Comparison**:
- v2.0: 500 lines in single file
- v3.0: 400 lines split across two focused files
- 20% reduction + better organization

---

### 3. API & User Experience

#### v2.0 Query Syntax

```mql
# Default behavior (automatic)
auditd.rules.files
auditd.rules.syscalls

# Explicit source selection
auditd.rules(source: "filesystem").files
auditd.rules(source: "runtime").files
auditd.rules(source: "both").files

# Comparison queries
auditd.rules(source: "filesystem").files.length
auditd.rules(source: "runtime").files.length

# Validation
auditd.rules(source: "filesystem").files.length == 
  auditd.rules(source: "runtime").files.length
```

**Pros**:
- ✅ Explicit control at query level
- ✅ Easy to compare sources in single query

**Cons**:
- ⚠️ Introduces new parameter syntax
- ⚠️ Not consistent with other providers (K8s doesn't have source parameter)
- ⚠️ Verbose for common case

---

#### v3.0 Query Syntax

```mql
# Always the same syntax
auditd.rules.files
auditd.rules.syscalls
auditd.rules.controls

# Behavior determined by connection automatically
# - Live systems: dual-source
# - Non-live systems: filesystem only
```

**Connection-level control** (optional, for debugging):
```bash
# Default
cnquery shell os

# Force filesystem-only
cnquery shell os --audit-source=filesystem

# Force runtime-only
cnquery shell os --audit-source=runtime
```

**Pros**:
- ✅ Consistent query syntax
- ✅ No parameter complexity
- ✅ Aligned with K8s provider pattern
- ✅ Automatic behavior (no code changes)

**Cons**:
- ⚠️ Source comparison requires two connections (edge case)
- ⚠️ Less explicit in query (but more automatic)

---

### 4. Alignment with cnquery Patterns

#### K8s Provider Pattern Analysis

**How K8s handles multiple data sources**:

```go
// Connection interface
type Connection interface {
    Resources(kind, name, namespace) (*ResourceResult, error)
    ServerVersion() *version.Info
    // ... other methods
}

// Different connection types
type ApiConnection struct {
    // Talks to live cluster via REST API
}

type ManifestConnection struct {
    // Reads static YAML files
}

type AdmissionConnection struct {
    // Processes admission reviews
}

// Resource just calls connection
func (k *mqlK8sPod) pods() ([]any, error) {
    conn := k.MqlRuntime.Connection.(shared.Connection)
    result, err := conn.Resources("pods", "", "")
    // Convert and return
}
```

**Key Insight**: Resources don't know about data sources

---

#### v2.0 Alignment: ❌ Low

**Differences from K8s pattern**:
1. ❌ Resource knows about multiple data sources
2. ❌ Source selection at query level (not connection level)
3. ❌ Resource manages complex loading logic
4. ❌ Custom pattern not used elsewhere

**This is a new pattern**, not consistent with how other providers work.

---

#### v3.0 Alignment: ✅ High

**Similarities to K8s pattern**:
1. ✅ Connection abstracts data sources
2. ✅ Resource just queries connection
3. ✅ Source selection at connection level
4. ✅ Provider handles data fetching complexity

**Differences** (necessary):
- K8s: One connection type per data source
- Auditd: One connection type with dual sources

**Why different?**
- K8s: Connect to cluster OR manifest (mutually exclusive)
- Auditd: Need to check BOTH sources simultaneously (logical AND)

**Solution**: Connection provider pattern (inspired by K8s Discovery pattern)

---

### 5. Testing Strategy

#### v2.0 Testing

**Resource-Level Tests**:
```go
func TestAuditdRules_SourceParameter(t *testing.T) {
    // Test source parameter validation
    // Test source="filesystem"
    // Test source="runtime"
    // Test source="both"
    // Test logical AND
    // Mock filesystem and runtime in resource tests
}
```

**Challenges**:
- ⚠️ Complex mocking (need to mock both OS operations and command execution)
- ⚠️ Tests are integration tests (can't unit test source logic separately)
- ⚠️ Resource tests tightly coupled to loading implementation

---

#### v3.0 Testing

**Provider-Level Tests** (Unit):
```go
func TestAuditRuleProvider_FilesystemOnly(t *testing.T) {
    conn := mockConnection(capabilities: [])
    provider := NewAuditRuleProvider(conn)
    // Test filesystem loading
}

func TestAuditRuleProvider_DualSource(t *testing.T) {
    conn := mockConnection(capabilities: [Capability_RunCommand])
    provider := NewAuditRuleProvider(conn)
    // Test dual-source loading and merging
}
```

**Resource-Level Tests** (Unit):
```go
func TestAuditdRules_Delegation(t *testing.T) {
    runtime := mockRuntimeWithProvider(mockData)
    rules := &mqlAuditdRules{MqlRuntime: runtime}
    // Test that resource correctly delegates to provider
}
```

**Benefits**:
- ✅ Provider tests are pure unit tests
- ✅ Easy to mock connection for resource tests
- ✅ Clear test boundaries
- ✅ Fast test execution

---

### 6. Extensibility

#### v2.0 Extensibility

**Adding a new source** (e.g., systemd journal):
1. Add new source value: `"journal"`
2. Add new internal storage in resource
3. Add new loading method in resource
4. Update `loadBySource()` dispatcher
5. Update logical AND evaluation
6. Update all tests

**Impact**: Medium-High (touches many resource internals)

---

#### v3.0 Extensibility

**Adding a new source** (e.g., systemd journal):
1. Add loading method to provider: `loadJournalRules()`
2. Update provider's source selection logic
3. Add provider tests

**Impact**: Low (isolated to provider)

**Resource**: Unchanged (just calls `provider.GetRules()`)

---

### 7. Error Handling

#### v2.0 Error Handling

```go
// In resource
func (s *mqlAuditdRules) files(path, source string) ([]any, error) {
    switch source {
    case "filesystem":
        return s.getFilesystemFiles(path)
    case "runtime":
        return s.getRuntimeFiles(path)
    case "both":
        fsFiles, fsErr := s.getFilesystemFiles(path)
        rtFiles, rtErr := s.getRuntimeFiles(path)
        if fsErr != nil && rtErr != nil {
            return nil, fmt.Errorf("Failed from both: [fs: %v, rt: %v]", fsErr, rtErr)
        }
        // ... more error handling
    }
}
```

**Characteristics**:
- ⚠️ Error logic spread across resource methods
- ⚠️ Repeated in controls(), files(), syscalls()
- ⚠️ Hard to maintain consistency

---

#### v3.0 Error Handling

```go
// In provider (centralized)
func (p *AuditRuleProvider) GetRules(path string) (*AuditRuleData, error) {
    if !p.useRuntime {
        return p.getFilesystemRules(path)
    }
    return p.getBothRules(path)
}

func (p *AuditRuleProvider) getBothRules(path string) (*AuditRuleData, error) {
    // All error handling logic here
    // Consistent across all rule types
}

// In resource (simple delegation)
func (s *mqlAuditdRules) files(path string) ([]any, error) {
    data, err := s.getAuditRuleData(path)
    if err != nil {
        return nil, err  // Just pass through
    }
    return data.Files, nil
}
```

**Characteristics**:
- ✅ Error logic centralized in provider
- ✅ Single source of truth
- ✅ Easy to maintain

---

### 8. Performance

#### v2.0 Performance

**Loading Pattern**:
```go
// Resource manages loading
type mqlAuditdRulesInternal struct {
    filesystemLock   sync.Mutex
    filesystemLoaded bool
    filesystemData   struct { ... }
    
    runtimeLock   sync.Mutex
    runtimeLoaded bool
    runtimeData   struct { ... }
}

// Loaded separately on first access
func (s *mqlAuditdRules) files(path, source string) {
    s.filesystemLock.Lock()
    if !s.filesystemLoaded {
        // Load filesystem
    }
    s.filesystemLock.Unlock()
    
    // Same for runtime...
}
```

**Characteristics**:
- ✅ Lazy loading
- ⚠️ Sequential loading (filesystem, then runtime)
- ⚠️ Multiple locks

---

#### v3.0 Performance

**Loading Pattern**:
```go
// Provider manages loading
type AuditRuleProvider struct {
    filesystemOnce sync.Once
    filesystemData *AuditRuleData
    
    runtimeOnce sync.Once
    runtimeData *AuditRuleData
}

func (p *AuditRuleProvider) getBothRules(path string) {
    var wg sync.WaitGroup
    wg.Add(2)
    
    go func() {
        defer wg.Done()
        p.filesystemOnce.Do(func() {
            p.filesystemData, _ = p.loadFilesystemRules(path)
        })
    }()
    
    go func() {
        defer wg.Done()
        p.runtimeOnce.Do(func() {
            p.runtimeData, _ = p.loadRuntimeRules()
        })
    }()
    
    wg.Wait()
}
```

**Characteristics**:
- ✅ Lazy loading
- ✅ **Parallel loading** (filesystem + runtime simultaneously)
- ✅ Simple synchronization (once.Do)

**Performance Improvement**: ~2x faster on dual-source loads

---

### 9. Real-World Example

Let's trace a query through both architectures:

**Query**: `auditd.rules.files`

#### v2.0 Flow

```
User Query: auditd.rules.files
         │
         ▼
┌────────────────────────────┐
│ mqlAuditdRules.files()     │
│ ┌────────────────────────┐ │
│ │ Get source parameter   │ │
│ │ (default: "both")      │ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Check capability       │ │
│ │ hasRunCommand?         │ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Load filesystem        │ │
│ │ - Lock                 │ │
│ │ - Read files           │ │
│ │ - Parse                │ │
│ │ - Store in filesystemData│ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Load runtime           │ │
│ │ - Lock                 │ │
│ │ - Execute auditctl     │ │
│ │ - Parse                │ │
│ │ - Store in runtimeData │ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Merge & validate       │ │
│ │ - Compare sets         │ │
│ │ - Return if match      │ │
│ │ - FAILED if mismatch   │ │
│ └────────────────────────┘ │
└────────────────────────────┘
         │
         ▼
    File rules returned
```

**Steps**: 6 major operations in resource

---

#### v3.0 Flow

```
User Query: auditd.rules.files
         │
         ▼
┌────────────────────────────┐
│ mqlAuditdRules.files()     │
│ ┌────────────────────────┐ │
│ │ Get connection         │ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Call provider.GetRules()│ │
│ └────────────────────────┘ │
└────────────────────────────┘
         │
         ▼
┌────────────────────────────┐
│ AuditRuleProvider          │
│ ┌────────────────────────┐ │
│ │ Check useRuntime flag  │ │
│ │ (set at connection init)│ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Load both in parallel: │ │
│ │ ┌──────────┐┌─────────┐│ │
│ │ │Filesystem││Runtime  ││ │
│ │ │  load    ││  load   ││ │
│ │ └──────────┘└─────────┘│ │
│ └────────────────────────┘ │
│ ┌────────────────────────┐ │
│ │ Merge & validate       │ │
│ └────────────────────────┘ │
└────────────────────────────┘
         │
         ▼
┌────────────────────────────┐
│ mqlAuditdRules.files()     │
│ ┌────────────────────────┐ │
│ │ Extract data.Files     │ │
│ │ Return                 │ │
│ └────────────────────────┘ │
└────────────────────────────┘
         │
         ▼
    File rules returned
```

**Steps**: 2 operations in resource + provider handles complexity

**Comparison**:
- v2.0: Resource does everything
- v3.0: Resource delegates, provider does work
- v3.0: Parallel loading (faster)

---

### 10. Migration Path

#### If v2.0 Already Implemented

**Step 1**: Create Provider (No Breaking Changes)
```bash
# Create new file
providers/os/connection/shared/audit_provider.go

# Move logic from resource to provider
# - Copy loading methods
# - Copy merging logic
# - Copy error handling
```

**Step 2**: Update Connection
```go
// Add provider to connection
type ConnectionImpl struct {
    // existing fields...
    auditProvider *AuditRuleProvider
}

func (c *ConnectionImpl) AuditRuleProvider() *AuditRuleProvider {
    return c.auditProvider
}
```

**Step 3**: Simplify Resource
```go
// Remove source parameter from schema
// Remove dual-source internal storage
// Replace loading methods with delegation
func (s *mqlAuditdRules) files(path string) ([]any, error) {
    data, err := s.getAuditRuleData(path)
    if err != nil {
        return nil, err
    }
    return data.Files, nil
}
```

**Step 4**: Update Tests
```bash
# Move provider logic tests to provider_test.go
# Simplify resource tests to delegation tests
```

**Effort**: ~4 hours for experienced developer

---

#### If Starting Fresh

**Recommended**: Go directly to v3.0

**Advantages**:
- Cleaner from the start
- No refactoring needed
- Better aligned with cnquery patterns

---

## 11. Decision Matrix

### When to Choose v2.0

Choose v2.0 if:
- ❌ Cannot modify connection layer
- ❌ Need query-level source selection as primary API
- ❌ Want self-contained resource (no dependencies)

**Likelihood**: Low - these constraints don't apply to auditd

---

### When to Choose v3.0

Choose v3.0 if:
- ✅ Want alignment with cnquery/K8s patterns
- ✅ Value clean separation of concerns
- ✅ Want better testability
- ✅ Plan to extend to more sources in future
- ✅ Can modify connection layer (we can)
- ✅ Want automatic behavior (no query changes)

**Likelihood**: High - all of these apply

---

## 12. Recommendation

### **Recommendation: Implement v3.0 (Connection-Level Provider)**

### Rationale Summary

1. **Architecture**: Better aligned with established cnquery patterns (K8s provider)
2. **Maintainability**: Cleaner separation of concerns, easier to extend
3. **Performance**: Parallel loading improves speed
4. **Testing**: Better testability with isolated provider tests
5. **User Experience**: Transparent enhancement, no query changes needed
6. **Code Quality**: Less code, better organized

### Trade-offs Accepted

1. **Connection Changes**: Requires connection-level changes (acceptable)
2. **Less Explicit**: Source selection via connection options, not queries (acceptable for 99% of use cases)

### Implementation Priority

**If v2.0 already implemented**: Refactor to v3.0 (worth the effort)
**If starting fresh**: Implement v3.0 directly

---

## 13. Conclusion

Both approaches achieve the functional requirements, but v3.0 provides a superior architectural foundation that:
- Aligns with cnquery provider patterns
- Reduces resource complexity
- Improves performance
- Enhances testability
- Facilitates future extensions

The v3.0 approach is the **recommended path forward** for the `auditd.rules` extension.

---

**Document Version**: 1.0  
**Comparison**: v2.0 vs v3.0  
**Recommendation**: v3.0 (Connection-Level Provider)  
**Author**: AI Assistant  
**Date**: 2025-10-24



