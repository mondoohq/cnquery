package dnsshake

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpfAst(t *testing.T) {
	spf := NewSpf()
	ast, err := spf.Parse("v=spf1")
	require.NoError(t, err)
	fmt.Printf("%v", ast)

	ast, err = spf.Parse("v=spf1 mx/30 mx:example.org/30 -all")
	require.NoError(t, err)
	fmt.Printf("%v", ast)

	ast, err = spf.Parse("v=spf1 mx -all exp=explain._spf.%{d}")
	require.NoError(t, err)
	fmt.Printf("%v", ast)
}

// TestSpfAstParser tests the examples from https://datatracker.ietf.org/doc/html/rfc7208#appendix-A.1
func TestRfcRecords(t *testing.T) {
	type test struct {
		Title        string
		DnsTxtRecord string
		Expected     *SpfRecord
	}

	testCases := []test{
		//  Simple Examples
		{
			Title:        "any <ip> passes",
			DnsTxtRecord: "v=spf1 +all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{{
					Qualifier: "+",
					Mechanism: "all",
				}},
			},
		},
		{
			Title:        "hosts 192.0.2.10 and 192.0.2.11 pass",
			DnsTxtRecord: "v=spf1 a -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "a",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "no sending hosts pass since example.org has no A records",
			DnsTxtRecord: "v=spf1 a:example.org -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "a",
						Value:     "example.org",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "sending hosts 192.0.2.129 and 192.0.2.130 pass",
			DnsTxtRecord: "v=spf1 mx -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "mx",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "sending host 192.0.2.140 passes",
			DnsTxtRecord: "v=spf1 mx:example.org -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "mx",
						Value:     "example.org",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "any sending host in 192.0.2.128/30 or 192.0.2.140/30 passes",
			DnsTxtRecord: "v=spf1 mx/30 mx:example.org/30 -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "mx",
						CIDR:      "30",
					},
					{
						Mechanism: "mx",
						Value:     "example.org",
						CIDR:      "30",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "sending host 192.0.2.65 passes (reverse DNS is valid and is in example.com)",
			DnsTxtRecord: "v=spf1 ptr -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "ptr",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "ending host 192.0.2.65 fails",
			DnsTxtRecord: "v=spf1 ip4:192.0.2.128/28 -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "ip4",
						Value:     "192.0.2.128",
						CIDR:      "28",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		// Multiple Domain Example
		{
			Title:        "Multiple Domain Example",
			DnsTxtRecord: "v=spf1 include:example.com include:example.net -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "include",
						Value:     "example.com",
					},
					{
						Mechanism: "include",
						Value:     "example.net",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "Redirect",
			DnsTxtRecord: "v=spf1 +mx redirect=_spf.example.com",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Qualifier: "+",
						Mechanism: "mx",
					},
				},
				Modifiers: []Modifier{
					{
						Modifier: "redirect",
						Value:    "_spf.example.com",
					},
				},
			},
		},
		{
			Title:        "Exists",
			DnsTxtRecord: "v=spf1 exists:%{ir}.%{l1r+-}._spf.%{d} -all",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "exists",
						Value:     "%{ir}.%{l1r+-}._spf.%{d}",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
			},
		},
		{
			Title:        "Explain",
			DnsTxtRecord: "v=spf1 mx -all exp=explain._spf.%{d}",
			Expected: &SpfRecord{
				Version: "spf1",
				Directives: []Directive{
					{
						Mechanism: "mx",
					},
					{
						Qualifier: "-",
						Mechanism: "all",
					},
				},
				Modifiers: []Modifier{
					{
						Modifier: "exp",
						Value:    "explain._spf.%{d}",
					},
				},
			},
		},
	}

	spf := NewSpf()
	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Title, func(t *testing.T) {
			ast, err := spf.Parse(tc.DnsTxtRecord)
			assert.NoError(t, err)

			// check that the data was parsed as expected
			assert.EqualValues(t, tc.Expected, ast)
		})
	}
}
