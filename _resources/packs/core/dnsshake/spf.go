package dnsshake

import (
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

// Sender Policy Framework (SPF) for Authorizing Use of Domains in Email, Version 1
// https://datatracker.ietf.org/doc/html/rfc7208
//
// 1. Parse SPF records
// 2. Validate SPF record
// 3. Validate EMAIL + DOMAIN

// Model after https://datatracker.ietf.org/doc/html/rfc7208#section-12
var spfLexer = lexer.MustSimple([]lexer.Rule{
	{`Header`, `v=`, nil},
	{`Version`, `spf\d`, nil},
	{"whitespace", `\s+`, nil},
	{`Colon`, `[:]`, nil},
	{`Slash`, `[/]`, nil},
	{`Equal`, `[=]`, nil},
	{`Mechanism`, `\b(all|include|a|mx|ptr|ip4|ip6|exists)\b`, nil},
	{`Modifier`, `\b(redirect|exp)\b`, nil},
	{`String`, `[^+\-:\s=\/][\w.%\-+{}]+`, nil},
	{`Qualifier`, `[\+\-~?]`, nil},
	{`Number`, `\d+`, nil},
})

// nolint: govet
type SpfRecord struct {
	Version    string      `"v=" @Version`
	Directives []Directive `@@*`
	Modifiers  []Modifier  `@@*`
}

// nolint: govet
type Directive struct {
	Qualifier string `(@Qualifier)?`
	Mechanism string `@Mechanism`
	Value     string `(":" @String)?`
	CIDR      string `("/" @String)?`
}

// nolint: govet
type Modifier struct {
	Modifier string `@Modifier "="`
	Value    string `@String`
}

var spfParser = participle.MustBuild(&SpfRecord{},
	participle.Lexer(spfLexer),
)

func NewSpf() *spf {
	return &spf{}
}

type spf struct{}

func (s *spf) Parse(txt string) (*SpfRecord, error) {
	lines := strings.Split(txt, " ")
	spf := &SpfRecord{}

	err := spfParser.Parse("", strings.NewReader(strings.Join(lines, "\n")), spf)
	if err != nil {
		return nil, err
	}
	return spf, nil
}
