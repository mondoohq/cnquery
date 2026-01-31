package resources_test

import (
    "testing"
    "go.mondoo.com/cnquery/v12/providers-sdk/v1/testutils"
    "github.com/stretchr/testify/require"
    "fmt"
    "github.com/lithammer/fuzzysearch/fuzzy"
)

func TestDebugFuzzy(t *testing.T) {
    x := testutils.InitTester(testutils.LinuxMock())

    // First get the params to see its type
    res := x.TestQuery(t, "parse.json('/dummy.json').params")
    require.NotEmpty(t, res)

    m := res[0].Data.Value.(map[string]interface{})
    fmt.Printf("Map has %d keys\n", len(m))
    for k := range m {
        fmt.Printf("  Key: %q\n", k)
    }

    // Test LevenshteinDistance directly
    fmt.Printf("\nLevenshtein tests:\n")
    fmt.Printf("  hallo vs hello: %d\n", fuzzy.LevenshteinDistance("hallo", "hello"))
    fmt.Printf("  Hello vs hello: %d\n", fuzzy.LevenshteinDistance("Hello", "hello"))
}
