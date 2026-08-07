// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"reflect"
	"strings"

	"github.com/Snowflake-Labs/terraform-provider-snowflake/pkg/sdk"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/snowflake/connection"
)

type mqlSnowflakeFileFormatInternal struct {
	cacheOwner string
}

// fileFormatTypePrefixes maps a file format type to the Go field-name prefix
// the SDK uses for that type's options. FileFormatTypeOptions is one flat
// struct holding the options of every type, so the prefix is what separates
// the options that apply to a given format from the ones that do not.
var fileFormatTypePrefixes = map[sdk.FileFormatType]string{
	sdk.FileFormatTypeCSV:     "CSV",
	sdk.FileFormatTypeJSON:    "JSON",
	sdk.FileFormatTypeAvro:    "Avro",
	sdk.FileFormatTypeORC:     "ORC",
	sdk.FileFormatTypeParquet: "Parquet",
	sdk.FileFormatTypeXML:     "XML",
}

func (r *mqlSnowflakeAccount) fileFormats() ([]any, error) {
	return listSnowflakeFileFormats(r.MqlRuntime, &sdk.In{Account: sdk.Bool(true)})
}

func (r *mqlSnowflakeDatabase) fileFormats() ([]any, error) {
	return listSnowflakeFileFormats(r.MqlRuntime, &sdk.In{Database: sdk.NewAccountObjectIdentifier(r.Name.Data)})
}

// listSnowflakeFileFormats fetches file formats within the given scope
// (account-wide or a single database) and maps them to resources.
func listSnowflakeFileFormats(runtime *plugin.Runtime, in *sdk.In) ([]any, error) {
	conn := runtime.Connection.(*connection.SnowflakeConnection)
	formats, err := conn.Client().FileFormats.Show(context.Background(), &sdk.ShowFileFormatsOptions{In: in})
	if err != nil {
		return nil, err
	}

	list := make([]any, 0, len(formats))
	for i := range formats {
		mqlFormat, err := newMqlSnowflakeFileFormat(runtime, formats[i])
		if err != nil {
			return nil, err
		}
		list = append(list, mqlFormat)
	}
	return list, nil
}

func newMqlSnowflakeFileFormat(runtime *plugin.Runtime, format sdk.FileFormat) (*mqlSnowflakeFileFormat, error) {
	databaseName := format.Name.DatabaseName()
	schemaName := format.Name.SchemaName()
	name := format.Name.Name()

	comment := ""
	if format.Options.Comment != nil {
		comment = *format.Options.Comment
	}
	if format.Comment != "" {
		comment = format.Comment
	}

	r, err := CreateResource(runtime, "snowflake.fileFormat", map[string]*llx.RawData{
		"__id":          llx.StringData(snowflakeSchemaObjectID(databaseName, schemaName, name)),
		"name":          llx.StringData(name),
		"databaseName":  llx.StringData(databaseName),
		"schemaName":    llx.StringData(schemaName),
		"ownerRoleType": llx.StringData(format.OwnerRoleType),
		"type":          llx.StringData(string(format.Type)),
		"options":       llx.DictData(fileFormatOptionsToDict(format.Type, format.Options)),
		"comment":       llx.StringData(comment),
		"createdAt":     llx.TimeData(format.CreatedOn),
	})
	if err != nil {
		return nil, err
	}
	mqlFormat := r.(*mqlSnowflakeFileFormat)
	mqlFormat.cacheOwner = format.Owner
	return mqlFormat, nil
}

// fileFormatOptionsToDict renders the options that apply to a file format's own
// type, keyed by the Snowflake option name taken from the SDK's sql struct tag.
// Options belonging to other types, and options left unset on this format, are
// omitted, so the result holds only what is actually configured.
//
// Reflection is used rather than a fixed field list so that options added to a
// future SDK release surface automatically instead of silently going missing.
// Values are reduced to JSON-native types (string, bool, int64, and slices of
// those); anything else is skipped, because a dict carrying a non-native value
// fails at query time rather than at compile time.
func fileFormatOptionsToDict(formatType sdk.FileFormatType, opts sdk.FileFormatTypeOptions) map[string]any {
	out := map[string]any{}
	prefix, ok := fileFormatTypePrefixes[formatType]
	if !ok {
		return out
	}

	v := reflect.ValueOf(opts)
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !strings.HasPrefix(field.Name, prefix) {
			continue
		}
		key := sqlTagName(field.Tag.Get("sql"))
		if key == "" {
			continue
		}
		if value, ok := jsonNativeValue(v.Field(i)); ok {
			out[key] = value
		}
	}
	return out
}

