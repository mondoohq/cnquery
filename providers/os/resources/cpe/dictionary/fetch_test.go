package dictionary

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"testing"
)

const cpeDictionaryURL = "https://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz"

func TestData(t *testing.T) {
	cpelist, err := load()
	if err != nil {
		t.Fatal(err)
	}

	entries := cpelist.Map()

	filteredEntries := map[string]map[string]string{}

	filterKeys := []string{"node.js", "python"}
	for _, key := range filterKeys {
		filteredEntries[key] = entries[key]
	}

	encodedData, err := json.Marshal(filteredEntries)
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile("testdata/cpe-dictionary.json", encodedData, 0700)
}

func load() (*CPEList, error) {
	r, err := os.Open("./testdata/official-cpe-dictionary_v2.3.xml")
	// r, err := os.Open("./testdata/cpe-dictionary_v2.3_test.xml")
	if err != nil {
		return nil, errors.New("unable to fetch CPE dictionary")
	}
	defer r.Close()

	return Decode(r)
}

func fetch() (*CPEList, error) {
	resp, err := http.Get(cpeDictionaryURL)
	if err != nil {
		return nil, errors.New("unable to fetch CPE dictionary")
	}
	defer resp.Body.Close()

	gzReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to decompress CPE dictionary: %w", err)
	}
	defer gzReader.Close()

	return Decode(gzReader)
}
