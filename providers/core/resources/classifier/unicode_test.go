// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package classifier

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClassifier(t *testing.T) {
	classifier := NewUnicodeClassifier()
	require.NotNil(t, classifier, "NewUnicodeClassifier() should not return nil")
}

func TestClassifyRune(t *testing.T) {
	classifier := NewUnicodeClassifier()

	testCases := []struct {
		name         string
		rune         rune
		expectedCat  string
		expectedDesc string
	}{
		{"uppercase letter", 'A', "Lu", "Letter, Uppercase"},
		{"lowercase letter", 'a', "Ll", "Letter, Lowercase"},
		{"digit", '1', "Nd", "Number, Decimal Digit"},
		{"space", ' ', "Zs", "Separator, Space"},
		{"punctuation", '!', "Po", "Punctuation, Other"},
		{"math symbol", '+', "Sm", "Symbol, Math"},
		{"currency symbol", '$', "Sc", "Symbol, Currency"},
		{"control character", '\t', "Cc", "Control"},
		{"greek lowercase", 'α', "Ll", "Letter, Lowercase"},
		{"greek uppercase", 'Ω', "Lu", "Letter, Uppercase"},
		{"chinese character", '中', "Lo", "Letter, Other"},
		{"emoji", '🙂', "So", "Symbol, Other"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			category, description := classifier.ClassifyRune(tc.rune)
			assert.Equal(t, tc.expectedCat, category, "Unexpected category for rune %c", tc.rune)
			assert.Equal(t, tc.expectedDesc, description, "Unexpected description for rune %c", tc.rune)
		})
	}
}

func TestClassifyString(t *testing.T) {
	classifier := NewUnicodeClassifier()

	t.Run("simple ASCII string", func(t *testing.T) {
		results, err := classifier.ClassifyString("A1!")
		require.NoError(t, err)
		require.Len(t, results, 3, "Should have 3 results")

		expected := []struct {
			char     string
			category string
			position int
		}{
			{"A", "Lu", 0},
			{"1", "Nd", 1},
			{"!", "Po", 2},
		}

		for i, exp := range expected {
			assert.Equal(t, exp.char, results[i].Character, "Character mismatch at position %d", i)
			assert.Equal(t, exp.category, results[i].Category, "Category mismatch at position %d", i)
			assert.Equal(t, exp.position, results[i].Position, "Position mismatch at position %d", i)
			assert.Contains(t, results[i].UnicodePoint, "U+", "Unicode point should be formatted correctly")
		}
	})

	t.Run("unicode string", func(t *testing.T) {
		results, err := classifier.ClassifyString("αβγ")
		require.NoError(t, err)
		require.Len(t, results, 3, "Should have 3 results for Greek letters")

		for i, result := range results {
			assert.Equal(t, "Ll", result.Category, "All Greek letters should be lowercase at position %d", i)
			assert.Equal(t, i, result.Position, "Position should match index")
		}
	})

	t.Run("empty string", func(t *testing.T) {
		results, err := classifier.ClassifyString("")
		require.NoError(t, err)
		assert.Empty(t, results, "Empty string should return empty results")
	})
}

func TestGetCategorySummary(t *testing.T) {
	classifier := NewUnicodeClassifier()

	t.Run("mixed content string", func(t *testing.T) {
		summary, err := classifier.GetCategorySummary("Hello World! 123")
		require.NoError(t, err)

		expected := map[string]int{
			"Lu": 2,
			"Ll": 8,
			"Zs": 2,
			"Po": 1,
			"Nd": 3,
		}

		assert.Equal(t, expected, summary, "Category summary should match expected counts")
	})

	t.Run("empty string", func(t *testing.T) {
		summary, err := classifier.GetCategorySummary("")
		require.NoError(t, err)
		assert.Empty(t, summary, "Empty string should return empty summary")
	})
}

func TestCategoryDescriptions(t *testing.T) {
	expectedCategories := []string{
		"Lu", "Ll", "Lt", "Lm", "Lo",
		"Mn", "Mc", "Me",
		"Nd", "Nl", "No",
		"Pc", "Pd", "Ps", "Pe", "Pi", "Pf", "Po",
		"Sm", "Sc", "Sk", "So",
		"Zs", "Zl", "Zp",
		"Cc", "Cf", "Cs", "Co", "Cn",
	}

	for _, cat := range expectedCategories {
		t.Run(cat, func(t *testing.T) {
			desc, exists := CategoryDescriptions[cat]
			assert.True(t, exists, "Category %s should exist in CategoryDescriptions", cat)
			assert.NotEmpty(t, desc, "Category %s should have a non-empty description", cat)
		})
	}
}

func TestComplexScenarios(t *testing.T) {
	classifier := NewUnicodeClassifier()

	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, classifier *UnicodeClassifier, input string)
	}{
		{
			name:  "mixed script text",
			input: "Hello 世界 αβγ 123 🌟",
			validate: func(t *testing.T, c *UnicodeClassifier, input string) {
				results, err := c.ClassifyString(input)
				require.NoError(t, err)
				assert.Greater(t, len(results), 10, "Should have multiple characters")

				summary, err := c.GetCategorySummary(input)
				require.NoError(t, err)
				assert.Greater(t, len(summary), 3, "Should have multiple categories")
			},
		},
		{
			name:  "emoji sequence",
			input: "🏳️\u200d🌈👨\u200d👩\u200d👧\u200d👦",
			validate: func(t *testing.T, c *UnicodeClassifier, input string) {
				results, err := c.ClassifyString(input)
				require.NoError(t, err)
				assert.Greater(t, len(results), 0, "Should classify emoji sequences")
			},
		},
		{
			name:  "mathematical symbols",
			input: "∑∞∂∇√±×÷",
			validate: func(t *testing.T, c *UnicodeClassifier, input string) {
				results, err := c.ClassifyString(input)
				require.NoError(t, err)
				for _, result := range results {
					assert.Equal(t, "Sm", result.Category, "Mathematical symbols should be classified as Sm")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, classifier, tt.input)
		})
	}
}
