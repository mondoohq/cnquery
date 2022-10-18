package detector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMajorMinorParser(t *testing.T) {
	data := []struct {
		release string
		major   string
		minor   string
		other   string
	}{{
		release: "8.1.1911",
		major:   "8",
		minor:   "1",
		other:   "1911",
	}, {
		release: "5.11",
		major:   "5",
		minor:   "11",
		other:   "",
	}, {
		release: "6.9",
		major:   "6",
		minor:   "9",
		other:   "",
	}, {
		release: "7.5.1804",
		major:   "7",
		minor:   "5",
		other:   "1804",
	}, {
		release: "7",
		major:   "7",
		minor:   "",
		other:   "",
	}, {
		release: "12.04",
		major:   "12",
		minor:   "04",
		other:   "",
	}, {
		release: "7.0.0.2",
		major:   "7",
		minor:   "0",
		other:   "0.2",
	}, {
		release: "2019.4",
		major:   "2019",
		minor:   "4",
		other:   "",
	}, {
		release: "20200305",
		major:   "20200305",
		minor:   "",
		other:   "",
	}, {
		release: "2017.09",
		major:   "2017",
		minor:   "09",
		other:   "",
	}, {
		release: "3.7.0",
		major:   "3",
		minor:   "7",
		other:   "0",
	}, {
		release: "17763.720",
		major:   "17763",
		minor:   "720",
		other:   "",
	}}

	for i := range data {
		v := ParseOsVersion(data[i].release)
		assert.Equal(t, data[i].major, v.Major, data[i].release)
		assert.Equal(t, data[i].minor, v.Minor, data[i].release)
		assert.Equal(t, data[i].other, v.Other, data[i].release)
	}
}
