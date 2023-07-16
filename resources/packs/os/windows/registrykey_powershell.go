package windows

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
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

type keyKindRaw struct {
	Kind int
	Data interface{}
}

func (k *RegistryKeyValue) UnmarshalJSON(b []byte) error {
	var raw keyKindRaw

	// try to unmarshal the type
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}
	k.Kind = raw.Kind

	if raw.Data == nil {
		return nil
	}

	// see https://docs.microsoft.com/en-us/powershell/scripting/samples/working-with-registry-entries?view=powershell-7
	switch raw.Kind {
	case NONE:
		// ignore
	case SZ: // Any string value
		value, ok := raw.Data.(string)
		if !ok {
			return fmt.Errorf("registry key value is not a string: %v", raw.Data)
		}
		k.String = value
	case EXPAND_SZ: // A string that can contain environment variables that are dynamically expanded
		value, ok := raw.Data.(string)
		if !ok {
			return fmt.Errorf("registry key value is not a string: %v", raw.Data)
		}
		k.String = value
	case BINARY: // Binary data
		rawData, ok := raw.Data.([]interface{})
		if !ok {
			return fmt.Errorf("registry key value is not a byte array: %v", raw.Data)
		}
		data := make([]byte, len(rawData))
		for i, v := range rawData {
			val, ok := v.(float64)
			if !ok {
				return fmt.Errorf("registry key value is not a byte array: %v", raw.Data)
			}
			data[i] = byte(val)
		}
		k.Binary = data
	case DWORD: // A number that is a valid UInt32
		data, ok := raw.Data.(float64)
		if !ok {
			return fmt.Errorf("registry key value is not a number: %v", raw.Data)
		}
		number := int64(data)
		// string fallback
		k.Number = number
		k.String = strconv.FormatInt(number, 10)
	case DWORD_BIG_ENDIAN:
		log.Warn().Msg("DWORD_BIG_ENDIAN for registry key is not supported")
	case LINK:
		log.Warn().Msg("LINK for registry key is not supported")
	case MULTI_SZ: // A multiline string
		switch value := raw.Data.(type) {
		case string:
			k.String = value
			if value != "" {
				k.MultiString = []string{value}
			}
		case []interface{}:
			if len(value) > 0 {
				var multiString []string
				for _, v := range value {
					multiString = append(multiString, v.(string))
				}
				// NOTE: this is to be consistent with the output before we moved to multi-datatype support for registry keys
				k.String = strings.Join(multiString, " ")
				k.MultiString = multiString
			}
		}
	case RESOURCE_LIST:
		log.Warn().Msg("RESOURCE_LIST for registry key is not supported")
	case FULL_RESOURCE_DESCRIPTOR:
		log.Warn().Msg("FULL_RESOURCE_DESCRIPTOR for registry key is not supported")
	case RESOURCE_REQUIREMENTS_LIST:
		log.Warn().Msg("RESOURCE_REQUIREMENTS_LIST for registry key is not supported")
	case QWORD: // 8 bytes of binary data
		f, ok := raw.Data.(float64)
		if !ok {
			return fmt.Errorf("registry key value is not a number: %v", raw.Data)
		}
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
		k.Binary = buf
	}
	return nil
}

func ParsePowershellRegistryKeyItems(r io.Reader) ([]RegistryKeyItem, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var items []RegistryKeyItem
	err = json.Unmarshal(data, &items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func ParsePowershellRegistryKeyChildren(r io.Reader) ([]RegistryKeyChild, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var children []RegistryKeyChild
	err = json.Unmarshal(data, &children)
	if err != nil {
		return nil, err
	}

	return children, nil
}