// sqlTagName returns the Snowflake option name carried in an SDK sql struct
// tag. Every such tag currently holds a bare option name (for example
// "SKIP_HEADER"), but the neighbouring ddl tags in the same structs are
// comma-separated lists, so anything after a comma is dropped rather than
// trusted: a future SDK release that adopts that convention here would
// otherwise turn the option name into a wrong dict key without failing.
func sqlTagName(tag string) string {
	if i := strings.IndexByte(tag, ','); i >= 0 {
		tag = tag[:i]
	}
	return strings.TrimSpace(tag)
}

// jsonNativeValue reduces an SDK option value to a type a dict can carry. It
// reports false for an unset pointer and for any value it cannot represent
// natively, so the caller omits the key entirely rather than storing a null.
func jsonNativeValue(v reflect.Value) (any, bool) {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), true
	case reflect.Bool:
		return v.Bool(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Slice:
		// The only slice option is NULL_IF, a list of single-field NullString
		// structs; flatten it to the strings it wraps.
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			elem := v.Index(i)
			if elem.Kind() == reflect.Struct && elem.NumField() == 1 && elem.Field(0).Kind() == reflect.String {
				out = append(out, elem.Field(0).String())
				continue
			}
			nested, ok := jsonNativeValue(elem)
			if !ok {
				return nil, false
			}
			out = append(out, nested)
		}
		return out, true
	}
	return nil, false
}

func (r *mqlSnowflakeFileFormat) database() (*mqlSnowflakeDatabase, error) {
	return resolveDatabaseRef(r.MqlRuntime, r.DatabaseName.Data, &r.Database)
}

func (r *mqlSnowflakeFileFormat) schema() (*mqlSnowflakeSchema, error) {
	return resolveSchemaRef(r.MqlRuntime, r.DatabaseName.Data, r.SchemaName.Data, &r.Schema)
}

func (r *mqlSnowflakeFileFormat) ownerRole() (*mqlSnowflakeRole, error) {
	return resolveOwnerRole(r.MqlRuntime, r.cacheOwner, &r.OwnerRole)
}

// resolveFileFormatRef resolves a fully qualified file format name (the form
// Snowflake reports in SHOW output) to its file format. A name that cannot be
// parsed, or a format the caller cannot see, resolves to null rather than
// failing the surrounding query.
func resolveFileFormatRef(runtime *plugin.Runtime, fqn string, field *plugin.TValue[*mqlSnowflakeFileFormat]) (*mqlSnowflakeFileFormat, error) {
	if fqn == "" {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	id, err := sdk.ParseSchemaObjectIdentifier(fqn)
	if err != nil {
		field.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}

	conn := runtime.Connection.(*connection.SnowflakeConnection)
	formats, err := conn.Client().FileFormats.Show(context.Background(), &sdk.ShowFileFormatsOptions{
		Like: &sdk.Like{Pattern: sdk.String(id.Name())},
		In:   &sdk.In{Schema: sdk.NewDatabaseObjectIdentifier(id.DatabaseName(), id.SchemaName())},
	})
	if err != nil {
		return nil, err
	}
	for i := range formats {
		if formats[i].Name.Name() == id.Name() && formats[i].Name.DatabaseName() == id.DatabaseName() &&
			formats[i].Name.SchemaName() == id.SchemaName() {
			return newMqlSnowflakeFileFormat(runtime, formats[i])
		}
	}
	field.State = plugin.StateIsSet | plugin.StateIsNull
	return nil, nil
}
