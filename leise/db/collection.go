package db

import (
	"encoding/base64"
	"regexp"
	"strings"

	"go.mondoo.io/mondoo/llx"

	"golang.org/x/crypto/blake2b"
)

func (c *Collection) checksum() string {
	originalID := c.Id
	c.Id = ""

	data, err := c.Marshal()
	if err != nil {
		panic("Failed to marshal Collection for checksum calculation. Critical failure.")
	}

	c.Id = originalID

	hash := blake2b.Sum512(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// UpdateID sets a new computed ID
func (c *Collection) UpdateID() {
	c.Id = c.checksum()
}

// HasLabel returns true if the collection has the given label
func (c *Collection) HasLabel(name string) bool {
	for i := range c.Labels {
		if c.Labels[i] == name {
			return true
		}
	}
	return false
}

// RemoveLabel from the list of labels
func (c *Collection) RemoveLabel(name string) {
	for i := 0; i < len(c.Labels); i++ {
		if c.Labels[i] == name {
			c.Labels = append(c.Labels[:i], c.Labels[i+1:]...)
		}
	}
}

// Cleanup fixes formatting issues on code and other fields
func (c *CollectionsBundle) Cleanup() {
	for i := range c.Collection {
		c.Collection[i].Cleanup()
	}
	for i := range c.Queries {
		c.Queries[i].Cleanup()
	}
	for i := range c.Code {
		cleanupCodeBundle(c.Code[i])
	}
}

// This method merges the data of the collection bundle
// organization and space are ignored and not changed on the original bundle
// TODO: This method does not verify if duplicates are within the queries and codes
// since they are having the same id anyway. Once stored, the data is flattened
func (c *CollectionsBundle) Merge(new *CollectionsBundle) {
	// just append data
	c.Code = append(c.Code, new.Code...)
	c.Queries = append(c.Queries, new.Queries...)
	c.Collection = append(c.Collection, new.Collection...)
}

// Cleanup fixes formatting issues on code and other fields
func (c *Collection) Cleanup() {
	c.Description = cleanText(c.Description)
	c.Title = cleanText(c.Title)
}

// Cleanup fixes formatting issues on code and other fields
func (c *Query) Cleanup() {
	c.Description = cleanText(c.Description)
	c.Title = cleanText(c.Title)
	c.Code = cleanText(c.Code)
}

// cleanupCodeBundle fixes formatting issues on code and other fields
func cleanupCodeBundle(c *llx.CodeBundle) {
	c.Source = cleanText(c.Source)
}

func cleanText(text string) string {
	return strings.TrimSpace(dedent(text))
}

// Dedent by https://github.com/lithammer/dedent MIT license:

var (
	whitespaceOnly    = regexp.MustCompile("(?m)^[ \t]+$")
	leadingWhitespace = regexp.MustCompile("(?m)(^[ \t]*)(?:[^ \t\n])")
)

// Dedent removes any common leading whitespace from every line in text.
//
// This can be used to make multiline strings to line up with the left edge of
// the display, while still presenting them in the source code in indented
// form.
func dedent(text string) string {
	var margin string

	text = whitespaceOnly.ReplaceAllString(text, "")
	indents := leadingWhitespace.FindAllStringSubmatch(text, -1)

	// Look for the longest leading string of spaces and tabs common to all
	// lines.
	for i, indent := range indents {
		if i == 0 {
			margin = indent[1]
		} else if strings.HasPrefix(indent[1], margin) {
			// Current line more deeply indented than previous winner:
			// no change (previous winner is still on top).
			continue
		} else if strings.HasPrefix(margin, indent[1]) {
			// Current line consistent with and no deeper than previous winner:
			// it's the new winner.
			margin = indent[1]
		} else {
			// Current line and previous winner have no common whitespace:
			// there is no margin.
			margin = ""
			break
		}
	}

	if margin != "" {
		text = regexp.MustCompile("(?m)^"+margin).ReplaceAllString(text, "")
	}
	return text
}
