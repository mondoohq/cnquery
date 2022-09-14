package llx

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/types"
)

type chunkHandlerV2 struct {
	Compiler func(types.Type, types.Type) (string, error)
	f        func(*blockExecutor, *RawData, *Chunk, uint64) (*RawData, uint64, error)
	Label    string
	Typ      types.Type
}

// BuiltinFunctions for all builtin types
var BuiltinFunctionsV2 map[types.Type]map[string]chunkHandlerV2

func init() {
	BuiltinFunctionsV2 = map[types.Type]map[string]chunkHandlerV2{
		types.Nil: {
			// == / !=
			string("==" + types.Nil):          {f: chunkEqTrueV2, Label: "=="},
			string("!=" + types.Nil):          {f: chunkNeqFalseV2, Label: "!="},
			string("==" + types.Bool):         {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Bool):         {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Int):          {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Int):          {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Float):        {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Float):        {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.String):       {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.String):       {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Regex):        {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Regex):        {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Time):         {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Time):         {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Dict):         {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Dict):         {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.ArrayLike):    {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):    {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.MapLike):      {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.MapLike):      {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.ResourceLike): {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ResourceLike): {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.FunctionLike): {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.FunctionLike): {f: chunkNeqTrueV2, Label: "!="},
		},
		types.Bool: {
			// == / !=
			string("==" + types.Nil):                 {f: boolCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: boolNotNilV2, Label: "!="},
			string("==" + types.Bool):                {f: boolCmpBoolV2, Label: "=="},
			string("!=" + types.Bool):                {f: boolNotBoolV2, Label: "!="},
			string("==" + types.Int):                 {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Int):                 {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Float):               {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Float):               {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.String):              {f: boolCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: boolNotStringV2, Label: "!="},
			string("==" + types.Regex):               {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Regex):               {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Time):                {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Time):                {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Dict):                {f: boolCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: boolNotDictV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: boolCmpBoolarrayV2, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: boolNotBoolarrayV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: boolCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: boolNotStringarrayV2, Label: "!="},
			string("==" + types.MapLike):             {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.MapLike):             {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.ResourceLike):        {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ResourceLike):        {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.FunctionLike):        {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.FunctionLike):        {f: chunkNeqTrueV2, Label: "!="},
			//
			string("&&" + types.Bool):      {f: boolAndBoolV2, Label: "&&"},
			string("||" + types.Bool):      {f: boolOrBoolV2, Label: "||"},
			string("&&" + types.Int):       {f: boolAndIntV2, Label: "&&"},
			string("||" + types.Int):       {f: boolOrIntV2, Label: "||"},
			string("&&" + types.Float):     {f: boolAndFloatV2, Label: "&&"},
			string("||" + types.Float):     {f: boolOrFloatV2, Label: "||"},
			string("&&" + types.String):    {f: boolAndStringV2, Label: "&&"},
			string("||" + types.String):    {f: boolOrStringV2, Label: "||"},
			string("&&" + types.Regex):     {f: boolAndRegexV2, Label: "&&"},
			string("||" + types.Regex):     {f: boolOrRegexV2, Label: "||"},
			string("&&" + types.Time):      {f: boolAndTimeV2, Label: "&&"},
			string("||" + types.Time):      {f: boolOrTimeV2, Label: "||"},
			string("&&" + types.Dict):      {f: boolAndDictV2, Label: "&&"},
			string("||" + types.Dict):      {f: boolOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike): {f: boolAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike): {f: boolOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):   {f: boolAndMapV2, Label: "&&"},
			string("||" + types.MapLike):   {f: boolOrMapV2, Label: "||"},
		},
		types.Int: {
			// == / !=
			string("==" + types.Nil):                 {f: intCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: intNotNilV2, Label: "!="},
			string("==" + types.Int):                 {f: intCmpIntV2, Label: "=="},
			string("!=" + types.Int):                 {f: intNotIntV2, Label: "!="},
			string("==" + types.Float):               {f: intCmpFloatV2, Label: "=="},
			string("!=" + types.Float):               {f: intNotFloatV2, Label: "!="},
			string("==" + types.String):              {f: intCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: intNotStringV2, Label: "!="},
			string("==" + types.Regex):               {f: intCmpRegexV2, Label: "=="},
			string("!=" + types.Regex):               {f: intNotRegexV2, Label: "!="},
			string("==" + types.Dict):                {f: intCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: intNotDictV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: intCmpIntarrayV2, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: intNotIntarrayV2, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: intCmpFloatarrayV2, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: intNotFloatarrayV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: intCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: intNotStringarrayV2, Label: "!="},
			string("+" + types.Int):                  {f: intPlusIntV2, Label: "+", Typ: types.Int},
			string("-" + types.Int):                  {f: intMinusIntV2, Label: "-", Typ: types.Int},
			string("*" + types.Int):                  {f: intTimesIntV2, Label: "*", Typ: types.Int},
			string("/" + types.Int):                  {f: intDividedIntV2, Label: "/", Typ: types.Int},
			string("+" + types.Float):                {f: intPlusFloatV2, Label: "+", Typ: types.Float},
			string("-" + types.Float):                {f: intMinusFloatV2, Label: "-", Typ: types.Float},
			string("*" + types.Float):                {f: intTimesFloatV2, Label: "*", Typ: types.Float},
			string("/" + types.Float):                {f: intDividedFloatV2, Label: "/", Typ: types.Float},
			string("+" + types.Dict):                 {f: intPlusDictV2, Label: "+", Typ: types.Float},
			string("-" + types.Dict):                 {f: intMinusDictV2, Label: "-", Typ: types.Float},
			string("*" + types.Dict):                 {f: intTimesDictV2, Label: "*", Typ: types.Float},
			string("/" + types.Dict):                 {f: intDividedDictV2, Label: "/", Typ: types.Float},
			string("*" + types.Time):                 {f: intTimesTimeV2, Label: "*", Typ: types.Time},
			string("<" + types.Int):                  {f: intLTIntV2, Label: "<"},
			string("<=" + types.Int):                 {f: intLTEIntV2, Label: "<="},
			string(">" + types.Int):                  {f: intGTIntV2, Label: ">"},
			string(">=" + types.Int):                 {f: intGTEIntV2, Label: ">="},
			string("<" + types.Float):                {f: intLTFloatV2, Label: "<"},
			string("<=" + types.Float):               {f: intLTEFloatV2, Label: "<="},
			string(">" + types.Float):                {f: intGTFloatV2, Label: ">"},
			string(">=" + types.Float):               {f: intGTEFloatV2, Label: ">="},
			string("<" + types.String):               {f: intLTStringV2, Label: "<"},
			string("<=" + types.String):              {f: intLTEStringV2, Label: "<="},
			string(">" + types.String):               {f: intGTStringV2, Label: ">"},
			string(">=" + types.String):              {f: intGTEStringV2, Label: ">="},
			string("<" + types.Dict):                 {f: intLTDictV2, Label: "<"},
			string("<=" + types.Dict):                {f: intLTEDictV2, Label: "<="},
			string(">" + types.Dict):                 {f: intGTDictV2, Label: ">"},
			string(">=" + types.Dict):                {f: intGTEDictV2, Label: ">="},
			string("&&" + types.Bool):                {f: intAndBoolV2, Label: "&&"},
			string("||" + types.Bool):                {f: intOrBoolV2, Label: "||"},
			string("&&" + types.Int):                 {f: intAndIntV2, Label: "&&"},
			string("||" + types.Int):                 {f: intOrIntV2, Label: "||"},
			string("&&" + types.Float):               {f: intAndFloatV2, Label: "&&"},
			string("||" + types.Float):               {f: intOrFloatV2, Label: "||"},
			string("&&" + types.String):              {f: intAndStringV2, Label: "&&"},
			string("||" + types.String):              {f: intOrStringV2, Label: "||"},
			string("&&" + types.Regex):               {f: intAndRegexV2, Label: "&&"},
			string("||" + types.Regex):               {f: intOrRegexV2, Label: "||"},
			string("&&" + types.Time):                {f: intAndTimeV2, Label: "&&"},
			string("||" + types.Time):                {f: intOrTimeV2, Label: "||"},
			string("&&" + types.Dict):                {f: intAndDictV2, Label: "&&"},
			string("||" + types.Dict):                {f: intOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike):           {f: intAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike):           {f: intOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):             {f: intAndMapV2, Label: "&&"},
			string("||" + types.MapLike):             {f: intOrMapV2, Label: "||"},
		},
		types.Float: {
			// == / !=
			string("==" + types.Nil):                 {f: floatCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: floatNotNilV2, Label: "!="},
			string("==" + types.Float):               {f: floatCmpFloatV2, Label: "=="},
			string("!=" + types.Float):               {f: floatNotFloatV2, Label: "!="},
			string("==" + types.String):              {f: floatCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: floatNotStringV2, Label: "!="},
			string("==" + types.Regex):               {f: floatCmpRegexV2, Label: "=="},
			string("!=" + types.Regex):               {f: floatNotRegexV2, Label: "!="},
			string("==" + types.Dict):                {f: floatCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: floatNotDictV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: floatCmpIntarrayV2, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: floatNotIntarrayV2, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: floatCmpFloatarrayV2, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: floatNotFloatarrayV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: floatCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: floatNotStringarrayV2, Label: "!="},
			string("+" + types.Int):                  {f: floatPlusIntV2, Label: "+", Typ: types.Float},
			string("-" + types.Int):                  {f: floatMinusIntV2, Label: "-", Typ: types.Float},
			string("*" + types.Int):                  {f: floatTimesIntV2, Label: "*", Typ: types.Float},
			string("/" + types.Int):                  {f: floatDividedIntV2, Label: "/", Typ: types.Float},
			string("+" + types.Float):                {f: floatPlusFloatV2, Label: "+", Typ: types.Float},
			string("-" + types.Float):                {f: floatMinusFloatV2, Label: "-", Typ: types.Float},
			string("*" + types.Float):                {f: floatTimesFloatV2, Label: "*", Typ: types.Float},
			string("/" + types.Float):                {f: floatDividedFloatV2, Label: "/", Typ: types.Float},
			string("+" + types.Dict):                 {f: floatPlusDictV2, Label: "+", Typ: types.Float},
			string("-" + types.Dict):                 {f: floatMinusDictV2, Label: "-", Typ: types.Float},
			string("*" + types.Dict):                 {f: floatTimesDictV2, Label: "*", Typ: types.Float},
			string("/" + types.Dict):                 {f: floatDividedDictV2, Label: "/", Typ: types.Float},
			string("*" + types.Time):                 {f: floatTimesTimeV2, Label: "*", Typ: types.Time},
			string("<" + types.Int):                  {f: floatLTIntV2, Label: "<"},
			string("<=" + types.Int):                 {f: floatLTEIntV2, Label: "<="},
			string(">" + types.Int):                  {f: floatGTIntV2, Label: ">"},
			string(">=" + types.Int):                 {f: floatGTEIntV2, Label: ">="},
			string("<" + types.Float):                {f: floatLTFloatV2, Label: "<"},
			string("<=" + types.Float):               {f: floatLTEFloatV2, Label: "<="},
			string(">" + types.Float):                {f: floatGTFloatV2, Label: ">"},
			string(">=" + types.Float):               {f: floatGTEFloatV2, Label: ">="},
			string("<" + types.String):               {f: floatLTStringV2, Label: "<"},
			string("<=" + types.String):              {f: floatLTEStringV2, Label: "<="},
			string(">" + types.String):               {f: floatGTStringV2, Label: ">"},
			string(">=" + types.String):              {f: floatGTEStringV2, Label: ">="},
			string("<" + types.Dict):                 {f: floatLTDictV2, Label: "<"},
			string("<=" + types.Dict):                {f: floatLTEDictV2, Label: "<="},
			string(">" + types.Dict):                 {f: floatGTDictV2, Label: ">"},
			string(">=" + types.Dict):                {f: floatGTEDictV2, Label: ">="},
			string("&&" + types.Bool):                {f: floatAndBoolV2, Label: "&&"},
			string("||" + types.Bool):                {f: floatOrBoolV2, Label: "||"},
			string("&&" + types.Int):                 {f: floatAndIntV2, Label: "&&"},
			string("||" + types.Int):                 {f: floatOrIntV2, Label: "||"},
			string("&&" + types.Float):               {f: floatAndFloatV2, Label: "&&"},
			string("||" + types.Float):               {f: floatOrFloatV2, Label: "||"},
			string("&&" + types.String):              {f: floatAndStringV2, Label: "&&"},
			string("||" + types.String):              {f: floatOrStringV2, Label: "||"},
			string("&&" + types.Regex):               {f: floatAndRegexV2, Label: "&&"},
			string("||" + types.Regex):               {f: floatOrRegexV2, Label: "||"},
			string("&&" + types.Time):                {f: floatAndTimeV2, Label: "&&"},
			string("||" + types.Time):                {f: floatOrTimeV2, Label: "||"},
			string("&&" + types.Dict):                {f: floatAndDictV2, Label: "&&"},
			string("||" + types.Dict):                {f: floatOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike):           {f: floatAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike):           {f: floatOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):             {f: floatAndMapV2, Label: "&&"},
			string("||" + types.MapLike):             {f: floatOrMapV2, Label: "&&"},
		},
		types.String: {
			// == / !=
			string("==" + types.Nil):                 {f: stringCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNilV2, Label: "!="},
			string("==" + types.String):              {f: stringCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: stringNotStringV2, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpRegexV2, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotRegexV2, Label: "!="},
			string("==" + types.Bool):                {f: stringCmpBoolV2, Label: "=="},
			string("!=" + types.Bool):                {f: stringNotBoolV2, Label: "!="},
			string("==" + types.Int):                 {f: stringCmpIntV2, Label: "=="},
			string("!=" + types.Int):                 {f: stringNotIntV2, Label: "!="},
			string("==" + types.Float):               {f: stringCmpFloatV2, Label: "=="},
			string("!=" + types.Float):               {f: stringNotFloatV2, Label: "!="},
			string("==" + types.Dict):                {f: stringCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: stringNotDictV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: stringCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: stringNotStringarrayV2, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: stringCmpBoolarrayV2, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: stringNotBoolarrayV2, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: stringCmpIntarrayV2, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: stringNotIntarrayV2, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: stringCmpFloatarrayV2, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: stringNotFloatarrayV2, Label: "!="},
			string("<" + types.Int):                  {f: stringLTIntV2, Label: "<"},
			string("<=" + types.Int):                 {f: stringLTEIntV2, Label: "<="},
			string(">" + types.Int):                  {f: stringGTIntV2, Label: ">"},
			string(">=" + types.Int):                 {f: stringGTEIntV2, Label: ">="},
			string("<" + types.Float):                {f: stringLTFloatV2, Label: "<"},
			string("<=" + types.Float):               {f: stringLTEFloatV2, Label: "<="},
			string(">" + types.Float):                {f: stringGTFloatV2, Label: ">"},
			string(">=" + types.Float):               {f: stringGTEFloatV2, Label: ">="},
			string("<" + types.String):               {f: stringLTStringV2, Label: "<"},
			string("<=" + types.String):              {f: stringLTEStringV2, Label: "<="},
			string(">" + types.String):               {f: stringGTStringV2, Label: ">"},
			string(">=" + types.String):              {f: stringGTEStringV2, Label: ">="},
			string("<" + types.Dict):                 {f: stringLTDictV2, Label: "<"},
			string("<=" + types.Dict):                {f: stringLTEDictV2, Label: "<="},
			string(">" + types.Dict):                 {f: stringGTDictV2, Label: ">"},
			string(">=" + types.Dict):                {f: stringGTEDictV2, Label: ">="},
			string("&&" + types.Bool):                {f: stringAndBoolV2, Label: "&&"},
			string("||" + types.Bool):                {f: stringOrBoolV2, Label: "||"},
			string("&&" + types.Int):                 {f: stringAndIntV2, Label: "&&"},
			string("||" + types.Int):                 {f: stringOrIntV2, Label: "||"},
			string("&&" + types.Float):               {f: stringAndFloatV2, Label: "&&"},
			string("||" + types.Float):               {f: stringOrFloatV2, Label: "||"},
			string("&&" + types.String):              {f: stringAndStringV2, Label: "&&"},
			string("||" + types.String):              {f: stringOrStringV2, Label: "||"},
			string("&&" + types.Regex):               {f: stringAndRegexV2, Label: "&&"},
			string("||" + types.Regex):               {f: stringOrRegexV2, Label: "||"},
			string("&&" + types.Time):                {f: stringAndTimeV2, Label: "&&"},
			string("||" + types.Time):                {f: stringOrTimeV2, Label: "||"},
			string("&&" + types.Dict):                {f: stringAndDictV2, Label: "&&"},
			string("||" + types.Dict):                {f: stringOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike):           {f: stringAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike):           {f: stringOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):             {f: stringAndMapV2, Label: "&&"},
			string("||" + types.MapLike):             {f: stringOrMapV2, Label: "&&"},
			string("+" + types.String):               {f: stringPlusStringV2, Label: "+"},
			// fields
			string("contains" + types.String):              {f: stringContainsStringV2, Label: "contains"},
			string("contains" + types.Array(types.String)): {f: stringContainsArrayStringV2, Label: "contains"},
			string("contains" + types.Int):                 {f: stringContainsIntV2, Label: "contains"},
			string("contains" + types.Array(types.Int)):    {f: stringContainsArrayIntV2, Label: "contains"},
			string("find"):      {f: stringFindV2, Label: "find"},
			string("camelcase"): {f: stringCamelcaseV2, Label: "camelcase"},
			string("downcase"):  {f: stringDowncaseV2, Label: "downcase"},
			string("upcase"):    {f: stringUpcaseV2, Label: "upcase"},
			string("length"):    {f: stringLengthV2, Label: "length"},
			string("lines"):     {f: stringLinesV2, Label: "lines"},
			string("split"):     {f: stringSplitV2, Label: "split"},
			string("trim"):      {f: stringTrimV2, Label: "trim"},
		},
		types.Regex: {
			// == / !=
			string("==" + types.Nil):                 {f: stringCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNilV2, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpStringV2, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotStringV2, Label: "!="},
			string("==" + types.Bool):                {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Bool):                {f: chunkNeqFalseV2, Label: "!="},
			string("==" + types.Int):                 {f: regexCmpIntV2, Label: "=="},
			string("!=" + types.Int):                 {f: regexNotIntV2, Label: "!="},
			string("==" + types.Float):               {f: regexCmpFloatV2, Label: "=="},
			string("!=" + types.Float):               {f: regexNotFloatV2, Label: "!="},
			string("==" + types.Dict):                {f: regexCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: regexNotDictV2, Label: "!="},
			string("==" + types.String):              {f: regexCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: regexNotStringV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV2, Label: "!="},
			string("==" + types.Array(types.Regex)):  {f: stringCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.Regex)):  {f: stringNotStringarrayV2, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: regexCmpIntarrayV2, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: regexNotIntarrayV2, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: regexCmpFloatarrayV2, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: regexNotFloatarrayV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: regexCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: regexNotStringarrayV2, Label: "!="},
			string("&&" + types.Bool):                {f: regexAndBoolV2, Label: "&&"},
			string("||" + types.Bool):                {f: regexOrBoolV2, Label: "||"},
			string("&&" + types.Int):                 {f: regexAndIntV2, Label: "&&"},
			string("||" + types.Int):                 {f: regexOrIntV2, Label: "||"},
			string("&&" + types.Float):               {f: regexAndFloatV2, Label: "&&"},
			string("||" + types.Float):               {f: regexOrFloatV2, Label: "||"},
			string("&&" + types.String):              {f: regexAndStringV2, Label: "&&"},
			string("||" + types.String):              {f: regexOrStringV2, Label: "||"},
			string("&&" + types.Regex):               {f: regexAndRegexV2, Label: "&&"},
			string("||" + types.Regex):               {f: regexOrRegexV2, Label: "||"},
			string("&&" + types.Time):                {f: regexAndTimeV2, Label: "&&"},
			string("||" + types.Time):                {f: regexOrTimeV2, Label: "||"},
			string("&&" + types.Dict):                {f: regexAndDictV2, Label: "&&"},
			string("||" + types.Dict):                {f: regexOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike):           {f: regexAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike):           {f: regexOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):             {f: regexAndMapV2, Label: "&&"},
			string("||" + types.MapLike):             {f: regexOrMapV2, Label: "&&"},
		},
		types.Time: {
			string("==" + types.Nil):       {f: timeCmpNilV2, Label: "=="},
			string("!=" + types.Nil):       {f: timeNotNilV2, Label: "!="},
			string("==" + types.Time):      {f: timeCmpTimeV2, Label: "=="},
			string("!=" + types.Time):      {f: timeNotTimeV2, Label: "!="},
			string("<" + types.Time):       {f: timeLTTimeV2, Label: "<"},
			string("<=" + types.Time):      {f: timeLTETimeV2, Label: "<="},
			string(">" + types.Time):       {f: timeGTTimeV2, Label: ">"},
			string(">=" + types.Time):      {f: timeGTETimeV2, Label: ">="},
			string("&&" + types.Bool):      {f: timeAndBoolV2, Label: "&&"},
			string("||" + types.Bool):      {f: timeOrBoolV2, Label: "||"},
			string("&&" + types.Int):       {f: timeAndIntV2, Label: "&&"},
			string("||" + types.Int):       {f: timeOrIntV2, Label: "||"},
			string("&&" + types.Float):     {f: timeAndFloatV2, Label: "&&"},
			string("||" + types.Float):     {f: timeOrFloatV2, Label: "||"},
			string("&&" + types.String):    {f: timeAndStringV2, Label: "&&"},
			string("||" + types.String):    {f: timeOrStringV2, Label: "||"},
			string("&&" + types.Regex):     {f: timeAndRegexV2, Label: "&&"},
			string("||" + types.Regex):     {f: timeOrRegexV2, Label: "||"},
			string("&&" + types.Time):      {f: timeAndTimeV2, Label: "&&"},
			string("||" + types.Time):      {f: timeOrTimeV2, Label: "||"},
			string("&&" + types.Dict):      {f: timeAndDictV2, Label: "&&"},
			string("||" + types.Dict):      {f: timeOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike): {f: timeAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike): {f: timeOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):   {f: timeAndMapV2, Label: "&&"},
			string("||" + types.MapLike):   {f: timeOrMapV2, Label: "||"},
			string("-" + types.Time):       {f: timeMinusTimeV2, Label: "-"},
			string("*" + types.Int):        {f: timeTimesIntV2, Label: "*", Typ: types.Time},
			string("*" + types.Float):      {f: timeTimesFloatV2, Label: "*", Typ: types.Time},
			string("*" + types.Dict):       {f: timeTimesDictV2, Label: "*", Typ: types.Time},
			// fields
			string("seconds"): {f: timeSecondsV2, Label: "seconds"},
			string("minutes"): {f: timeMinutesV2, Label: "minutes"},
			string("hours"):   {f: timeHoursV2, Label: "hours"},
			string("days"):    {f: timeDaysV2, Label: "days"},
			string("unix"):    {f: timeUnixV2, Label: "unix"},
		},
		types.Dict: {
			string("==" + types.Nil):                 {f: dictCmpNilV2, Label: "=="},
			string("!=" + types.Nil):                 {f: dictNotNilV2, Label: "!="},
			string("==" + types.Bool):                {f: dictCmpBoolV2, Label: "=="},
			string("!=" + types.Bool):                {f: dictNotBoolV2, Label: "!="},
			string("==" + types.Int):                 {f: dictCmpIntV2, Label: "=="},
			string("!=" + types.Int):                 {f: dictNotIntV2, Label: "!="},
			string("==" + types.Float):               {f: dictCmpFloatV2, Label: "=="},
			string("!=" + types.Float):               {f: dictNotFloatV2, Label: "!="},
			string("==" + types.Dict):                {f: dictCmpDictV2, Label: "=="},
			string("!=" + types.Dict):                {f: dictNotDictV2, Label: "!="},
			string("==" + types.String):              {f: dictCmpStringV2, Label: "=="},
			string("!=" + types.String):              {f: dictNotStringV2, Label: "!="},
			string("==" + types.Regex):               {f: dictCmpRegexV2, Label: "=="},
			string("!=" + types.Regex):               {f: dictNotRegexV2, Label: "!="},
			string("==" + types.ArrayLike):           {f: dictCmpArrayV2, Label: "=="},
			string("!=" + types.ArrayLike):           {f: dictNotArrayV2, Label: "!="},
			string("==" + types.Array(types.String)): {f: dictCmpStringarrayV2, Label: "=="},
			string("!=" + types.Array(types.String)): {f: dictNotStringarrayV2, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: dictCmpBoolarrayV2, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: dictNotBoolarrayV2, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: dictCmpIntarrayV2, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: dictNotIntarrayV2, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: dictCmpFloatarrayV2, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: dictNotFloatarrayV2, Label: "!="},
			string("<" + types.Int):                  {f: dictLTIntV2, Label: "<"},
			string("<=" + types.Int):                 {f: dictLTEIntV2, Label: "<="},
			string(">" + types.Int):                  {f: dictGTIntV2, Label: ">"},
			string(">=" + types.Int):                 {f: dictGTEIntV2, Label: ">="},
			string("<" + types.Float):                {f: dictLTFloatV2, Label: "<"},
			string("<=" + types.Float):               {f: dictLTEFloatV2, Label: "<="},
			string(">" + types.Float):                {f: dictGTFloatV2, Label: ">"},
			string(">=" + types.Float):               {f: dictGTEFloatV2, Label: ">="},
			string("<" + types.String):               {f: dictLTStringV2, Label: "<"},
			string("<=" + types.String):              {f: dictLTEStringV2, Label: "<="},
			string(">" + types.String):               {f: dictGTStringV2, Label: ">"},
			string(">=" + types.String):              {f: dictGTEStringV2, Label: ">="},
			string("<" + types.Dict):                 {f: dictLTDictV2, Label: "<"},
			string("<=" + types.Dict):                {f: dictLTEDictV2, Label: "<="},
			string(">" + types.Dict):                 {f: dictGTDictV2, Label: ">"},
			string(">=" + types.Dict):                {f: dictGTEDictV2, Label: ">="},
			string("&&" + types.Bool):                {f: dictAndBoolV2, Label: "&&"},
			string("||" + types.Bool):                {f: dictOrBoolV2, Label: "||"},
			string("&&" + types.Int):                 {f: dictAndIntV2, Label: "&&"},
			string("||" + types.Int):                 {f: dictOrIntV2, Label: "||"},
			string("&&" + types.Float):               {f: dictAndFloatV2, Label: "&&"},
			string("||" + types.Float):               {f: dictOrFloatV2, Label: "||"},
			string("&&" + types.String):              {f: dictAndStringV2, Label: "&&"},
			string("||" + types.String):              {f: dictOrStringV2, Label: "||"},
			string("&&" + types.Regex):               {f: dictAndRegexV2, Label: "&&"},
			string("||" + types.Regex):               {f: dictOrRegexV2, Label: "||"},
			string("&&" + types.Time):                {f: dictAndTimeV2, Label: "&&"},
			string("||" + types.Time):                {f: dictOrTimeV2, Label: "||"},
			string("&&" + types.Dict):                {f: dictAndDictV2, Label: "&&"},
			string("||" + types.Dict):                {f: dictOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike):           {f: dictAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike):           {f: dictOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):             {f: dictAndMapV2, Label: "&&"},
			string("||" + types.MapLike):             {f: dictOrMapV2, Label: "||"},
			string("+" + types.String):               {f: dictPlusStringV2, Label: "+"},
			string("+" + types.Int):                  {f: dictPlusIntV2, Label: "+"},
			string("-" + types.Int):                  {f: dictMinusIntV2, Label: "-"},
			string("*" + types.Int):                  {f: dictTimesIntV2, Label: "*"},
			string("/" + types.Int):                  {f: dictDividedIntV2, Label: "/"},
			string("+" + types.Float):                {f: dictPlusFloatV2, Label: "+"},
			string("-" + types.Float):                {f: dictMinusFloatV2, Label: "-"},
			string("*" + types.Float):                {f: dictTimesFloatV2, Label: "*"},
			string("/" + types.Float):                {f: dictDividedFloatV2, Label: "/"},
			string("*" + types.Time):                 {f: dictTimesTimeV2, Label: "*"},
			// fields
			"[]":                              {f: dictGetIndexV2},
			"length":                          {f: dictLengthV2},
			"{}":                              {f: dictBlockCallV2},
			"camelcase":                       {f: dictCamelcaseV2, Label: "camelcase"},
			"downcase":                        {f: dictDowncaseV2, Label: "downcase"},
			"upcase":                          {f: dictUpcaseV2, Label: "upcase"},
			"lines":                           {f: dictLinesV2, Label: "lines"},
			"split":                           {f: dictSplitV2, Label: "split"},
			"trim":                            {f: dictTrimV2, Label: "trim"},
			"keys":                            {f: dictKeysV2, Label: "keys"},
			"values":                          {f: dictValuesV2, Label: "values"},
			"where":                           {f: dictWhereV2, Label: "where"},
			"$whereNot":                       {f: dictWhereNotV2},
			"$all":                            {f: dictAllV2},
			"$none":                           {f: dictNoneV2},
			"$any":                            {f: dictAnyV2},
			"$one":                            {f: dictOneV2},
			"map":                             {f: dictMapV2},
			string("contains" + types.String): {f: dictContainsStringV2, Label: "contains"},
			string("contains" + types.Array(types.String)): {f: dictContainsArrayStringV2, Label: "contains"},
			string("contains" + types.Int):                 {f: dictContainsIntV2, Label: "contains"},
			string("contains" + types.Array(types.Int)):    {f: dictContainsArrayIntV2, Label: "contains"},
			string("find"): {f: dictFindV2, Label: "find"},
			// NOTE: the following functions are internal ONLY!
			// We have not yet decided if and how these may be exposed to users
			"notEmpty": {f: dictNotEmptyV2},
		},
		types.ArrayLike: {
			"[]":                     {f: arrayGetIndexV2},
			"first":                  {f: arrayGetFirstIndexV2},
			"last":                   {f: arrayGetLastIndexV2},
			"{}":                     {f: arrayBlockListV2},
			"${}":                    {f: arrayBlockV2},
			"length":                 {f: arrayLengthV2},
			"where":                  {f: arrayWhereV2},
			"$whereNot":              {f: arrayWhereNotV2},
			"$all":                   {f: arrayAllV2},
			"$none":                  {f: arrayNoneV2},
			"$any":                   {f: arrayAnyV2},
			"$one":                   {f: arrayOneV2},
			"map":                    {f: arrayMapV2},
			"duplicates":             {f: arrayDuplicatesV2},
			"fieldDuplicates":        {f: arrayFieldDuplicatesV2},
			"unique":                 {f: arrayUniqueV2},
			"difference":             {f: arrayDifferenceV2},
			"containsNone":           {f: arrayContainsNoneV2},
			"==":                     {Compiler: compileArrayOpArray("=="), f: tarrayCmpTarrayV2, Label: "=="},
			"!=":                     {Compiler: compileArrayOpArray("!="), f: tarrayNotTarrayV2, Label: "!="},
			"==" + string(types.Nil): {f: arrayCmpNilV2},
			"!=" + string(types.Nil): {f: arrayNotNilV2},
			"&&":                     {Compiler: compileLogicalArrayOp(types.ArrayLike, "&&")},
			"||":                     {Compiler: compileLogicalArrayOp(types.ArrayLike, "||")},
			"+":                      {Compiler: compileArrayOpArray("+"), f: tarrayConcatTarrayV2, Label: "+"},
			// logical operations []<T> -- K
			string(types.Any + "&&" + types.Bool):      {f: arrayAndBoolV2, Label: "&&"},
			string(types.Any + "||" + types.Bool):      {f: arrayOrBoolV2, Label: "||"},
			string(types.Any + "&&" + types.Int):       {f: arrayAndIntV2, Label: "&&"},
			string(types.Any + "||" + types.Int):       {f: arrayOrIntV2, Label: "||"},
			string(types.Any + "&&" + types.Float):     {f: arrayAndFloatV2, Label: "&&"},
			string(types.Any + "||" + types.Float):     {f: arrayOrFloatV2, Label: "||"},
			string(types.Any + "&&" + types.String):    {f: arrayAndStringV2, Label: "&&"},
			string(types.Any + "||" + types.String):    {f: arrayOrStringV2, Label: "||"},
			string(types.Any + "&&" + types.Regex):     {f: arrayAndRegexV2, Label: "&&"},
			string(types.Any + "||" + types.Regex):     {f: arrayOrRegexV2, Label: "||"},
			string(types.Any + "&&" + types.ArrayLike): {f: arrayAndArrayV2, Label: "&&"},
			string(types.Any + "||" + types.ArrayLike): {f: arrayOrArrayV2, Label: "||"},
			string(types.Any + "&&" + types.MapLike):   {f: arrayAndMapV2, Label: "&&"},
			string(types.Any + "||" + types.MapLike):   {f: arrayOrMapV2, Label: "||"},
			// []T -- []T
			string(types.Bool + "==" + types.Array(types.Bool)):     {f: boolarrayCmpBoolarrayV2, Label: "=="},
			string(types.Bool + "!=" + types.Array(types.Bool)):     {f: boolarrayNotBoolarrayV2, Label: "!="},
			string(types.Int + "==" + types.Array(types.Int)):       {f: intarrayCmpIntarrayV2, Label: "=="},
			string(types.Int + "!=" + types.Array(types.Int)):       {f: intarrayNotIntarrayV2, Label: "!="},
			string(types.Float + "==" + types.Array(types.Float)):   {f: floatarrayCmpFloatarrayV2, Label: "=="},
			string(types.Float + "!=" + types.Array(types.Float)):   {f: floatarrayNotFloatarrayV2, Label: "!="},
			string(types.String + "==" + types.Array(types.String)): {f: stringarrayCmpStringarrayV2, Label: "=="},
			string(types.String + "!=" + types.Array(types.String)): {f: stringarrayNotStringarrayV2, Label: "!="},
			string(types.Regex + "==" + types.Array(types.Regex)):   {f: stringarrayCmpStringarrayV2, Label: "=="},
			string(types.Regex + "!=" + types.Array(types.Regex)):   {f: stringarrayNotStringarrayV2, Label: "!="},
			// []T -- T
			string(types.Bool + "==" + types.Bool):     {f: boolarrayCmpBoolV2, Label: "=="},
			string(types.Bool + "!=" + types.Bool):     {f: boolarrayNotBoolV2, Label: "!="},
			string(types.Int + "==" + types.Int):       {f: intarrayCmpIntV2, Label: "=="},
			string(types.Int + "!=" + types.Int):       {f: intarrayNotIntV2, Label: "!="},
			string(types.Float + "==" + types.Float):   {f: floatarrayCmpFloatV2, Label: "=="},
			string(types.Float + "!=" + types.Float):   {f: floatarrayNotFloatV2, Label: "!="},
			string(types.String + "==" + types.String): {f: stringarrayCmpStringV2, Label: "=="},
			string(types.String + "!=" + types.String): {f: stringarrayNotStringV2, Label: "!="},
			string(types.Regex + "==" + types.Regex):   {f: stringarrayCmpStringV2, Label: "=="},
			string(types.Regex + "!=" + types.Regex):   {f: stringarrayNotStringV2, Label: "!="},
			// []int/float
			string(types.Int + "==" + types.Float): {f: intarrayCmpFloatV2, Label: "=="},
			string(types.Int + "!=" + types.Float): {f: intarrayNotFloatV2, Label: "!="},
			string(types.Float + "==" + types.Int): {f: floatarrayCmpIntV2, Label: "=="},
			string(types.Float + "!=" + types.Int): {f: floatarrayNotIntV2, Label: "!="},
			// []string -- T
			string(types.String + "==" + types.Bool):  {f: stringarrayCmpBoolV2, Label: "=="},
			string(types.String + "!=" + types.Bool):  {f: stringarrayNotBoolV2, Label: "!="},
			string(types.String + "==" + types.Int):   {f: stringarrayCmpIntV2, Label: "=="},
			string(types.String + "!=" + types.Int):   {f: stringarrayNotIntV2, Label: "!="},
			string(types.String + "==" + types.Float): {f: stringarrayCmpFloatV2, Label: "=="},
			string(types.String + "!=" + types.Float): {f: stringarrayNotFloatV2, Label: "!="},
			// []T -- string
			string(types.Bool + "==" + types.String):  {f: boolarrayCmpStringV2, Label: "=="},
			string(types.Bool + "!=" + types.String):  {f: boolarrayNotStringV2, Label: "!="},
			string(types.Int + "==" + types.String):   {f: intarrayCmpStringV2, Label: "=="},
			string(types.Int + "!=" + types.String):   {f: intarrayNotStringV2, Label: "!="},
			string(types.Float + "==" + types.String): {f: floatarrayCmpStringV2, Label: "=="},
			string(types.Float + "!=" + types.String): {f: floatarrayNotStringV2, Label: "!="},
			// []T -- regex
			string(types.Int + "==" + types.Regex):    {f: intarrayCmpRegexV2, Label: "=="},
			string(types.Int + "!=" + types.Regex):    {f: intarrayNotRegexV2, Label: "!="},
			string(types.Float + "==" + types.Regex):  {f: floatarrayCmpRegexV2, Label: "=="},
			string(types.Float + "!=" + types.Regex):  {f: floatarrayNotRegexV2, Label: "!="},
			string(types.String + "==" + types.Regex): {f: stringarrayCmpRegexV2, Label: "=="},
			string(types.String + "!=" + types.Regex): {f: stringarrayNotRegexV2, Label: "!="},
			// NOTE: the following functions are internal ONLY!
			// We have not yet decided if and how these may be exposed to users
			"notEmpty": {f: arrayNotEmptyV2},
		},
		types.MapLike: {
			"[]":        {f: mapGetIndexV2},
			"length":    {f: mapLengthV2},
			"where":     {f: mapWhereV2},
			"$whereNot": {f: mapWhereNotV2},
			"{}":        {f: mapBlockCallV2},
			"keys":      {f: mapKeysV2, Label: "keys"},
			"values":    {f: mapValuesV2, Label: "values"},
			// {}T -- T
			string("&&" + types.Bool):      {f: chunkEqFalseV2, Label: "&&"},
			string("||" + types.Bool):      {f: chunkNeqTrueV2, Label: "||"},
			string("&&" + types.Int):       {f: mapAndIntV2, Label: "&&"},
			string("||" + types.Int):       {f: mapOrIntV2, Label: "||"},
			string("&&" + types.Float):     {f: mapAndFloatV2, Label: "&&"},
			string("||" + types.Float):     {f: mapOrFloatV2, Label: "||"},
			string("&&" + types.String):    {f: mapAndStringV2, Label: "&&"},
			string("||" + types.String):    {f: mapOrStringV2, Label: "||"},
			string("&&" + types.Regex):     {f: mapAndRegexV2, Label: "&&"},
			string("||" + types.Regex):     {f: mapOrRegexV2, Label: "||"},
			string("&&" + types.Time):      {f: mapAndTimeV2, Label: "&&"},
			string("||" + types.Time):      {f: mapOrTimeV2, Label: "||"},
			string("&&" + types.Dict):      {f: mapAndDictV2, Label: "&&"},
			string("||" + types.Dict):      {f: mapOrDictV2, Label: "||"},
			string("&&" + types.ArrayLike): {f: mapAndArrayV2, Label: "&&"},
			string("||" + types.ArrayLike): {f: mapOrArrayV2, Label: "||"},
			string("&&" + types.MapLike):   {f: mapAndMapV2, Label: "&&"},
			string("||" + types.MapLike):   {f: mapOrMapV2, Label: "||"},
		},
		types.ResourceLike: {
			// == / !=
			string("==" + types.Nil): {f: chunkEqFalseV2, Label: "=="},
			string("!=" + types.Nil): {f: chunkNeqTrueV2, Label: "!="},
			// fields
			"where":     {f: resourceWhereV2},
			"$whereNot": {f: resourceWhereNotV2},
			"map":       {f: resourceMapV2},
			"length":    {f: resourceLengthV2},
			"{}": {f: func(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
				return e.runBlock(bind, chunk.Function.Args[0], chunk.Function.Args[1:], ref)
			}},
			// TODO: [#32] unique builtin fields that need a long-term support in LR
			string(types.Resource("parse") + ".date"): {f: resourceDateV2},
		},
	}

	validateBuiltinFunctionsV2()
}

