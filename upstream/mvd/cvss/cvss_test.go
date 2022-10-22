package cvss

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCvss2Parsing(t *testing.T) {
	c, err := New("1.2/AV:L/AC:H/Au:N/C:N/I:N/A:P")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")

	assert.Equal(t, float32(1.2), c.Score, "score properly detected")
	assert.Equal(t, "Low", c.Severity().String(), "severity properly extracted")
	assert.Equal(t, "2.0", c.Version(), "vector format version")

	metrics, err := c.Metrics()
	assert.Nil(t, err, "metrics could be extracted")

	assert.Equal(t, "L", metrics["AV"], "AV properly detected")
	assert.Equal(t, "H", metrics["AC"], "AC properly detected")
	assert.Equal(t, "N", metrics["AU"], "AU properly detected")
	assert.Equal(t, "N", metrics["C"], "C properly detected")
	assert.Equal(t, "N", metrics["I"], "I properly detected")
	assert.Equal(t, "P", metrics["A"], "A properly detected")
}

func TestCvss2Parsing2(t *testing.T) {
	c, err := New("7.5/AV:N/AC:L/Au:N/C:P/I:P/A:P")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")
	assert.Equal(t, "2.0", c.Version(), "vector format version")

	assert.Equal(t, float32(7.5), c.Score, "score properly detected")
	assert.Equal(t, "High", c.Severity().String(), "severity properly extracted")
}

func TestCvss2Parsing3(t *testing.T) {
	c, err := New("7.5/AV:N/AC:L/Au:N/C:P/I:P/A:P")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")
	assert.Equal(t, "2.0", c.Version(), "vector format version")

	assert.Equal(t, float32(7.5), c.Score, "score properly detected")
	assert.Equal(t, "High", c.Severity().String(), "severity properly extracted")
}

func TestCvss30Parsing(t *testing.T) {
	c, err := New("8.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:R/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")

	assert.Equal(t, float32(8.8), c.Score, "score properly detected")
	assert.Equal(t, "High", c.Severity().String(), "severity properly extracted")
	assert.Equal(t, "3.0", c.Version(), "vector format version")

	metrics, err := c.Metrics()
	assert.Nil(t, err, "metrics could be extracted")

	assert.Equal(t, "N", metrics["AV"], "AV properly detected")
	assert.Equal(t, "L", metrics["AC"], "AC properly detected")
	assert.Equal(t, "N", metrics["PR"], "PR properly detected")
	assert.Equal(t, "R", metrics["UI"], "UI properly detected")
	assert.Equal(t, "U", metrics["S"], "S properly detected")
	assert.Equal(t, "H", metrics["C"], "C properly detected")
	assert.Equal(t, "H", metrics["I"], "I properly detected")
	assert.Equal(t, "H", metrics["A"], "A properly detected")
}

func TestCvss30Parsing2(t *testing.T) {
	c, err := New("8.1/CVSS:3.0/AV:A/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:N")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")

	assert.Equal(t, float32(8.1), c.Score, "score properly detected")
	assert.Equal(t, "High", c.Severity().String(), "severity properly extracted")
	assert.Equal(t, "3.0", c.Version(), "vector format version")

	metrics, err := c.Metrics()
	assert.Nil(t, err, "metrics could be extracted")

	assert.Equal(t, "A", metrics["AV"], "AV properly detected")
	assert.Equal(t, "L", metrics["AC"], "AC properly detected")
	assert.Equal(t, "N", metrics["PR"], "PR properly detected")
	assert.Equal(t, "N", metrics["UI"], "UI properly detected")
	assert.Equal(t, "U", metrics["S"], "S properly detected")
	assert.Equal(t, "H", metrics["C"], "C properly detected")
	assert.Equal(t, "H", metrics["I"], "I properly detected")
	assert.Equal(t, "N", metrics["A"], "A properly detected")
}

func TestCvss30Parsing3(t *testing.T) {
	c, err := New("9.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")
	assert.Equal(t, "3.0", c.Version(), "vector format version")

	assert.Equal(t, float32(9.8), c.Score, "score properly detected")
	assert.Equal(t, "Critical", c.Severity().String(), "severity properly extracted")
}

