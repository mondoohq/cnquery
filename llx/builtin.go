package llx

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/lumi"
	"go.mondoo.io/mondoo/types"
)

type chunkHandler struct {
	Compiler func(types.Type, types.Type) (string, error)
	f        func(*LeiseExecutor, *RawData, *Chunk, int32) (*RawData, int32, error)
	Label    string
	Typ      types.Type
}

// BuiltinFunctions for all builtin types
var BuiltinFunctions map[types.Type]map[string]chunkHandler

func init() {
	BuiltinFunctions = map[types.Type]map[string]chunkHandler{
		types.Bool: {
			string("==" + types.Nil):                 {f: boolCmpNil, Label: "=="},
			string("!=" + types.Nil):                 {f: boolNotNil, Label: "!="},
			string("==" + types.Bool):                {f: boolCmpBool, Label: "=="},
			string("!=" + types.Bool):                {f: boolNotBool, Label: "!="},
			string("==" + types.String):              {f: boolCmpString, Label: "=="},
			string("!=" + types.String):              {f: boolNotString, Label: "!="},
			string("==" + types.Regex):               {f: boolCmpRegex, Label: "=="},
			string("!=" + types.Regex):               {f: boolNotRegex, Label: "!="},
			string("==" + types.Dict):                {f: boolCmpDict, Label: "=="},
			string("!=" + types.Dict):                {f: boolNotDict, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: boolCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: boolNotBoolarray, Label: "!="},
			string("==" + types.Array(types.String)): {f: boolCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): {f: boolNotStringarray, Label: "!="},
			string("&&" + types.Bool):                {f: boolAndBool, Label: "&&"},
			string("||" + types.Bool):                {f: boolOrBool, Label: "||"},
			string("&&" + types.Int):                 {f: boolAndInt, Label: "&&"},
			string("||" + types.Int):                 {f: boolOrInt, Label: "||"},
			string("&&" + types.Float):               {f: boolAndFloat, Label: "&&"},
			string("||" + types.Float):               {f: boolOrFloat, Label: "||"},
			string("&&" + types.String):              {f: boolAndString, Label: "&&"},
			string("||" + types.String):              {f: boolOrString, Label: "||"},
			string("&&" + types.Regex):               {f: boolAndRegex, Label: "&&"},
			string("||" + types.Regex):               {f: boolOrRegex, Label: "||"},
			string("&&" + types.Time):                {f: boolAndTime, Label: "&&"},
			string("||" + types.Time):                {f: boolOrTime, Label: "||"},
			string("&&" + types.ArrayLike):           {f: boolAndArray, Label: "&&"},
			string("||" + types.ArrayLike):           {f: boolOrArray, Label: "||"},
		},
		types.Int: {
			string("==" + types.Nil):                 {f: intCmpNil, Label: "=="},
			string("!=" + types.Nil):                 {f: intNotNil, Label: "!="},
			string("==" + types.Int):                 {f: intCmpInt, Label: "=="},
			string("!=" + types.Int):                 {f: intNotInt, Label: "!="},
			string("==" + types.String):              {f: intCmpString, Label: "=="},
			string("!=" + types.String):              {f: intNotString, Label: "!="},
			string("==" + types.Regex):               {f: intCmpRegex, Label: "=="},
			string("!=" + types.Regex):               {f: intNotRegex, Label: "!="},
			string("==" + types.Dict):                {f: intCmpDict, Label: "=="},
			string("!=" + types.Dict):                {f: intNotDict, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: intCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: intNotIntarray, Label: "!="},
			string("==" + types.Array(types.String)): {f: intCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): {f: intNotStringarray, Label: "!="},
			string("<" + types.Int):                  {f: intLTInt, Label: "<"},
			string("<=" + types.Int):                 {f: intLTEInt, Label: "<="},
			string(">" + types.Int):                  {f: intGTInt, Label: ">"},
			string(">=" + types.Int):                 {f: intGTEInt, Label: ">="},
			string("<" + types.Float):                {f: intLTFloat, Label: "<"},
			string("<=" + types.Float):               {f: intLTEFloat, Label: "<="},
			string(">" + types.Float):                {f: intGTFloat, Label: ">"},
			string(">=" + types.Float):               {f: intGTEFloat, Label: ">="},
			string("<" + types.String):               {f: intLTString, Label: "<"},
			string("<=" + types.String):              {f: intLTEString, Label: "<="},
			string(">" + types.String):               {f: intGTString, Label: ">"},
			string(">=" + types.String):              {f: intGTEString, Label: ">="},
			string("&&" + types.Bool):                {f: intAndBool, Label: "&&"},
			string("||" + types.Bool):                {f: intOrBool, Label: "||"},
			string("&&" + types.Int):                 {f: intAndInt, Label: "&&"},
			string("||" + types.Int):                 {f: intOrInt, Label: "||"},
			string("&&" + types.Float):               {f: intAndFloat, Label: "&&"},
			string("||" + types.Float):               {f: intOrFloat, Label: "||"},
			string("&&" + types.String):              {f: intAndString, Label: "&&"},
			string("||" + types.String):              {f: intOrString, Label: "||"},
			string("&&" + types.Regex):               {f: intAndRegex, Label: "&&"},
			string("||" + types.Regex):               {f: intOrRegex, Label: "||"},
			string("&&" + types.ArrayLike):           {f: intAndArray, Label: "&&"},
			string("||" + types.ArrayLike):           {f: intOrArray, Label: "||"},
			string("*" + types.Time):                 {f: intTimesTime, Label: "*", Typ: types.Time},
		},
		types.Float: {
			string("==" + types.Nil):                 {f: floatCmpNil, Label: "=="},
			string("!=" + types.Nil):                 {f: floatNotNil, Label: "!="},
			string("==" + types.Float):               {f: floatCmpFloat, Label: "=="},
			string("!=" + types.Float):               {f: floatNotFloat, Label: "!="},
			string("==" + types.String):              {f: floatCmpString, Label: "=="},
			string("!=" + types.String):              {f: floatNotString, Label: "!="},
			string("==" + types.Regex):               {f: floatCmpRegex, Label: "=="},
			string("!=" + types.Regex):               {f: floatNotRegex, Label: "!="},
			string("==" + types.Dict):                {f: floatCmpDict, Label: "=="},
			string("!=" + types.Dict):                {f: floatNotDict, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: floatCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: floatNotFloatarray, Label: "!="},
			string("==" + types.Array(types.String)): {f: floatCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): {f: floatNotStringarray, Label: "!="},
			string("<" + types.Float):                {f: floatLTFloat, Label: "<"},
			string("<=" + types.Float):               {f: floatLTEFloat, Label: "<="},
			string(">" + types.Float):                {f: floatGTFloat, Label: ">"},
			string(">=" + types.Float):               {f: floatGTEFloat, Label: ">="},
			string("<" + types.Int):                  {f: floatLTInt, Label: "<"},
			string("<=" + types.Int):                 {f: floatLTEInt, Label: "<="},
			string(">" + types.Int):                  {f: floatGTInt, Label: ">"},
			string(">=" + types.Int):                 {f: floatGTEInt, Label: ">="},
			string("<" + types.String):               {f: floatLTString, Label: "<"},
			string("<=" + types.String):              {f: floatLTEString, Label: "<="},
			string(">" + types.String):               {f: floatGTString, Label: ">"},
			string(">=" + types.String):              {f: floatGTEString, Label: ">="},
			string("&&" + types.Bool):                {f: floatAndBool, Label: "&&"},
			string("||" + types.Bool):                {f: floatOrBool, Label: "||"},
			string("&&" + types.Int):                 {f: floatAndInt, Label: "&&"},
			string("||" + types.Int):                 {f: floatOrInt, Label: "||"},
			string("&&" + types.Float):               {f: floatAndFloat, Label: "&&"},
			string("||" + types.Float):               {f: floatOrFloat, Label: "||"},
			string("&&" + types.String):              {f: floatAndString, Label: "&&"},
			string("||" + types.String):              {f: floatOrString, Label: "||"},
			string("&&" + types.Regex):               {f: floatAndRegex, Label: "&&"},
			string("||" + types.Regex):               {f: floatOrRegex, Label: "||"},
			string("&&" + types.ArrayLike):           {f: floatAndArray, Label: "&&"},
			string("||" + types.ArrayLike):           {f: floatOrArray, Label: "||"},
		},
		types.String: {
			string("==" + types.Nil):                 {f: stringCmpNil, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNil, Label: "!="},
			string("==" + types.String):              {f: stringCmpString, Label: "=="},
			string("!=" + types.String):              {f: stringNotString, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpRegex, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotRegex, Label: "!="},
			string("==" + types.Bool):                {f: stringCmpBool, Label: "=="},
			string("!=" + types.Bool):                {f: stringNotBool, Label: "!="},
			string("==" + types.Int):                 {f: stringCmpInt, Label: "=="},
			string("!=" + types.Int):                 {f: stringNotInt, Label: "!="},
			string("==" + types.Float):               {f: stringCmpFloat, Label: "=="},
			string("!=" + types.Float):               {f: stringNotFloat, Label: "!="},
			string("==" + types.Dict):                {f: stringCmpDict, Label: "=="},
			string("!=" + types.Dict):                {f: stringNotDict, Label: "!="},
			string("==" + types.Array(types.String)): {f: stringCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): {f: stringNotStringarray, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: stringCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: stringNotBoolarray, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: stringCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: stringNotIntarray, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: stringCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: stringNotFloatarray, Label: "!="},
			string("<" + types.String):               {f: stringLTString, Label: "<"},
			string("<=" + types.String):              {f: stringLTEString, Label: "<="},
			string(">" + types.String):               {f: stringGTString, Label: ">"},
			string(">=" + types.String):              {f: stringGTEString, Label: ">="},
			string("<" + types.Int):                  {f: stringLTInt, Label: "<"},
			string("<=" + types.Int):                 {f: stringLTEInt, Label: "<="},
			string(">" + types.Int):                  {f: stringGTInt, Label: ">"},
			string(">=" + types.Int):                 {f: stringGTEInt, Label: ">="},
			string("<" + types.Float):                {f: stringLTFloat, Label: "<"},
			string("<=" + types.Float):               {f: stringLTEFloat, Label: "<="},
			string(">" + types.Float):                {f: stringGTFloat, Label: ">"},
			string(">=" + types.Float):               {f: stringGTEFloat, Label: ">="},
			string("&&" + types.Bool):                {f: stringAndBool, Label: "&&"},
			string("||" + types.Bool):                {f: stringOrBool, Label: "||"},
			string("&&" + types.Int):                 {f: stringAndInt, Label: "&&"},
			string("||" + types.Int):                 {f: stringOrInt, Label: "||"},
			string("&&" + types.Float):               {f: stringAndFloat, Label: "&&"},
			string("||" + types.Float):               {f: stringOrFloat, Label: "||"},
			string("&&" + types.String):              {f: stringAndString, Label: "&&"},
			string("||" + types.String):              {f: stringOrString, Label: "||"},
			string("&&" + types.Regex):               {f: stringAndRegex, Label: "&&"},
			string("||" + types.Regex):               {f: stringOrRegex, Label: "||"},
			string("&&" + types.ArrayLike):           {f: stringAndArray, Label: "&&"},
			string("||" + types.ArrayLike):           {f: stringOrArray, Label: "||"},
			// fields
			string("contains" + types.String):              {f: stringContainsString, Label: "contains"},
			string("contains" + types.Array(types.String)): {f: stringContainsArrayString, Label: "contains"},
			string("downcase"):                             {f: stringDowncase, Label: "downcase"},
			string("length"):                               {f: stringLength, Label: "length"},
			string("lines"):                                {f: stringLines, Label: "lines"},
			string("split"):                                {f: stringSplit, Label: "split"},
		},
		types.Regex: {
			string("==" + types.Nil):                 {f: stringCmpNil, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNil, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpString, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotString, Label: "!="},
			string("==" + types.Bool):                {f: regexCmpBool, Label: "=="},
			string("!=" + types.Bool):                {f: regexNotBool, Label: "!="},
			string("==" + types.Int):                 {f: regexCmpInt, Label: "=="},
			string("!=" + types.Int):                 {f: regexNotInt, Label: "!="},
			string("==" + types.Float):               {f: regexCmpFloat, Label: "=="},
			string("!=" + types.Float):               {f: regexNotFloat, Label: "!="},
			string("==" + types.Dict):                {f: regexCmpDict, Label: "=="},
			string("!=" + types.Dict):                {f: regexNotDict, Label: "!="},
			string("==" + types.String):              {f: regexCmpString, Label: "=="},
			string("!=" + types.String):              {f: regexNotString, Label: "!="},
			string("==" + types.Array(types.Regex)):  {f: stringCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.Regex)):  {f: stringNotStringarray, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: regexCmpBoolarray, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: regexNotBoolarray, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: regexCmpIntarray, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: regexNotIntarray, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: regexCmpFloatarray, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: regexNotFloatarray, Label: "!="},
			string("==" + types.Array(types.String)): {f: regexCmpStringarray, Label: "=="},
			string("!=" + types.Array(types.String)): {f: regexNotStringarray, Label: "!="},
			string("&&" + types.Bool):                {f: regexAndBool, Label: "&&"},
			string("||" + types.Bool):                {f: regexOrBool, Label: "||"},
			string("&&" + types.Int):                 {f: regexAndInt, Label: "&&"},
			string("||" + types.Int):                 {f: regexOrInt, Label: "||"},
			string("&&" + types.Float):               {f: regexAndFloat, Label: "&&"},
			string("||" + types.Float):               {f: regexOrFloat, Label: "||"},
			string("&&" + types.String):              {f: regexAndString, Label: "&&"},
			string("||" + types.String):              {f: regexOrString, Label: "||"},
			string("&&" + types.Regex):               {f: regexAndRegex, Label: "&&"},
			string("||" + types.Regex):               {f: regexOrRegex, Label: "||"},
			string("&&" + types.ArrayLike):           {f: regexAndArray, Label: "&&"},
			string("||" + types.ArrayLike):           {f: regexOrArray, Label: "||"},
		},
		types.Time: {
			string("==" + types.Nil):  {f: timeCmpNil, Label: "=="},
			string("!=" + types.Nil):  {f: timeNotNil, Label: "!="},
			string("&&" + types.Bool): {f: timeAndBool, Label: "&&"},
			string("||" + types.Bool): {f: timeOrBool, Label: "||"},
			string("==" + types.Time): {f: timeCmpTime, Label: "=="},
			string("!=" + types.Time): {f: timeNotTime, Label: "!="},
			string("<" + types.Time):  {f: timeLTTime, Label: "<"},
			string("<=" + types.Time): {f: timeLTETime, Label: "<="},
			string(">" + types.Time):  {f: timeGTTime, Label: ">"},
			string(">=" + types.Time): {f: timeGTETime, Label: ">="},
			string("-" + types.Time):  {f: timeMinusTime, Label: "-"},
			string("*" + types.Int):   {f: timeTimesInt, Label: "*", Typ: types.Time},
			// fields
			string("seconds"): {f: timeSeconds, Label: "seconds"},
			string("minutes"): {f: timeMinutes, Label: "minutes"},
			string("hours"):   {f: timeHours, Label: "hours"},
			string("days"):    {f: timeDays, Label: "days"},
			string("unix"):    {f: timeUnix, Label: "unix"},
		},
		types.Dict: {
			"[]":                        {f: dictGetIndex},
			"length":                    {f: dictLength},
			"{}":                        {f: dictBlockCall},
			string("==" + types.Nil):    {f: dictCmpNil, Label: "=="},
			string("!=" + types.Nil):    {f: dictNotNil, Label: "!="},
			string("==" + types.Bool):   {f: dictCmpBool, Label: "=="},
			string("!=" + types.Bool):   {f: dictNotBool, Label: "!="},
			string("==" + types.Int):    {f: dictCmpInt, Label: "=="},
			string("!=" + types.Int):    {f: dictNotInt, Label: "!="},
			string("==" + types.Float):  {f: dictCmpFloat, Label: "=="},
			string("!=" + types.Float):  {f: dictNotFloat, Label: "!="},
			string("==" + types.Dict):   {f: dictCmpDict, Label: "=="},
			string("!=" + types.Dict):   {f: dictNotDict, Label: "!="},
			string("==" + types.String): {f: dictCmpString, Label: "=="},
			string("!=" + types.String): {f: dictNotString, Label: "!="},
			string("==" + types.Regex):  {f: dictCmpRegex, Label: "=="},
			string("!=" + types.Regex):  {f: dictNotRegex, Label: "!="},
		},
		types.ArrayLike: {
			"[]":     {f: arrayGetIndex},
			"{}":     {f: arrayBlockList},
			"length": {f: arrayLength},
			"where":  {f: arrayWhere},
			"==":     {Compiler: compileArrayOpArray("==")},
			"!=":     {Compiler: compileArrayOpArray("!=")},
			"&&":     {Compiler: compileLogicalArrayOp(types.ArrayLike, "&&")},
			"||":     {Compiler: compileLogicalArrayOp(types.ArrayLike, "||")},
			// special case: []<T> -- []<T>
			"<T>==<T>": {f: tarrayCmpTarray, Label: "=="},
			"<T>!=<T>": {f: tarrayNotTarray, Label: "!="},
			// logical operations []<T> -- K
			string(types.Any + "&&" + types.Bool):      {f: arrayAndBool, Label: "&&"},
			string(types.Any + "&&" + types.Int):       {f: arrayAndInt, Label: "&&"},
			string(types.Any + "&&" + types.Float):     {f: arrayAndFloat, Label: "&&"},
			string(types.Any + "&&" + types.String):    {f: arrayAndString, Label: "&&"},
			string(types.Any + "&&" + types.Regex):     {f: arrayAndRegex, Label: "&&"},
			string(types.Any + "&&" + types.ArrayLike): {f: arrayAndArray, Label: "&&"},
			string(types.Any + "||" + types.Bool):      {f: arrayOrBool, Label: "||"},
			string(types.Any + "||" + types.Int):       {f: arrayOrInt, Label: "||"},
			string(types.Any + "||" + types.Float):     {f: arrayOrFloat, Label: "||"},
			string(types.Any + "||" + types.String):    {f: arrayOrString, Label: "||"},
			string(types.Any + "||" + types.Regex):     {f: arrayOrRegex, Label: "||"},
			string(types.Any + "||" + types.ArrayLike): {f: arrayOrArray, Label: "||"},
			// []T -- []T
			string(types.Bool + "==" + types.Array(types.Bool)):     {f: boolarrayCmpBoolarray, Label: "=="},
			string(types.Bool + "!=" + types.Array(types.Bool)):     {f: boolarrayNotBoolarray, Label: "!="},
			string(types.Int + "==" + types.Array(types.Int)):       {f: intarrayCmpIntarray, Label: "=="},
			string(types.Int + "!=" + types.Array(types.Int)):       {f: intarrayNotIntarray, Label: "!="},
			string(types.Float + "==" + types.Array(types.Float)):   {f: floatarrayCmpFloatarray, Label: "=="},
			string(types.Float + "!=" + types.Array(types.Float)):   {f: floatarrayNotFloatarray, Label: "!="},
			string(types.String + "==" + types.Array(types.String)): {f: stringarrayCmpStringarray, Label: "=="},
			string(types.String + "!=" + types.Array(types.String)): {f: stringarrayNotStringarray, Label: "!="},
			string(types.Regex + "==" + types.Array(types.Regex)):   {f: stringarrayCmpStringarray, Label: "=="},
			string(types.Regex + "!=" + types.Array(types.Regex)):   {f: stringarrayNotStringarray, Label: "!="},
			// []T -- T
			string(types.Bool + "==" + types.Bool):     {f: boolarrayCmpBool, Label: "=="},
			string(types.Bool + "!=" + types.Bool):     {f: boolarrayNotBool, Label: "!="},
			string(types.Int + "==" + types.Int):       {f: intarrayCmpInt, Label: "=="},
			string(types.Int + "!=" + types.Int):       {f: intarrayNotInt, Label: "!="},
			string(types.Float + "==" + types.Float):   {f: floatarrayCmpFloat, Label: "=="},
			string(types.Float + "!=" + types.Float):   {f: floatarrayNotFloat, Label: "!="},
			string(types.String + "==" + types.String): {f: stringarrayCmpString, Label: "=="},
			string(types.String + "!=" + types.String): {f: stringarrayNotString, Label: "!="},
			string(types.Regex + "==" + types.Regex):   {f: stringarrayCmpString, Label: "=="},
			string(types.Regex + "!=" + types.Regex):   {f: stringarrayNotString, Label: "!="},
			// []string -- T
			string(types.String + "==" + types.Bool):  {f: stringarrayCmpBool, Label: "=="},
			string(types.String + "!=" + types.Bool):  {f: stringarrayNotBool, Label: "!="},
			string(types.String + "==" + types.Int):   {f: stringarrayCmpInt, Label: "=="},
			string(types.String + "!=" + types.Int):   {f: stringarrayNotInt, Label: "!="},
			string(types.String + "==" + types.Float): {f: stringarrayCmpFloat, Label: "=="},
			string(types.String + "!=" + types.Float): {f: stringarrayNotFloat, Label: "!="},
			// []T -- string
			string(types.Bool + "==" + types.String):  {f: boolarrayCmpString, Label: "=="},
			string(types.Bool + "!=" + types.String):  {f: boolarrayNotString, Label: "!="},
			string(types.Int + "==" + types.String):   {f: intarrayCmpString, Label: "=="},
			string(types.Int + "!=" + types.String):   {f: intarrayNotString, Label: "!="},
			string(types.Float + "==" + types.String): {f: floatarrayCmpString, Label: "=="},
			string(types.Float + "!=" + types.String): {f: floatarrayNotString, Label: "!="},
			// []T -- regex
			string(types.Bool + "==" + types.Regex):   {f: boolarrayCmpRegex, Label: "=="},
			string(types.Bool + "!=" + types.Regex):   {f: boolarrayNotRegex, Label: "!="},
			string(types.Int + "==" + types.Regex):    {f: intarrayCmpRegex, Label: "=="},
			string(types.Int + "!=" + types.Regex):    {f: intarrayNotRegex, Label: "!="},
			string(types.Float + "==" + types.Regex):  {f: floatarrayCmpRegex, Label: "=="},
			string(types.Float + "!=" + types.Regex):  {f: floatarrayNotRegex, Label: "!="},
			string(types.String + "==" + types.Regex): {f: stringarrayCmpRegex, Label: "=="},
			string(types.String + "!=" + types.Regex): {f: stringarrayNotRegex, Label: "!="},
		},
		types.MapLike: {
			"[]":     {f: mapGetIndex},
			"length": {f: mapLength},
			"{}":     {f: mapBlockCall},
		},
		types.ResourceLike: {
			"where":  {f: resourceWhere},
			"length": {f: resourceLength},
			"{}": {f: func(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
				return c.runBlock(bind, chunk.Function.Args[0], ref)
			}},
			// TODO: [#32] unique builtin fields that need a long-term support in LR
			string(types.Resource("parse") + ".date"): {f: resourceDate},
		},
	}
}

