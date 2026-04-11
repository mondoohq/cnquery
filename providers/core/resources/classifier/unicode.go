// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package classifier

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// CategoryDescriptions maps Unicode general category codes to their descriptions
var CategoryDescriptions = map[string]string{
	// Letter categories
	"Lu": "Letter, Uppercase",
	"Ll": "Letter, Lowercase",
	"Lt": "Letter, Titlecase",
	"Lm": "Letter, Modifier",
	"Lo": "Letter, Other",

	// Mark categories
	"Mn": "Mark, Nonspacing",
	"Mc": "Mark, Spacing Combining",
	"Me": "Mark, Enclosing",

	// Number categories
	"Nd": "Number, Decimal Digit",
	"Nl": "Number, Letter",
	"No": "Number, Other",

	// Punctuation categories
	"Pc": "Punctuation, Connector",
	"Pd": "Punctuation, Dash",
	"Ps": "Punctuation, Open",
	"Pe": "Punctuation, Close",
	"Pi": "Punctuation, Initial quote",
	"Pf": "Punctuation, Final quote",
	"Po": "Punctuation, Other",

	// Symbol categories
	"Sm": "Symbol, Math",
	"Sc": "Symbol, Currency",
	"Sk": "Symbol, Modifier",
	"So": "Symbol, Other",

	// Separator categories
	"Zs": "Separator, Space",
	"Zl": "Separator, Line",
	"Zp": "Separator, Paragraph",

	// Control categories
	"Cc": "Control",
	"Cf": "Format",
	"Cs": "Surrogate",
	"Co": "Private Use",
	"Cn": "Unassigned",
}

// CharacterInfo represents detailed information about a Unicode character
type CharacterInfo struct {
	// Position is the zero-based index of the character in the original string
	Position int `json:"position"`

	// Character is the actual Unicode character as a string
	Character string `json:"character"`

	// UnicodePoint is the Unicode code point in U+XXXX format
	UnicodePoint string `json:"unicodePoint"`

	// MajorCategory is the major Unicode category (single letter)
	MajorCategory string `json:"majorCategory"`

	// Category is the Unicode general category code (two-letter code)
	Category string `json:"category"`

	// Description is the human-readable description of the category
	Description string `json:"description"`

	// Rune is the raw Go rune (int32) value of the character
	Rune rune `json:"rune"`
}

// UnicodeClassifier provides Unicode character classification functionality
type UnicodeClassifier struct{}

// NewUnicodeClassifier creates a new Unicode classifier instance
func NewUnicodeClassifier() *UnicodeClassifier {
	return &UnicodeClassifier{}
}

// validateUTF8 is a helper function to validate UTF-8 strings
func validateUTF8(text string) error {
	if !utf8.ValidString(text) {
		return errors.New("invalid UTF-8 string")
	}
	return nil
}

// formatUnicodePoint efficiently formats a rune as a Unicode code point
func formatUnicodePoint(r rune) string {
	return fmt.Sprintf("U+%04X", r)
}

// getUnicodeCategory determines the Unicode general category for a rune
// see https://en.wikipedia.org/wiki/Unicode_character_property#General_Category
func (c *UnicodeClassifier) getUnicodeCategory(r rune) string {
	switch {
	case unicode.IsUpper(r):
		return "Lu"
	case unicode.IsLower(r):
		return "Ll"
	case unicode.IsTitle(r):
		return "Lt"
	case unicode.In(r, unicode.Lm):
		return "Lm"
	case unicode.IsLetter(r):
		return "Lo"
	case unicode.In(r, unicode.Mn):
		return "Mn"
	case unicode.In(r, unicode.Mc):
		return "Mc"
	case unicode.In(r, unicode.Me):
		return "Me"
	case unicode.IsDigit(r):
		return "Nd"
	case unicode.In(r, unicode.Nl):
		return "Nl"
	case unicode.IsNumber(r):
		return "No"
	case unicode.In(r, unicode.Pc):
		return "Pc"
	case unicode.In(r, unicode.Pd):
		return "Pd"
	case unicode.In(r, unicode.Ps):
		return "Ps"
	case unicode.In(r, unicode.Pe):
		return "Pe"
	case unicode.In(r, unicode.Pi):
		return "Pi"
	case unicode.In(r, unicode.Pf):
		return "Pf"
	case unicode.IsPunct(r):
		return "Po"
	case unicode.In(r, unicode.Sm):
		return "Sm"
	case unicode.In(r, unicode.Sc):
		return "Sc"
	case unicode.In(r, unicode.Sk):
		return "Sk"
	case unicode.IsSymbol(r):
		return "So"
	case unicode.In(r, unicode.Zs):
		return "Zs"
	case unicode.In(r, unicode.Zl):
		return "Zl"
	case unicode.In(r, unicode.Zp):
		return "Zp"
	case unicode.IsControl(r):
		return "Cc"
	case unicode.In(r, unicode.Cf):
		return "Cf"
	case unicode.In(r, unicode.Cs):
		return "Cs"
	case unicode.In(r, unicode.Co):
		return "Co"
	default:
		return "Cn"
	}
}

// ClassifyRune classifies a single rune and returns its category and description
func (c *UnicodeClassifier) ClassifyRune(r rune) (category, description string) {
	category = c.getUnicodeCategory(r)
	if desc, exists := CategoryDescriptions[category]; exists {
		description = desc
	} else {
		description = "Unknown category"
	}
	return
}

// ClassifyString analyzes all characters in a string and returns detailed information
func (c *UnicodeClassifier) ClassifyString(text string) ([]CharacterInfo, error) {
	if err := validateUTF8(text); err != nil {
		return nil, err
	}

	runeCount := utf8.RuneCountInString(text)
	results := make([]CharacterInfo, 0, runeCount)

	position := 0
	for _, r := range text {
		category, description := c.ClassifyRune(r)

		info := CharacterInfo{
			Position:      position,
			Character:     string(r),
			UnicodePoint:  formatUnicodePoint(r),
			MajorCategory: string(category[0]),
			Category:      category,
			Description:   description,
			Rune:          r,
		}

		results = append(results, info)
		position++
	}

	return results, nil
}

// GetCategorySummary returns a count of each Unicode category in the text
func (c *UnicodeClassifier) GetCategorySummary(text string) (map[string]int, error) {
	if err := validateUTF8(text); err != nil {
		return nil, err
	}

	categoryCounts := make(map[string]int)

	for _, r := range text {
		category := c.getUnicodeCategory(r)
		categoryCounts[category]++
	}

	return categoryCounts, nil
}