func TestCvss31Parsing1(t *testing.T) {
	c, err := New("7.5/CVSS:3.1/AV:N/AC:H/PR:N/UI:R/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.True(t, c.Verify(), "valid cvss vector")
	assert.Equal(t, "3.1", c.Version(), "vector format version")

	assert.Equal(t, float32(7.5), c.Score, "score properly detected")
	assert.Equal(t, "High", c.Severity().String(), "severity properly extracted")
}

func TestCvss3Comparison(t *testing.T) {
	c, err := New("9.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")
	d, err := New("2.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")

	assert.Equal(t, 1, c.Compare(d), "c > d")
	assert.Equal(t, -1, d.Compare(c), "d > c")
	assert.Equal(t, 0, c.Compare(c), "c == c")
}

func TestCvss3ParseEmpty(t *testing.T) {
	c, err := New("")
	assert.NotNil(t, err, "could not parse the cvss vector")
	assert.Nil(t, c, "no object returned")
}

func TestCvssNone(t *testing.T) {
	c, err := New("0.0/CVSS:3.0")
	assert.Nil(t, err, "could parse the cvss vector")
	assert.Equal(t, float32(0.0), c.Score, "score properly detected")
	assert.Equal(t, "None", c.Severity().String(), "severity properly extracted")
	assert.Equal(t, "3.0", c.Version(), "vector format version")
}

func TestCvssVector(t *testing.T) {
	b, err := New("5.8/AV:N/AC:M/Au:N/C:P/I:P/A:N")
	assert.Nil(t, err, "could parse the cvss vector")

	assert.Equal(t, float32(5.8), b.Score, "score properly detected")
	assert.Equal(t, "Medium", b.Severity().String(), "severity properly extracted")
}

func TestCvssVectorWithTrailingSlash(t *testing.T) {
	b, err := New("4.3/AV:N/AC:M/Au:N/C:P/I:N/A:N/")
	assert.Nil(t, err, "could parse the cvss vector")

	assert.Equal(t, float32(4.3), b.Score, "score properly detected")
	assert.Equal(t, "Medium", b.Severity().String(), "severity properly extracted")
}

func TestCvssVectorWithTrailingSpace(t *testing.T) {
	b, err := New("6.8/AV:N/AC:M/Au:N/C:P/I:P/A:P ")
	assert.Nil(t, err, "could parse the cvss vector")

	assert.Equal(t, float32(6.8), b.Score, "score properly detected")
	assert.Equal(t, "Medium", b.Severity().String(), "severity properly extracted")
}

func TestMaxCvss(t *testing.T) {
	b, err := New("1.2/AV:L/AC:H/Au:N/C:N/I:N/A:P")
	assert.Nil(t, err, "could parse the cvss vector")
	c, err := New("9.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")
	d, err := New("2.8/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Nil(t, err, "could parse the cvss vector")

	max, err := MaxScore([]*Cvss{b, c, d})
	assert.Nil(t, err, "could determine max cvss vector")

	assert.Equal(t, float32(9.8), max.Score, "score properly detected")
	assert.Equal(t, "Critical", max.Severity().String(), "severity properly extracted")
}

func TestMaxCvss2(t *testing.T) {
	list := []*Cvss{
		&Cvss{
			Vector: "7.5/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			Source: "cve://nvd/2017",
			Score:  7.5,
		},
		&Cvss{
			Vector: "7.7/CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:C/C:N/I:N/A:H",
			Source: "cve://nvd/2017",
			Score:  7.7,
		},
		&Cvss{
			Vector: "6.5/CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:N/A:H",
			Source: "cve://nvd/2017",
			Score:  6.5,
		},
		&Cvss{
			Vector: "4.9/CVSS:3.0/AV:N/AC:L/PR:H/UI:N/S:U/C:N/I:N/A:H",
			Source: "cve://nvd/2017",
			Score:  4.9,
		},
		&Cvss{
			Vector: "4.3/CVSS:3.0/AV:N/AC:L/PR:L/UI:N/S:U/C:N/I:L/A:N",
			Source: "cve://nvd/2017",
			Score:  4.3,
		},
		&Cvss{
			Vector: "6.6/CVSS:3.0/AV:N/AC:H/PR:H/UI:N/S:U/C:H/I:H/A:H",
			Source: "cve://nvd/2017",
			Score:  6.6,
		},
		&Cvss{
			Vector: "5.3/CVSS:3.0/AV:N/AC:H/PR:L/UI:N/S:U/C:H/I:N/A:N",
			Source: "cve://nvd/2017",
			Score:  5.3,
		},
		&Cvss{
			Vector: "7.5/CVSS:3.0/AV:N/AC:L/PR:N/UI:N/S:U/C:N/I:N/A:H",
			Source: "cve://nvd/2017",
			Score:  7.5,
		},
	}

	max, err := MaxScore(list)
	assert.Nil(t, err, "could determine max cvss vector")
	assert.Equal(t, float32(7.7), max.Score, "score properly detected")
	assert.Equal(t, "High", max.Severity().String(), "severity properly extracted")
}
