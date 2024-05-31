// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package windows

import (
	"fmt"
)

// RegistryKeyItem represents a registry key item and its properties
const getRegistryKeyItemScript = `
$path = '%s'
$reg = Get-Item ('Registry::' + $path)
if ($reg -eq $null) {
  Write-Error "Could not find registry key"
  exit 1
}
$properties = @()
$reg.Property | ForEach-Object {
    $fetchKeyValue = $_
    if ("(default)".Equals($_)) { $fetchKeyValue = '' }
	$data = $(Get-ItemProperty ('Registry::' + $path)).$_;
	$kind = $reg.GetValueKind($fetchKeyValue);
	if ($kind -eq 7) {
      $data = $(Get-ItemProperty ('Registry::' + $path)) | Select-Object -ExpandProperty $_
	}
    $entry = New-Object psobject -Property @{
      "key" = $_
      "value" = New-Object psobject -Property @{
        "data" = $data;
        "kind" = $kind;
      }
    }
    $properties += $entry
}
ConvertTo-Json -Depth 3 -Compress $properties
`

func GetRegistryKeyItemScript(path string) string {
	return fmt.Sprintf(getRegistryKeyItemScript, path)
}

// getRegistryKeyChildItemsScript represents a registry key item and its children
const getRegistryKeyChildItemsScript = `
$path = '%s'
$children = Get-ChildItem -Path ('Registry::' + $path) -rec -ea SilentlyContinue

$properties = @()
$children | ForEach-Object {
  $entry = New-Object psobject -Property @{
    "name" = $_.PSChildName
    "path" = $_.Name
    "properties" = $_.Property
    "children" = $_.SubKeyCount
  }
  $properties += $entry
}
ConvertTo-Json -compress $properties
`

func GetRegistryKeyChildItemsScript(path string) string {
	return fmt.Sprintf(getRegistryKeyChildItemsScript, path)
}
