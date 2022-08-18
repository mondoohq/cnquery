package windows

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"strconv"

	"github.com/rs/zerolog/log"
)

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
    $entry = New-Object psobject -Property @{
      "key" = $_
      "value" = New-Object psobject -Property @{
        "data" =  $(Get-ItemProperty ('Registry::' + $path)).$_;
        "kind"  = $reg.GetValueKind($fetchKeyValue);
      }
    }
    $properties += $entry
}
ConvertTo-Json -Compress $properties
`

func GetRegistryKeyItemScript(path string) string {
	return fmt.Sprintf(getRegistryKeyItemScript, path)
}

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

// derrived from "golang.org/x/sys/windows/registry"
// see https://github.com/golang/sys/blob/master/windows/registry/value.go#L17-L31
const (
	NONE                       = 0
	SZ                         = 1
	EXPAND_SZ                  = 2
	BINARY                     = 3
	DWORD                      = 4
	DWORD_BIG_ENDIAN           = 5
	LINK                       = 6
	MULTI_SZ                   = 7
	RESOURCE_LIST              = 8
	FULL_RESOURCE_DESCRIPTOR   = 9
	RESOURCE_REQUIREMENTS_LIST = 10
	QWORD                      = 11
)

type RegistryKeyItem struct {
	Key   string
	Value RegistryKeyValue
}

type RegistryKeyValue struct {
	Kind   int
	Binary []byte
	Number int64
	String string
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
		k.String = raw.Data.(string)
	case EXPAND_SZ: // A string that can contain environment variables that are dynamically expanded
		k.String = raw.Data.(string)
	case BINARY: // Binary data
		k.Binary = []byte(raw.Data.(string))
	case DWORD: // A number that is a valid UInt32
		data := raw.Data.(float64)
		number := int64(data)
		// string fallback
		k.Number = number
		k.String = strconv.FormatInt(number, 10)
	case DWORD_BIG_ENDIAN:
		log.Warn().Msg("DWORD_BIG_ENDIAN for registry key is not supported")
	case LINK:
		log.Warn().Msg("LINK for registry key is not supported")
	case MULTI_SZ: // A multiline string
		k.String = raw.Data.(string)
	case RESOURCE_LIST:
		log.Warn().Msg("RESOURCE_LIST for registry key is not supported")
	case FULL_RESOURCE_DESCRIPTOR:
		log.Warn().Msg("FULL_RESOURCE_DESCRIPTOR for registry key is not supported")
	case RESOURCE_REQUIREMENTS_LIST:
		log.Warn().Msg("RESOURCE_REQUIREMENTS_LIST for registry key is not supported")
	case QWORD: // 8 bytes of binary data
		f := raw.Data.(float64)
		buf := make([]byte, 8)
		binary.LittleEndian.PutUint64(buf[:], math.Float64bits(f))
		k.Binary = buf
	}
	return nil
}

func (k RegistryKeyItem) GetValue() string {
	return k.Value.String
}

type RegistryKeyChild struct {
	Name       string
	Path       string
	Properties []string
	Children   int
}

func ParseRegistryKeyItems(r io.Reader) ([]RegistryKeyItem, error) {
	data, err := ioutil.ReadAll(r)
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

func ParseRegistryKeyChildren(r io.Reader) ([]RegistryKeyChild, error) {
	data, err := ioutil.ReadAll(r)
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
