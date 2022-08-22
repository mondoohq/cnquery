package llx

import (
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/resources"
	"go.mondoo.io/mondoo/types"
)

type chunkHandlerV1 struct {
	Compiler func(types.Type, types.Type) (string, error)
	f        func(*MQLExecutorV1, *RawData, *Chunk, int32) (*RawData, int32, error)
	Label    string
	Typ      types.Type
}

// BuiltinFunctions for all builtin types
var BuiltinFunctionsV1 map[types.Type]map[string]chunkHandlerV1

func init() {
	BuiltinFunctionsV1 = map[types.Type]map[string]chunkHandlerV1{
		types.Nil: {
			// == / !=
			string("==" + types.Nil):          {f: chunkEqTrueV1, Label: "=="},
			string("!=" + types.Nil):          {f: chunkNeqFalseV1, Label: "!="},
			string("==" + types.Bool):         {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Bool):         {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Int):          {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Int):          {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Float):        {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Float):        {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.String):       {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.String):       {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Regex):        {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Regex):        {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Time):         {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Time):         {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Dict):         {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Dict):         {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.ArrayLike):    {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):    {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.MapLike):      {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.MapLike):      {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.ResourceLike): {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ResourceLike): {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.FunctionLike): {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.FunctionLike): {f: chunkNeqTrueV1, Label: "!="},
		},
		types.Bool: {
			// == / !=
			string("==" + types.Nil):                 {f: boolCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: boolNotNilV1, Label: "!="},
			string("==" + types.Bool):                {f: boolCmpBoolV1, Label: "=="},
			string("!=" + types.Bool):                {f: boolNotBoolV1, Label: "!="},
			string("==" + types.Int):                 {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Int):                 {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Float):               {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Float):               {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.String):              {f: boolCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: boolNotStringV1, Label: "!="},
			string("==" + types.Regex):               {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Regex):               {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Time):                {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Time):                {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Dict):                {f: boolCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: boolNotDictV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: boolCmpBoolarrayV1, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: boolNotBoolarrayV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: boolCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: boolNotStringarrayV1, Label: "!="},
			string("==" + types.MapLike):             {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.MapLike):             {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.ResourceLike):        {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ResourceLike):        {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.FunctionLike):        {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.FunctionLike):        {f: chunkNeqTrueV1, Label: "!="},
			//
			string("&&" + types.Bool):      {f: boolAndBoolV1, Label: "&&"},
			string("||" + types.Bool):      {f: boolOrBoolV1, Label: "||"},
			string("&&" + types.Int):       {f: boolAndIntV1, Label: "&&"},
			string("||" + types.Int):       {f: boolOrIntV1, Label: "||"},
			string("&&" + types.Float):     {f: boolAndFloatV1, Label: "&&"},
			string("||" + types.Float):     {f: boolOrFloatV1, Label: "||"},
			string("&&" + types.String):    {f: boolAndStringV1, Label: "&&"},
			string("||" + types.String):    {f: boolOrStringV1, Label: "||"},
			string("&&" + types.Regex):     {f: boolAndRegexV1, Label: "&&"},
			string("||" + types.Regex):     {f: boolOrRegexV1, Label: "||"},
			string("&&" + types.Time):      {f: boolAndTimeV1, Label: "&&"},
			string("||" + types.Time):      {f: boolOrTimeV1, Label: "||"},
			string("&&" + types.Dict):      {f: boolAndDictV1, Label: "&&"},
			string("||" + types.Dict):      {f: boolOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike): {f: boolAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike): {f: boolOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):   {f: boolAndMapV1, Label: "&&"},
			string("||" + types.MapLike):   {f: boolOrMapV1, Label: "||"},
		},
		types.Int: {
			// == / !=
			string("==" + types.Nil):                 {f: intCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: intNotNilV1, Label: "!="},
			string("==" + types.Int):                 {f: intCmpIntV1, Label: "=="},
			string("!=" + types.Int):                 {f: intNotIntV1, Label: "!="},
			string("==" + types.Float):               {f: intCmpFloatV1, Label: "=="},
			string("!=" + types.Float):               {f: intNotFloatV1, Label: "!="},
			string("==" + types.String):              {f: intCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: intNotStringV1, Label: "!="},
			string("==" + types.Regex):               {f: intCmpRegexV1, Label: "=="},
			string("!=" + types.Regex):               {f: intNotRegexV1, Label: "!="},
			string("==" + types.Dict):                {f: intCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: intNotDictV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: intCmpIntarrayV1, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: intNotIntarrayV1, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: intCmpFloatarrayV1, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: intNotFloatarrayV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: intCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: intNotStringarrayV1, Label: "!="},
			string("+" + types.Int):                  {f: intPlusIntV1, Label: "+", Typ: types.Int},
			string("-" + types.Int):                  {f: intMinusIntV1, Label: "-", Typ: types.Int},
			string("*" + types.Int):                  {f: intTimesIntV1, Label: "*", Typ: types.Int},
			string("/" + types.Int):                  {f: intDividedIntV1, Label: "/", Typ: types.Int},
			string("+" + types.Float):                {f: intPlusFloatV1, Label: "+", Typ: types.Float},
			string("-" + types.Float):                {f: intMinusFloatV1, Label: "-", Typ: types.Float},
			string("*" + types.Float):                {f: intTimesFloatV1, Label: "*", Typ: types.Float},
			string("/" + types.Float):                {f: intDividedFloatV1, Label: "/", Typ: types.Float},
			string("+" + types.Dict):                 {f: intPlusDictV1, Label: "+", Typ: types.Float},
			string("-" + types.Dict):                 {f: intMinusDictV1, Label: "-", Typ: types.Float},
			string("*" + types.Dict):                 {f: intTimesDictV1, Label: "*", Typ: types.Float},
			string("/" + types.Dict):                 {f: intDividedDictV1, Label: "/", Typ: types.Float},
			string("*" + types.Time):                 {f: intTimesTimeV1, Label: "*", Typ: types.Time},
			string("<" + types.Int):                  {f: intLTIntV1, Label: "<"},
			string("<=" + types.Int):                 {f: intLTEIntV1, Label: "<="},
			string(">" + types.Int):                  {f: intGTIntV1, Label: ">"},
			string(">=" + types.Int):                 {f: intGTEIntV1, Label: ">="},
			string("<" + types.Float):                {f: intLTFloatV1, Label: "<"},
			string("<=" + types.Float):               {f: intLTEFloatV1, Label: "<="},
			string(">" + types.Float):                {f: intGTFloatV1, Label: ">"},
			string(">=" + types.Float):               {f: intGTEFloatV1, Label: ">="},
			string("<" + types.String):               {f: intLTStringV1, Label: "<"},
			string("<=" + types.String):              {f: intLTEStringV1, Label: "<="},
			string(">" + types.String):               {f: intGTStringV1, Label: ">"},
			string(">=" + types.String):              {f: intGTEStringV1, Label: ">="},
			string("<" + types.Dict):                 {f: intLTDictV1, Label: "<"},
			string("<=" + types.Dict):                {f: intLTEDictV1, Label: "<="},
			string(">" + types.Dict):                 {f: intGTDictV1, Label: ">"},
			string(">=" + types.Dict):                {f: intGTEDictV1, Label: ">="},
			string("&&" + types.Bool):                {f: intAndBoolV1, Label: "&&"},
			string("||" + types.Bool):                {f: intOrBoolV1, Label: "||"},
			string("&&" + types.Int):                 {f: intAndIntV1, Label: "&&"},
			string("||" + types.Int):                 {f: intOrIntV1, Label: "||"},
			string("&&" + types.Float):               {f: intAndFloatV1, Label: "&&"},
			string("||" + types.Float):               {f: intOrFloatV1, Label: "||"},
			string("&&" + types.String):              {f: intAndStringV1, Label: "&&"},
			string("||" + types.String):              {f: intOrStringV1, Label: "||"},
			string("&&" + types.Regex):               {f: intAndRegexV1, Label: "&&"},
			string("||" + types.Regex):               {f: intOrRegexV1, Label: "||"},
			string("&&" + types.Time):                {f: intAndTimeV1, Label: "&&"},
			string("||" + types.Time):                {f: intOrTimeV1, Label: "||"},
			string("&&" + types.Dict):                {f: intAndDictV1, Label: "&&"},
			string("||" + types.Dict):                {f: intOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike):           {f: intAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike):           {f: intOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):             {f: intAndMapV1, Label: "&&"},
			string("||" + types.MapLike):             {f: intOrMapV1, Label: "||"},
		},
		types.Float: {
			// == / !=
			string("==" + types.Nil):                 {f: floatCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: floatNotNilV1, Label: "!="},
			string("==" + types.Float):               {f: floatCmpFloatV1, Label: "=="},
			string("!=" + types.Float):               {f: floatNotFloatV1, Label: "!="},
			string("==" + types.String):              {f: floatCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: floatNotStringV1, Label: "!="},
			string("==" + types.Regex):               {f: floatCmpRegexV1, Label: "=="},
			string("!=" + types.Regex):               {f: floatNotRegexV1, Label: "!="},
			string("==" + types.Dict):                {f: floatCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: floatNotDictV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: floatCmpIntarrayV1, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: floatNotIntarrayV1, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: floatCmpFloatarrayV1, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: floatNotFloatarrayV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: floatCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: floatNotStringarrayV1, Label: "!="},
			string("+" + types.Int):                  {f: floatPlusIntV1, Label: "+", Typ: types.Float},
			string("-" + types.Int):                  {f: floatMinusIntV1, Label: "-", Typ: types.Float},
			string("*" + types.Int):                  {f: floatTimesIntV1, Label: "*", Typ: types.Float},
			string("/" + types.Int):                  {f: floatDividedIntV1, Label: "/", Typ: types.Float},
			string("+" + types.Float):                {f: floatPlusFloatV1, Label: "+", Typ: types.Float},
			string("-" + types.Float):                {f: floatMinusFloatV1, Label: "-", Typ: types.Float},
			string("*" + types.Float):                {f: floatTimesFloatV1, Label: "*", Typ: types.Float},
			string("/" + types.Float):                {f: floatDividedFloatV1, Label: "/", Typ: types.Float},
			string("+" + types.Dict):                 {f: floatPlusDictV1, Label: "+", Typ: types.Float},
			string("-" + types.Dict):                 {f: floatMinusDictV1, Label: "-", Typ: types.Float},
			string("*" + types.Dict):                 {f: floatTimesDictV1, Label: "*", Typ: types.Float},
			string("/" + types.Dict):                 {f: floatDividedDictV1, Label: "/", Typ: types.Float},
			string("*" + types.Time):                 {f: floatTimesTimeV1, Label: "*", Typ: types.Time},
			string("<" + types.Int):                  {f: floatLTIntV1, Label: "<"},
			string("<=" + types.Int):                 {f: floatLTEIntV1, Label: "<="},
			string(">" + types.Int):                  {f: floatGTIntV1, Label: ">"},
			string(">=" + types.Int):                 {f: floatGTEIntV1, Label: ">="},
			string("<" + types.Float):                {f: floatLTFloatV1, Label: "<"},
			string("<=" + types.Float):               {f: floatLTEFloatV1, Label: "<="},
			string(">" + types.Float):                {f: floatGTFloatV1, Label: ">"},
			string(">=" + types.Float):               {f: floatGTEFloatV1, Label: ">="},
			string("<" + types.String):               {f: floatLTStringV1, Label: "<"},
			string("<=" + types.String):              {f: floatLTEStringV1, Label: "<="},
			string(">" + types.String):               {f: floatGTStringV1, Label: ">"},
			string(">=" + types.String):              {f: floatGTEStringV1, Label: ">="},
			string("<" + types.Dict):                 {f: floatLTDictV1, Label: "<"},
			string("<=" + types.Dict):                {f: floatLTEDictV1, Label: "<="},
			string(">" + types.Dict):                 {f: floatGTDictV1, Label: ">"},
			string(">=" + types.Dict):                {f: floatGTEDictV1, Label: ">="},
			string("&&" + types.Bool):                {f: floatAndBoolV1, Label: "&&"},
			string("||" + types.Bool):                {f: floatOrBoolV1, Label: "||"},
			string("&&" + types.Int):                 {f: floatAndIntV1, Label: "&&"},
			string("||" + types.Int):                 {f: floatOrIntV1, Label: "||"},
			string("&&" + types.Float):               {f: floatAndFloatV1, Label: "&&"},
			string("||" + types.Float):               {f: floatOrFloatV1, Label: "||"},
			string("&&" + types.String):              {f: floatAndStringV1, Label: "&&"},
			string("||" + types.String):              {f: floatOrStringV1, Label: "||"},
			string("&&" + types.Regex):               {f: floatAndRegexV1, Label: "&&"},
			string("||" + types.Regex):               {f: floatOrRegexV1, Label: "||"},
			string("&&" + types.Time):                {f: floatAndTimeV1, Label: "&&"},
			string("||" + types.Time):                {f: floatOrTimeV1, Label: "||"},
			string("&&" + types.Dict):                {f: floatAndDictV1, Label: "&&"},
			string("||" + types.Dict):                {f: floatOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike):           {f: floatAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike):           {f: floatOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):             {f: floatAndMapV1, Label: "&&"},
			string("||" + types.MapLike):             {f: floatOrMapV1, Label: "&&"},
		},
		types.String: {
			// == / !=
			string("==" + types.Nil):                 {f: stringCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNilV1, Label: "!="},
			string("==" + types.String):              {f: stringCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: stringNotStringV1, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpRegexV1, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotRegexV1, Label: "!="},
			string("==" + types.Bool):                {f: stringCmpBoolV1, Label: "=="},
			string("!=" + types.Bool):                {f: stringNotBoolV1, Label: "!="},
			string("==" + types.Int):                 {f: stringCmpIntV1, Label: "=="},
			string("!=" + types.Int):                 {f: stringNotIntV1, Label: "!="},
			string("==" + types.Float):               {f: stringCmpFloatV1, Label: "=="},
			string("!=" + types.Float):               {f: stringNotFloatV1, Label: "!="},
			string("==" + types.Dict):                {f: stringCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: stringNotDictV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: stringCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: stringNotStringarrayV1, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: stringCmpBoolarrayV1, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: stringNotBoolarrayV1, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: stringCmpIntarrayV1, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: stringNotIntarrayV1, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: stringCmpFloatarrayV1, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: stringNotFloatarrayV1, Label: "!="},
			string("<" + types.Int):                  {f: stringLTIntV1, Label: "<"},
			string("<=" + types.Int):                 {f: stringLTEIntV1, Label: "<="},
			string(">" + types.Int):                  {f: stringGTIntV1, Label: ">"},
			string(">=" + types.Int):                 {f: stringGTEIntV1, Label: ">="},
			string("<" + types.Float):                {f: stringLTFloatV1, Label: "<"},
			string("<=" + types.Float):               {f: stringLTEFloatV1, Label: "<="},
			string(">" + types.Float):                {f: stringGTFloatV1, Label: ">"},
			string(">=" + types.Float):               {f: stringGTEFloatV1, Label: ">="},
			string("<" + types.String):               {f: stringLTStringV1, Label: "<"},
			string("<=" + types.String):              {f: stringLTEStringV1, Label: "<="},
			string(">" + types.String):               {f: stringGTStringV1, Label: ">"},
			string(">=" + types.String):              {f: stringGTEStringV1, Label: ">="},
			string("<" + types.Dict):                 {f: stringLTDictV1, Label: "<"},
			string("<=" + types.Dict):                {f: stringLTEDictV1, Label: "<="},
			string(">" + types.Dict):                 {f: stringGTDictV1, Label: ">"},
			string(">=" + types.Dict):                {f: stringGTEDictV1, Label: ">="},
			string("&&" + types.Bool):                {f: stringAndBoolV1, Label: "&&"},
			string("||" + types.Bool):                {f: stringOrBoolV1, Label: "||"},
			string("&&" + types.Int):                 {f: stringAndIntV1, Label: "&&"},
			string("||" + types.Int):                 {f: stringOrIntV1, Label: "||"},
			string("&&" + types.Float):               {f: stringAndFloatV1, Label: "&&"},
			string("||" + types.Float):               {f: stringOrFloatV1, Label: "||"},
			string("&&" + types.String):              {f: stringAndStringV1, Label: "&&"},
			string("||" + types.String):              {f: stringOrStringV1, Label: "||"},
			string("&&" + types.Regex):               {f: stringAndRegexV1, Label: "&&"},
			string("||" + types.Regex):               {f: stringOrRegexV1, Label: "||"},
			string("&&" + types.Time):                {f: stringAndTimeV1, Label: "&&"},
			string("||" + types.Time):                {f: stringOrTimeV1, Label: "||"},
			string("&&" + types.Dict):                {f: stringAndDictV1, Label: "&&"},
			string("||" + types.Dict):                {f: stringOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike):           {f: stringAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike):           {f: stringOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):             {f: stringAndMapV1, Label: "&&"},
			string("||" + types.MapLike):             {f: stringOrMapV1, Label: "&&"},
			string("+" + types.String):               {f: stringPlusStringV1, Label: "+"},
			// fields
			string("contains" + types.String):              {f: stringContainsStringV1, Label: "contains"},
			string("contains" + types.Array(types.String)): {f: stringContainsArrayStringV1, Label: "contains"},
			string("contains" + types.Int):                 {f: stringContainsIntV1, Label: "contains"},
			string("contains" + types.Array(types.Int)):    {f: stringContainsArrayIntV1, Label: "contains"},
			string("find"):      {f: stringFindV1, Label: "find"},
			string("camelcase"): {f: stringCamelcaseV1, Label: "camelcase"},
			string("downcase"):  {f: stringDowncaseV1, Label: "downcase"},
			string("upcase"):    {f: stringUpcaseV1, Label: "upcase"},
			string("length"):    {f: stringLengthV1, Label: "length"},
			string("lines"):     {f: stringLinesV1, Label: "lines"},
			string("split"):     {f: stringSplitV1, Label: "split"},
			string("trim"):      {f: stringTrimV1, Label: "trim"},
		},
		types.Regex: {
			// == / !=
			string("==" + types.Nil):                 {f: stringCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: stringNotNilV1, Label: "!="},
			string("==" + types.Regex):               {f: stringCmpStringV1, Label: "=="},
			string("!=" + types.Regex):               {f: stringNotStringV1, Label: "!="},
			string("==" + types.Bool):                {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Bool):                {f: chunkNeqFalseV1, Label: "!="},
			string("==" + types.Int):                 {f: regexCmpIntV1, Label: "=="},
			string("!=" + types.Int):                 {f: regexNotIntV1, Label: "!="},
			string("==" + types.Float):               {f: regexCmpFloatV1, Label: "=="},
			string("!=" + types.Float):               {f: regexNotFloatV1, Label: "!="},
			string("==" + types.Dict):                {f: regexCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: regexNotDictV1, Label: "!="},
			string("==" + types.String):              {f: regexCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: regexNotStringV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: chunkNeqTrueV1, Label: "!="},
			string("==" + types.Array(types.Regex)):  {f: stringCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.Regex)):  {f: stringNotStringarrayV1, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: regexCmpIntarrayV1, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: regexNotIntarrayV1, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: regexCmpFloatarrayV1, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: regexNotFloatarrayV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: regexCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: regexNotStringarrayV1, Label: "!="},
			string("&&" + types.Bool):                {f: regexAndBoolV1, Label: "&&"},
			string("||" + types.Bool):                {f: regexOrBoolV1, Label: "||"},
			string("&&" + types.Int):                 {f: regexAndIntV1, Label: "&&"},
			string("||" + types.Int):                 {f: regexOrIntV1, Label: "||"},
			string("&&" + types.Float):               {f: regexAndFloatV1, Label: "&&"},
			string("||" + types.Float):               {f: regexOrFloatV1, Label: "||"},
			string("&&" + types.String):              {f: regexAndStringV1, Label: "&&"},
			string("||" + types.String):              {f: regexOrStringV1, Label: "||"},
			string("&&" + types.Regex):               {f: regexAndRegexV1, Label: "&&"},
			string("||" + types.Regex):               {f: regexOrRegexV1, Label: "||"},
			string("&&" + types.Time):                {f: regexAndTimeV1, Label: "&&"},
			string("||" + types.Time):                {f: regexOrTimeV1, Label: "||"},
			string("&&" + types.Dict):                {f: regexAndDictV1, Label: "&&"},
			string("||" + types.Dict):                {f: regexOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike):           {f: regexAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike):           {f: regexOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):             {f: regexAndMapV1, Label: "&&"},
			string("||" + types.MapLike):             {f: regexOrMapV1, Label: "&&"},
		},
		types.Time: {
			string("==" + types.Nil):       {f: timeCmpNilV1, Label: "=="},
			string("!=" + types.Nil):       {f: timeNotNilV1, Label: "!="},
			string("==" + types.Time):      {f: timeCmpTimeV1, Label: "=="},
			string("!=" + types.Time):      {f: timeNotTimeV1, Label: "!="},
			string("<" + types.Time):       {f: timeLTTimeV1, Label: "<"},
			string("<=" + types.Time):      {f: timeLTETimeV1, Label: "<="},
			string(">" + types.Time):       {f: timeGTTimeV1, Label: ">"},
			string(">=" + types.Time):      {f: timeGTETimeV1, Label: ">="},
			string("&&" + types.Bool):      {f: timeAndBoolV1, Label: "&&"},
			string("||" + types.Bool):      {f: timeOrBoolV1, Label: "||"},
			string("&&" + types.Int):       {f: timeAndIntV1, Label: "&&"},
			string("||" + types.Int):       {f: timeOrIntV1, Label: "||"},
			string("&&" + types.Float):     {f: timeAndFloatV1, Label: "&&"},
			string("||" + types.Float):     {f: timeOrFloatV1, Label: "||"},
			string("&&" + types.String):    {f: timeAndStringV1, Label: "&&"},
			string("||" + types.String):    {f: timeOrStringV1, Label: "||"},
			string("&&" + types.Regex):     {f: timeAndRegexV1, Label: "&&"},
			string("||" + types.Regex):     {f: timeOrRegexV1, Label: "||"},
			string("&&" + types.Time):      {f: timeAndTimeV1, Label: "&&"},
			string("||" + types.Time):      {f: timeOrTimeV1, Label: "||"},
			string("&&" + types.Dict):      {f: timeAndDictV1, Label: "&&"},
			string("||" + types.Dict):      {f: timeOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike): {f: timeAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike): {f: timeOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):   {f: timeAndMapV1, Label: "&&"},
			string("||" + types.MapLike):   {f: timeOrMapV1, Label: "||"},
			string("-" + types.Time):       {f: timeMinusTimeV1, Label: "-"},
			string("*" + types.Int):        {f: timeTimesIntV1, Label: "*", Typ: types.Time},
			string("*" + types.Float):      {f: timeTimesFloatV1, Label: "*", Typ: types.Time},
			string("*" + types.Dict):       {f: timeTimesDictV1, Label: "*", Typ: types.Time},
			// fields
			string("seconds"): {f: timeSecondsV1, Label: "seconds"},
			string("minutes"): {f: timeMinutesV1, Label: "minutes"},
			string("hours"):   {f: timeHoursV1, Label: "hours"},
			string("days"):    {f: timeDaysV1, Label: "days"},
			string("unix"):    {f: timeUnixV1, Label: "unix"},
		},
		types.Dict: {
			string("==" + types.Nil):                 {f: dictCmpNilV1, Label: "=="},
			string("!=" + types.Nil):                 {f: dictNotNilV1, Label: "!="},
			string("==" + types.Bool):                {f: dictCmpBoolV1, Label: "=="},
			string("!=" + types.Bool):                {f: dictNotBoolV1, Label: "!="},
			string("==" + types.Int):                 {f: dictCmpIntV1, Label: "=="},
			string("!=" + types.Int):                 {f: dictNotIntV1, Label: "!="},
			string("==" + types.Float):               {f: dictCmpFloatV1, Label: "=="},
			string("!=" + types.Float):               {f: dictNotFloatV1, Label: "!="},
			string("==" + types.Dict):                {f: dictCmpDictV1, Label: "=="},
			string("!=" + types.Dict):                {f: dictNotDictV1, Label: "!="},
			string("==" + types.String):              {f: dictCmpStringV1, Label: "=="},
			string("!=" + types.String):              {f: dictNotStringV1, Label: "!="},
			string("==" + types.Regex):               {f: dictCmpRegexV1, Label: "=="},
			string("!=" + types.Regex):               {f: dictNotRegexV1, Label: "!="},
			string("==" + types.ArrayLike):           {f: dictCmpArrayV1, Label: "=="},
			string("!=" + types.ArrayLike):           {f: dictNotArrayV1, Label: "!="},
			string("==" + types.Array(types.String)): {f: dictCmpStringarrayV1, Label: "=="},
			string("!=" + types.Array(types.String)): {f: dictNotStringarrayV1, Label: "!="},
			string("==" + types.Array(types.Bool)):   {f: dictCmpBoolarrayV1, Label: "=="},
			string("!=" + types.Array(types.Bool)):   {f: dictNotBoolarrayV1, Label: "!="},
			string("==" + types.Array(types.Int)):    {f: dictCmpIntarrayV1, Label: "=="},
			string("!=" + types.Array(types.Int)):    {f: dictNotIntarrayV1, Label: "!="},
			string("==" + types.Array(types.Float)):  {f: dictCmpFloatarrayV1, Label: "=="},
			string("!=" + types.Array(types.Float)):  {f: dictNotFloatarrayV1, Label: "!="},
			string("<" + types.Int):                  {f: dictLTIntV1, Label: "<"},
			string("<=" + types.Int):                 {f: dictLTEIntV1, Label: "<="},
			string(">" + types.Int):                  {f: dictGTIntV1, Label: ">"},
			string(">=" + types.Int):                 {f: dictGTEIntV1, Label: ">="},
			string("<" + types.Float):                {f: dictLTFloatV1, Label: "<"},
			string("<=" + types.Float):               {f: dictLTEFloatV1, Label: "<="},
			string(">" + types.Float):                {f: dictGTFloatV1, Label: ">"},
			string(">=" + types.Float):               {f: dictGTEFloatV1, Label: ">="},
			string("<" + types.String):               {f: dictLTStringV1, Label: "<"},
			string("<=" + types.String):              {f: dictLTEStringV1, Label: "<="},
			string(">" + types.String):               {f: dictGTStringV1, Label: ">"},
			string(">=" + types.String):              {f: dictGTEStringV1, Label: ">="},
			string("<" + types.Dict):                 {f: dictLTDictV1, Label: "<"},
			string("<=" + types.Dict):                {f: dictLTEDictV1, Label: "<="},
			string(">" + types.Dict):                 {f: dictGTDictV1, Label: ">"},
			string(">=" + types.Dict):                {f: dictGTEDictV1, Label: ">="},
			string("&&" + types.Bool):                {f: dictAndBoolV1, Label: "&&"},
			string("||" + types.Bool):                {f: dictOrBoolV1, Label: "||"},
			string("&&" + types.Int):                 {f: dictAndIntV1, Label: "&&"},
			string("||" + types.Int):                 {f: dictOrIntV1, Label: "||"},
			string("&&" + types.Float):               {f: dictAndFloatV1, Label: "&&"},
			string("||" + types.Float):               {f: dictOrFloatV1, Label: "||"},
			string("&&" + types.String):              {f: dictAndStringV1, Label: "&&"},
			string("||" + types.String):              {f: dictOrStringV1, Label: "||"},
			string("&&" + types.Regex):               {f: dictAndRegexV1, Label: "&&"},
			string("||" + types.Regex):               {f: dictOrRegexV1, Label: "||"},
			string("&&" + types.Time):                {f: dictAndTimeV1, Label: "&&"},
			string("||" + types.Time):                {f: dictOrTimeV1, Label: "||"},
			string("&&" + types.Dict):                {f: dictAndDictV1, Label: "&&"},
			string("||" + types.Dict):                {f: dictOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike):           {f: dictAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike):           {f: dictOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):             {f: dictAndMapV1, Label: "&&"},
			string("||" + types.MapLike):             {f: dictOrMapV1, Label: "||"},
			string("+" + types.String):               {f: dictPlusStringV1, Label: "+"},
			string("+" + types.Int):                  {f: dictPlusIntV1, Label: "+"},
			string("-" + types.Int):                  {f: dictMinusIntV1, Label: "-"},
			string("*" + types.Int):                  {f: dictTimesIntV1, Label: "*"},
			string("/" + types.Int):                  {f: dictDividedIntV1, Label: "/"},
			string("+" + types.Float):                {f: dictPlusFloatV1, Label: "+"},
			string("-" + types.Float):                {f: dictMinusFloatV1, Label: "-"},
			string("*" + types.Float):                {f: dictTimesFloatV1, Label: "*"},
			string("/" + types.Float):                {f: dictDividedFloatV1, Label: "/"},
			string("*" + types.Time):                 {f: dictTimesTimeV1, Label: "*"},
			// fields
			"[]":                              {f: dictGetIndexV1},
			"length":                          {f: dictLengthV1},
			"{}":                              {f: dictBlockCallV1},
			"camelcase":                       {f: dictCamelcaseV1, Label: "camelcase"},
			"downcase":                        {f: dictDowncaseV1, Label: "downcase"},
			"upcase":                          {f: dictUpcaseV1, Label: "upcase"},
			"lines":                           {f: dictLinesV1, Label: "lines"},
			"split":                           {f: dictSplitV1, Label: "split"},
			"trim":                            {f: dictTrimV1, Label: "trim"},
			"keys":                            {f: dictKeysV1, Label: "keys"},
			"values":                          {f: dictValuesV1, Label: "values"},
			"where":                           {f: dictWhereV1, Label: "where"},
			"$whereNot":                       {f: dictWhereNotV1},
			"$all":                            {f: dictAllV1},
			"$none":                           {f: dictNoneV1},
			"$any":                            {f: dictAnyV1},
			"$one":                            {f: dictOneV1},
			"map":                             {f: dictMapV1},
			string("contains" + types.String): {f: dictContainsStringV1, Label: "contains"},
			string("contains" + types.Array(types.String)): {f: dictContainsArrayStringV1, Label: "contains"},
			string("contains" + types.Int):                 {f: dictContainsIntV1, Label: "contains"},
			string("contains" + types.Array(types.Int)):    {f: dictContainsArrayIntV1, Label: "contains"},
			string("find"): {f: dictFindV1, Label: "find"},
			// NOTE: the following functions are internal ONLY!
			// We have not yet decided if and how these may be exposed to users
			"notEmpty": {f: dictNotEmptyV1},
		},
		types.ArrayLike: {
			"[]":                     {f: arrayGetIndexV1},
			"first":                  {f: arrayGetFirstIndexV1},
			"last":                   {f: arrayGetLastIndexV1},
			"{}":                     {f: arrayBlockListV1},
			"${}":                    {f: arrayBlockV1},
			"length":                 {f: arrayLengthV1},
			"where":                  {f: arrayWhereV1},
			"$whereNot":              {f: arrayWhereNotV1},
			"$all":                   {f: arrayAllV1},
			"$none":                  {f: arrayNoneV1},
			"$any":                   {f: arrayAnyV1},
			"$one":                   {f: arrayOneV1},
			"map":                    {f: arrayMapV1},
			"duplicates":             {f: arrayDuplicatesV1},
			"fieldDuplicates":        {f: arrayFieldDuplicatesV1},
			"unique":                 {f: arrayUniqueV1},
			"difference":             {f: arrayDifferenceV1},
			"containsNone":           {f: arrayContainsNoneV1},
			"==":                     {Compiler: compileArrayOpArray("=="), f: tarrayCmpTarrayV1, Label: "=="},
			"!=":                     {Compiler: compileArrayOpArray("!="), f: tarrayNotTarrayV1, Label: "!="},
			"==" + string(types.Nil): {f: arrayCmpNilV1},
			"!=" + string(types.Nil): {f: arrayNotNilV1},
			"&&":                     {Compiler: compileLogicalArrayOp(types.ArrayLike, "&&")},
			"||":                     {Compiler: compileLogicalArrayOp(types.ArrayLike, "||")},
			// logical operations []<T> -- K
			string(types.Any + "&&" + types.Bool):      {f: arrayAndBoolV1, Label: "&&"},
			string(types.Any + "||" + types.Bool):      {f: arrayOrBoolV1, Label: "||"},
			string(types.Any + "&&" + types.Int):       {f: arrayAndIntV1, Label: "&&"},
			string(types.Any + "||" + types.Int):       {f: arrayOrIntV1, Label: "||"},
			string(types.Any + "&&" + types.Float):     {f: arrayAndFloatV1, Label: "&&"},
			string(types.Any + "||" + types.Float):     {f: arrayOrFloatV1, Label: "||"},
			string(types.Any + "&&" + types.String):    {f: arrayAndStringV1, Label: "&&"},
			string(types.Any + "||" + types.String):    {f: arrayOrStringV1, Label: "||"},
			string(types.Any + "&&" + types.Regex):     {f: arrayAndRegexV1, Label: "&&"},
			string(types.Any + "||" + types.Regex):     {f: arrayOrRegexV1, Label: "||"},
			string(types.Any + "&&" + types.ArrayLike): {f: arrayAndArrayV1, Label: "&&"},
			string(types.Any + "||" + types.ArrayLike): {f: arrayOrArrayV1, Label: "||"},
			string(types.Any + "&&" + types.MapLike):   {f: arrayAndMapV1, Label: "&&"},
			string(types.Any + "||" + types.MapLike):   {f: arrayOrMapV1, Label: "||"},
			// []T -- []T
			string(types.Bool + "==" + types.Array(types.Bool)):     {f: boolarrayCmpBoolarrayV1, Label: "=="},
			string(types.Bool + "!=" + types.Array(types.Bool)):     {f: boolarrayNotBoolarrayV1, Label: "!="},
			string(types.Int + "==" + types.Array(types.Int)):       {f: intarrayCmpIntarrayV1, Label: "=="},
			string(types.Int + "!=" + types.Array(types.Int)):       {f: intarrayNotIntarrayV1, Label: "!="},
			string(types.Float + "==" + types.Array(types.Float)):   {f: floatarrayCmpFloatarrayV1, Label: "=="},
			string(types.Float + "!=" + types.Array(types.Float)):   {f: floatarrayNotFloatarrayV1, Label: "!="},
			string(types.String + "==" + types.Array(types.String)): {f: stringarrayCmpStringarrayV1, Label: "=="},
			string(types.String + "!=" + types.Array(types.String)): {f: stringarrayNotStringarrayV1, Label: "!="},
			string(types.Regex + "==" + types.Array(types.Regex)):   {f: stringarrayCmpStringarrayV1, Label: "=="},
			string(types.Regex + "!=" + types.Array(types.Regex)):   {f: stringarrayNotStringarrayV1, Label: "!="},
			// []T -- T
			string(types.Bool + "==" + types.Bool):     {f: boolarrayCmpBoolV1, Label: "=="},
			string(types.Bool + "!=" + types.Bool):     {f: boolarrayNotBoolV1, Label: "!="},
			string(types.Int + "==" + types.Int):       {f: intarrayCmpIntV1, Label: "=="},
			string(types.Int + "!=" + types.Int):       {f: intarrayNotIntV1, Label: "!="},
			string(types.Float + "==" + types.Float):   {f: floatarrayCmpFloatV1, Label: "=="},
			string(types.Float + "!=" + types.Float):   {f: floatarrayNotFloatV1, Label: "!="},
			string(types.String + "==" + types.String): {f: stringarrayCmpStringV1, Label: "=="},
			string(types.String + "!=" + types.String): {f: stringarrayNotStringV1, Label: "!="},
			string(types.Regex + "==" + types.Regex):   {f: stringarrayCmpStringV1, Label: "=="},
			string(types.Regex + "!=" + types.Regex):   {f: stringarrayNotStringV1, Label: "!="},
			// []int/float
			string(types.Int + "==" + types.Float): {f: intarrayCmpFloatV1, Label: "=="},
			string(types.Int + "!=" + types.Float): {f: intarrayNotFloatV1, Label: "!="},
			string(types.Float + "==" + types.Int): {f: floatarrayCmpIntV1, Label: "=="},
			string(types.Float + "!=" + types.Int): {f: floatarrayNotIntV1, Label: "!="},
			// []string -- T
			string(types.String + "==" + types.Bool):  {f: stringarrayCmpBoolV1, Label: "=="},
			string(types.String + "!=" + types.Bool):  {f: stringarrayNotBoolV1, Label: "!="},
			string(types.String + "==" + types.Int):   {f: stringarrayCmpIntV1, Label: "=="},
			string(types.String + "!=" + types.Int):   {f: stringarrayNotIntV1, Label: "!="},
			string(types.String + "==" + types.Float): {f: stringarrayCmpFloatV1, Label: "=="},
			string(types.String + "!=" + types.Float): {f: stringarrayNotFloatV1, Label: "!="},
			// []T -- string
			string(types.Bool + "==" + types.String):  {f: boolarrayCmpStringV1, Label: "=="},
			string(types.Bool + "!=" + types.String):  {f: boolarrayNotStringV1, Label: "!="},
			string(types.Int + "==" + types.String):   {f: intarrayCmpStringV1, Label: "=="},
			string(types.Int + "!=" + types.String):   {f: intarrayNotStringV1, Label: "!="},
			string(types.Float + "==" + types.String): {f: floatarrayCmpStringV1, Label: "=="},
			string(types.Float + "!=" + types.String): {f: floatarrayNotStringV1, Label: "!="},
			// []T -- regex
			string(types.Int + "==" + types.Regex):    {f: intarrayCmpRegexV1, Label: "=="},
			string(types.Int + "!=" + types.Regex):    {f: intarrayNotRegexV1, Label: "!="},
			string(types.Float + "==" + types.Regex):  {f: floatarrayCmpRegexV1, Label: "=="},
			string(types.Float + "!=" + types.Regex):  {f: floatarrayNotRegexV1, Label: "!="},
			string(types.String + "==" + types.Regex): {f: stringarrayCmpRegexV1, Label: "=="},
			string(types.String + "!=" + types.Regex): {f: stringarrayNotRegexV1, Label: "!="},
			// NOTE: the following functions are internal ONLY!
			// We have not yet decided if and how these may be exposed to users
			"notEmpty": {f: arrayNotEmptyV1},
		},
		types.MapLike: {
			"[]":        {f: mapGetIndexV1},
			"length":    {f: mapLengthV1},
			"where":     {f: mapWhereV1},
			"$whereNot": {f: mapWhereNotV1},
			"{}":        {f: mapBlockCallV1},
			"keys":      {f: mapKeysV1, Label: "keys"},
			"values":    {f: mapValuesV1, Label: "values"},
			// {}T -- T
			string("&&" + types.Bool):      {f: chunkEqFalseV1, Label: "&&"},
			string("||" + types.Bool):      {f: chunkNeqTrueV1, Label: "||"},
			string("&&" + types.Int):       {f: mapAndIntV1, Label: "&&"},
			string("||" + types.Int):       {f: mapOrIntV1, Label: "||"},
			string("&&" + types.Float):     {f: mapAndFloatV1, Label: "&&"},
			string("||" + types.Float):     {f: mapOrFloatV1, Label: "||"},
			string("&&" + types.String):    {f: mapAndStringV1, Label: "&&"},
			string("||" + types.String):    {f: mapOrStringV1, Label: "||"},
			string("&&" + types.Regex):     {f: mapAndRegexV1, Label: "&&"},
			string("||" + types.Regex):     {f: mapOrRegexV1, Label: "||"},
			string("&&" + types.Time):      {f: mapAndTimeV1, Label: "&&"},
			string("||" + types.Time):      {f: mapOrTimeV1, Label: "||"},
			string("&&" + types.Dict):      {f: mapAndDictV1, Label: "&&"},
			string("||" + types.Dict):      {f: mapOrDictV1, Label: "||"},
			string("&&" + types.ArrayLike): {f: mapAndArrayV1, Label: "&&"},
			string("||" + types.ArrayLike): {f: mapOrArrayV1, Label: "||"},
			string("&&" + types.MapLike):   {f: mapAndMapV1, Label: "&&"},
			string("||" + types.MapLike):   {f: mapOrMapV1, Label: "||"},
		},
		types.ResourceLike: {
			// == / !=
			string("==" + types.Nil): {f: chunkEqFalseV1, Label: "=="},
			string("!=" + types.Nil): {f: chunkNeqTrueV1, Label: "!="},
			// fields
			"where":     {f: resourceWhereV1},
			"$whereNot": {f: resourceWhereNotV1},
			"map":       {f: resourceMapV1},
			"length":    {f: resourceLengthV1},
			"{}": {f: func(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
				return c.runBlock(bind, chunk.Function.Args[0], nil, ref)
			}},
			// TODO: [#32] unique builtin fields that need a long-term support in LR
			string(types.Resource("parse") + ".date"): {f: resourceDateV1},
		},
	}

	validateBuiltinFunctionsV1()
}

