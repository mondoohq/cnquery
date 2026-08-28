# ADR: cnspec Windows Driver Resource

**Status:** Proposed
**Date:** 2026-07-20

---

## Context

The `windows` namespace covers hotfixes, scheduled tasks, optional features, and
several hardening controls, but there is no way to enumerate the device and
kernel-mode drivers installed on a host. (osquery exposes this via its Windows-only
`drivers` table.)

Drivers are a high-value security surface:

- **Unsigned or improperly-signed drivers** — a driver that loads into kernel space
  without a valid signature, or one signed by an unexpected signer, is a strong
  tampering / rootkit indicator. CIS and STIG baselines require driver signing to be
  enforced.
- **BYOVD ("bring your own vulnerable driver")** — attackers ship a legitimately
  signed but exploitable driver (e.g. `RTCore64.sys`, `dbutil_2_3.sys`) to gain
  kernel primitives and disable EDR. Detecting these requires enumerating driver
  names, versions, and signers and matching them against blocklists such as the
  Microsoft Vulnerable Driver Blocklist. Because a BYOVD driver only grants
  primitives while it is *running*, loaded state matters as much as presence.
- **Third-party driver inventory** — provider, class, and version enable patch and
  supply-chain audits (which vendors' drivers are present, at what versions).

A `windows.driver` resource lets policies express these checks directly in MQL, e.g.
`windows.drivers.where(signed == false)` or matching a driver name against a
known-vulnerable list.

"Drivers" here means **Windows device/kernel-mode drivers only**. Linux kernel
modules (`kernel.module`) and macOS kernel extensions are separate concepts modeled
elsewhere.

## Decision

Attach a `drivers()` list field to the existing **`windows`** namespace resource —
the natural parent, alongside `hotfixes()` and `scheduledTasks()` — backed by a new
**`private windows.driver`** element resource.

`private` matches the sibling element resources (`windows.scheduledTask`,
`windows.serverFeature`, `windows.optionalFeature`): the element is only reachable
through its parent list, never constructed directly by users.

Fields (camelCase, mapped from the WMI/CIM property names):

`name`, `displayName`, `description`, `class`, `provider`, `version`, `date`,
`signed`, `signer`, `path`, plus the loaded-state enrichment fields `started`,
`startMode`, `status`, and `deviceId`.

`@defaults("name version signed")` selects the fields a bare `windows.drivers` print
shows, mirroring the default selection on `windows.hotfix` and
`windows.scheduledTask`.

`windows.drivers` resolves only on Windows assets; the `windows` namespace itself is
Windows-only, so on other platforms it is simply not selected.

## Data Gathering

### Options compared

| Source | Command | Admin? | Coverage | Signing data |
|--------|---------|--------|----------|--------------|
| **`Win32_PnPSignedDriver`** (WMI/CIM) | `Get-CimInstance Win32_PnPSignedDriver` | **No** | All PnP device drivers (inbox + third-party) | `IsSigned` + `Signer` |
| `Win32_SystemDriver` (WMI/CIM) | `Get-CimInstance Win32_SystemDriver` | No | Kernel *service* drivers only (`CurrentControlSet\Services`) | none, but has `State`/`StartMode`/`Status`/`PathName` |
| `Get-WindowsDriver -Online` (DISM) | DISM cmdlet | **Yes (admin)** | Inbox + third-party OEM driver packages | signer/version, richer package metadata |
| `driverquery /v /fo csv` | legacy CLI | No | Basic list, no signing detail | none |

### Primary: `Win32_PnPSignedDriver`

Chosen as the primary source because it gives the broadest coverage **without
requiring administrator privileges** and carries the two security-critical fields
(`IsSigned`, `Signer`) directly. It works over WinRM against a standard-user session,
which matters for fleet scanning.

`Get-WindowsDriver -Online` (DISM) offers richer driver-package metadata, but it
**requires admin** and only covers driver *packages* (not every loaded device). It is
rejected as the primary to keep the resource usable without elevation; it could be
added later as optional enrichment behind an admin check.

`driverquery` is not needed: `Win32_PnPSignedDriver` is available on every supported
Windows release and requires no elevation, so there is no realistic connection where
it is unavailable but `driverquery` would succeed. Omitting it keeps a single code
path.

### Enrichment: `Win32_SystemDriver`

`Win32_PnPSignedDriver` does not report whether a driver is currently **loaded**. For
kernel-service drivers we join `Win32_SystemDriver` (keyed by driver `Name`) to
populate `started`, `startMode`, and `status`. This lets policies distinguish an
installed-but-stopped driver from a live loaded one — relevant for BYOVD, where the
attacker's driver must be running. The join is best-effort: PnP-only drivers with no
matching service simply have null state fields.

### Output

One PowerShell call per source, `... | Select-Object <props> | ConvertTo-Json`,
mirroring the existing `Get-HotFix | … | ConvertTo-Json` pattern. `DriverDate` is
normalized to ISO-8601 in PowerShell via a calculated property, so the Go side
receives a stable `time` string instead of a CIM `/Date(...)/` form.

Note the `ConvertTo-Json` single-object-vs-array quirk: PowerShell emits a bare JSON
object when a source returns exactly one row and an array otherwise. The parser must
handle both (the same issue the packages parser already documents).

## Resource Schema

`.lr` additions (`providers/os/resources/os.lr`):

```lr
windows {
  // …existing (computerInfo, hotfixes, serverFeatures, optionalFeatures,
  //  scheduledTasks, deviceGuard, exploitProtection, smartScreen)…

  // Installed device and kernel-mode drivers
  drivers() []windows.driver
}

// A single installed Windows driver
//
// Reported by `Get-CimInstance Win32_PnPSignedDriver` and enriched with loaded
// state from `Win32_SystemDriver`. Use for driver-signing posture and
// vulnerable-driver (BYOVD) detection, for example
// `windows.drivers.where(signed == false)`.
private windows.driver @defaults("name version signed") {
  // Driver name (Win32_PnPSignedDriver.DriverName / Name)
  name string
  // Friendly display name (FriendlyName)
  displayName string
  // Driver description (Description)
  description string
  // Device class (DeviceClass), e.g. "Net", "System", "Display"
  class string
  // Driver provider (DriverProviderName)
  provider string
  // Driver version (DriverVersion)
  version string
  // Driver build date (DriverDate), normalized to a timestamp
  date time
  // Whether the driver is digitally signed (IsSigned)
  signed bool
  // Signer of the driver, if signed (Signer)
  signer string
  // Backing file / INF path (InfName or Win32_SystemDriver.PathName)
  path string
  // Whether the driver is currently loaded/running (Win32_SystemDriver.Started); null if not a service driver
  started bool
  // Start mode: Boot, System, Auto, Manual, Disabled (Win32_SystemDriver.StartMode); null if unavailable
  startMode string
  // Operational status, e.g. "OK" (Win32_SystemDriver.Status); null if unavailable
  status string
  // Device identifier (Win32_PnPSignedDriver.DeviceID)
  deviceId string
}
```

### Field mapping

| MQL field | WMI/CIM source property | Type | Notes |
|-----------|-------------------------|------|-------|
| `name` | `DriverName` (fallback `Name`) | string | stable id component |
| `displayName` | `FriendlyName` | string | |
| `description` | `Description` | string | |
| `class` | `DeviceClass` | string | |
| `provider` | `DriverProviderName` | string | |
| `version` | `DriverVersion` | string | |
| `date` | `DriverDate` | time | normalized to ISO-8601 in PowerShell |
| `signed` | `IsSigned` | bool | security-critical |
| `signer` | `Signer` | string | may be empty when unsigned |
| `path` | `InfName` / `PathName` | string | |
| `started` | `Win32_SystemDriver.Started` | bool | nullable (join) |
| `startMode` | `Win32_SystemDriver.StartMode` | string | nullable (join) |
| `status` | `Win32_SystemDriver.Status` | string | nullable (join) |
| `deviceId` | `DeviceID` | string | uniqueness key |

## Transport Compatibility

| Transport | Method | Notes |
|-----------|--------|-------|
| Local (Windows) | `conn.RunCommand("powershell …")` | works as standard user |
| WinRM | remote PowerShell | works as standard user; primary fleet path |
| Container image / offline / mounted image | unsupported | resource requires a live command channel; there is no registry/file fallback (CIM providers are not readable offline). `windows.drivers` errors on filesystem-only connections, consistent with `scheduledTasks()`. |

## Verification

- **Unit test** (`windows_driver_test.go`): drive `drivers()` through a mock
  connection whose recorded output is a captured
  `Get-CimInstance Win32_PnPSignedDriver | … | ConvertTo-Json` fixture (add the
  command to a `testdata/windows_*.toml` mock, mirroring
  `windows_scheduledtask_test.go`). Assert field mapping, the
  single-object-vs-array path, and that unsigned rows yield `signed == false`.
- **Interactive:**
  - `cnquery run os -c 'windows.drivers.where(signed == false)'`
  - `cnquery run os -c 'windows.drivers.where(class == "Net").all(signed)'`
- **Content check** (CIS-style, all drivers signed):
  ```yaml
  mql: windows.drivers.all(signed == true)
  ```

## Implementation

**Target Go file:** `providers/os/resources/windows_driver.go` (new), following
`windows_scheduledtask.go`. The PowerShell command constants and parser live
alongside it (or under `providers/os/resources/windows/`, matching
`windows.SCHEDULED_TASKS`).

Generated getter on `mqlWindows`:

```go
func (w *mqlWindows) drivers() ([]any, error)
```

Sketch (PnP primary; `Win32_SystemDriver` join elided for brevity):

```go
const WINDOWS_QUERY_DRIVERS = `Get-CimInstance -ClassName Win32_PnPSignedDriver |
Select-Object DeviceName, FriendlyName, Description, DeviceClass, DriverProviderName,
  DriverVersion, @{N='DriverDate';E={ if ($_.DriverDate) { $_.DriverDate.ToString('o') } }},
  IsSigned, Signer, InfName, DeviceID, DriverName |
ConvertTo-Json -Depth 3`

type psDriver struct {
	DriverName, DeviceName, FriendlyName, Description string
	DeviceClass, DriverProviderName, DriverVersion    string
	DriverDate, Signer, InfName, DeviceID             string
	IsSigned                                          bool
}

func (w *mqlWindows) drivers() ([]any, error) {
	conn := w.MqlRuntime.Connection.(shared.Connection)
	cmd, err := conn.RunCommand(powershell.Encode(WINDOWS_QUERY_DRIVERS))
	if err != nil {
		return nil, err
	}
	if cmd.ExitStatus != 0 {
		stderr, _ := io.ReadAll(cmd.Stderr)
		return nil, errors.New("failed to retrieve drivers: " + string(stderr))
	}
	data, err := io.ReadAll(cmd.Stdout)
	if err != nil {
		return nil, err
	}
	var drivers []psDriver
	// ConvertTo-Json emits a bare object for a single row; normalize to an array.
	if err := json.Unmarshal(bytes.TrimSpace(data), &drivers); err != nil {
		var one psDriver
		if err2 := json.Unmarshal(bytes.TrimSpace(data), &one); err2 != nil {
			return nil, err
		}
		drivers = []psDriver{one}
	}

	res := make([]any, 0, len(drivers))
	for _, d := range drivers {
		name := d.DriverName
		if name == "" {
			name = d.DeviceName
		}
		r, err := CreateResource(w.MqlRuntime, "windows.driver", map[string]*llx.RawData{
			"__id":        llx.StringData(d.DeviceID + "|" + name),
			"name":        llx.StringData(name),
			"displayName": llx.StringData(d.FriendlyName),
			"description": llx.StringData(d.Description),
			"class":       llx.StringData(d.DeviceClass),
			"provider":    llx.StringData(d.DriverProviderName),
			"version":     llx.StringData(d.DriverVersion),
			"date":        llx.TimeDataPtr(parseRFC3339(d.DriverDate)),
			"signed":      llx.BoolData(d.IsSigned),
			"signer":      llx.StringData(d.Signer),
			"path":        llx.StringData(d.InfName),
			"deviceId":    llx.StringData(d.DeviceID),
			// started/startMode/status filled from Win32_SystemDriver join, else null
		})
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}
```

The `Win32_SystemDriver` enrichment is a second `Get-CimInstance` keyed by `Name`,
merged into a `map[string]psSystemDriver` before building resources; drivers with no
matching service keep null state fields.

**Build:** edit `os.lr` → `make providers/build/os` (regenerates `os.lr.go`,
`os.lr.json`, and the manifest) → implement `drivers()` and the `windows.driver`
getters in `windows_driver.go`.