func runResourceFunction(c *LeiseExecutor, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	// ugh something is wrong here.... fix it later
	rr, ok := bind.Value.(lumi.ResourceType)
	if !ok {
		// TODO: can we get rid of this fmt call
		return nil, 0, fmt.Errorf("cannot cast resource to resource type: %+v", bind.Value)
	}

	info := rr.LumiResource()
	// resource := c.runtime.Registry.Resources[bind.Type]
	if info == nil {
		return nil, 0, errors.New("Cannot retrieve resource from the binding to run the raw function")
	}

	resource, ok := c.runtime.Registry.Resources[info.Name]
	if !ok || resource == nil {
		return nil, 0, errors.New("Cannot retrieve resource definition for resource '" + info.Name + "'")
	}

	// record this watcher on the executors watcher IDs
	wid := c.watcherUID(ref)
	// log.Debug().Str("wid", wid).Msg("exec> add watcher id ")
	c.watcherIds.Store(wid)

	// watch this field in the resource
	err := c.runtime.WatchAndUpdate(rr, chunk.Id, wid, func(fieldData interface{}, fieldError error) {
		data := &RawData{
			Type:  types.Type(resource.Fields[chunk.Id].Type),
			Value: fieldData,
			Error: fieldError,
		}

		c.cache.Store(ref, &stepCache{
			Result: data,
		})

		codeID, ok := c.callbackPoints[ref]
		if ok {
			c.callback(&RawResult{Data: data, CodeID: codeID})
		}

		if fieldError != nil {
			c.triggerChainError(ref, fieldError)
		}

		c.triggerChain(ref)
	})

	if err != nil {
		if _, ok := err.(lumi.NotReadyError); !ok {
			// TODO: Deduplicate storage between cache and resource storage
			// This will take some work, but clearly we don't need both

			info.Cache.Store(chunk.Id, &lumi.CacheEntry{
				Timestamp: time.Now().Unix(),
				Valid:     true,
				Error:     err,
			})

			c.cache.Store(ref, &stepCache{
				Result: &RawData{
					Type:  types.Type(resource.Fields[chunk.Id].Type),
					Value: nil,
					Error: err,
				},
			})
		}
	}

	// we are done executing this chain
	return nil, 0, err
}

// BuiltinFunction provides the handler for this type's function
func BuiltinFunction(typ types.Type, name string) (*chunkHandler, error) {
	h, ok := BuiltinFunctions[typ.Underlying()]
	if !ok {
		return nil, errors.New("cannot find functions for type '" + typ.Label() + "' (called '" + name + "')")
	}
	fh, ok := h[name]
	if !ok {
		return nil, errors.New("cannot find function '" + name + "' for type '" + typ.Label() + "'")
	}
	return &fh, nil
}

// this is called for objects that call a function
func (c *LeiseExecutor) runBoundFunction(bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	log.Debug().Int32("ref", ref).Str("id", chunk.Id).Msg("exec> run bound function")

	fh, err := BuiltinFunction(bind.Type, chunk.Id)
	if err == nil {
		res, dref, err := fh.f(c, bind, chunk, ref)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		}
		return res, dref, err
	}

	if bind.Type.IsResource() {
		return runResourceFunction(c, bind, chunk, ref)
	}
	return nil, 0, err
}