func validateBuiltinFunctionsV1() {
	missing := []string{}

	// dict must have all string methods supported
	dictFun := BuiltinFunctionsV1[types.Dict]
	if dictFun == nil {
		dictFun = map[string]chunkHandlerV1{}
	}

	stringFun := BuiltinFunctionsV1[types.String]
	if stringFun == nil {
		stringFun = map[string]chunkHandlerV1{}
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

func runResourceFunctionV1(c *MQLExecutorV1, bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
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

	resource, ok := c.runtime.Registry.Resources[info.Name]
	if !ok || resource == nil {
		return nil, 0, fmt.Errorf("cannot retrieve resource definition for resource %q", info.Name)
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

		c.triggerChain(ref, data)
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

			c.cache.Store(ref, &stepCache{
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
func BuiltinFunctionV1(typ types.Type, name string) (*chunkHandlerV1, error) {
	h, ok := BuiltinFunctionsV1[typ.Underlying()]
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
func (c *MQLExecutorV1) runBoundFunctionV1(bind *RawData, chunk *Chunk, ref int32) (*RawData, int32, error) {
	log.Trace().Int32("ref", ref).Str("id", chunk.Id).Msg("exec> run bound function")

	fh, err := BuiltinFunctionV1(bind.Type, chunk.Id)
	if err == nil {
		res, dref, err := fh.f(c, bind, chunk, ref)
		if res != nil {
			c.cache.Store(ref, &stepCache{Result: res})
		}
		if err != nil {
			c.cache.Store(ref, &stepCache{Result: &RawData{
				Error: err,
			}})
		}
		return res, dref, err
	}

	if bind.Type.IsResource() {
		return runResourceFunctionV1(c, bind, chunk, ref)
	}
	return nil, 0, err
}