func validateBuiltinFunctionsV2() {
	missing := []string{}

	// dict must have all string methods supported
	dictFun := BuiltinFunctionsV2[types.Dict]
	if dictFun == nil {
		dictFun = map[string]chunkHandlerV2{}
	}

	stringFun := BuiltinFunctionsV2[types.String]
	if stringFun == nil {
		stringFun = map[string]chunkHandlerV2{}
	}

	for id := range stringFun {
		if _, ok := dictFun[id]; !ok {
			missing = append(missing, fmt.Sprintf("dict> missing dict counterpart of string function %#v", id))
		}
	}

	// finalize
	if len(missing) == 0 {
		return
	}
	fmt.Println("Missing functions:")
	for _, msg := range missing {
		fmt.Println(msg)
	}
	panic("missing functions must be added")
}

func runResourceFunction(e *blockExecutor, bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	// ugh something is wrong here.... fix it later
	rr, ok := bind.Value.(resources.ResourceType)
	if !ok {
		// TODO: can we get rid of this fmt call
		return nil, 0, fmt.Errorf("cannot cast resource to resource type: %+v", bind.Value)
	}

	info := rr.MqlResource()
	// resource := c.runtime.Registry.Resources[bind.Type]
	if info == nil {
		return nil, 0, errors.New("cannot retrieve resource from the binding to run the raw function")
	}

	resource, ok := e.ctx.runtime.Registry.Resources[info.Name]
	if !ok || resource == nil {
		return nil, 0, fmt.Errorf("cannot retrieve resource definition for resource %q", info.Name)
	}

	// record this watcher on the executors watcher IDs
	wid := e.watcherUID(ref)
	// log.Debug().Str("wid", wid).Msg("exec> add watcher id ")
	e.watcherIds.Store(wid)

	// watch this field in the resource
	err := e.ctx.runtime.WatchAndUpdate(rr, chunk.Id, wid, func(fieldData interface{}, fieldError error) {
		data := &RawData{
			Type:  types.Type(resource.Fields[chunk.Id].Type),
			Value: fieldData,
			Error: fieldError,
		}
		e.cache.Store(ref, &stepCache{
			Result: data,
		})

		codeID, ok := e.callbackPoints[ref]
		if ok {
			e.callback(&RawResult{Data: data, CodeID: codeID})
		}

		if fieldError != nil {
			e.triggerChainError(ref, fieldError)
		}

		e.triggerChain(ref, data)
	})
	if err != nil {
		if _, ok := err.(resources.NotReadyError); !ok {
			// TODO: Deduplicate storage between cache and resource storage
			// This will take some work, but clearly we don't need both

			info.Cache.Store(chunk.Id, &resources.CacheEntry{
				Timestamp: time.Now().Unix(),
				Valid:     true,
				Error:     err,
			})

			fieldType := types.Unset
			if field := resource.Fields[chunk.Id]; field != nil {
				fieldType = types.Type(field.Type)
			}

			e.cache.Store(ref, &stepCache{
				Result: &RawData{
					Type:  fieldType,
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
func BuiltinFunctionV2(typ types.Type, name string) (*chunkHandlerV2, error) {
	h, ok := BuiltinFunctionsV2[typ.Underlying()]
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
func (e *blockExecutor) runBoundFunction(bind *RawData, chunk *Chunk, ref uint64) (*RawData, uint64, error) {
	log.Trace().Uint64("ref", ref).Str("id", chunk.Id).Msg("exec> run bound function")

	fh, err := BuiltinFunctionV2(bind.Type, chunk.Id)
	if err == nil {
		res, dref, err := fh.f(e, bind, chunk, ref)
		if res != nil {
			e.cache.Store(ref, &stepCache{Result: res})
		}
		if err != nil {
			e.cache.Store(ref, &stepCache{Result: &RawData{
				Error: err,
			}})
		}
		return res, dref, err
	}

	if bind.Type.IsResource() {
		return runResourceFunction(e, bind, chunk, ref)
	}
	return nil, 0, err
}
