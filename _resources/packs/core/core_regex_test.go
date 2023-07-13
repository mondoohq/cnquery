package core

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Adding unit tests for regex

var lre = mqlRegex{}

func testRegex(t *testing.T, f func() (string, error), matches, fails []string) {
	r, err := f()
	require.NoError(t, err)

	re, err := regexp.Compile("^" + r + "$")
	require.NoError(t, err)

	for i := range matches {
		s := matches[i]
		t.Run("matches "+s, func(t *testing.T) {
			assert.True(t, re.MatchString(s))
		})
	}

	for i := range fails {
		s := fails[i]
		t.Run("fails "+s, func(t *testing.T) {
			assert.False(t, re.MatchString(s))
		})
	}
}

func TestResource_RegexEmail(t *testing.T) {
	// The following tests are copied from:
	//   https://en.wikipedia.org/wiki/Email_address
	//   Example section
	matches := []string{
		"simple@example.com",
		"very.common@example.com",
		"disposable.style.email.with+symbol@example.com",
		"other.email-with-hyphen@example.com",
		"fully-qualified-domain@example.com",
		"user.name+tag+sorting@example.com",
		"x@example.com",
		"example-indeed@strange-example.com",
		"test/test@test.com",
		"admin@mailserver1", // local domain name with no TLD, although ICANN highly discourages dotless email addresses
		"example@s.example",
		"\" \"@example.org",
		"\"john..doe\"@example.org",
		"mailhost!username@example.org",
		"user%example.com@example.org",
		"user-@example.org",
		"jsmith@[192.168.2.1]",
		"jsmith@[IPv6:2001:db8::1]",
	}

	fails := []string{
		"Abc.example.com",
		"A@b@c@example.com",
		"a\"b(c)d,e:f;g<h>i[j\\k]l@example.com",
		"just\"not\"right@example.com",
		"this is\"not\\allowed@example.com",
		"this\\ still\\\"not\\\\allowed@example.com",
		"1234567890123456789012345678901234567890123456789012345678901234+x@example.com",
		"i_like_underscore@but_its_not_allowed_in_this_part.example.com",
		"QA[icon]CHOCOLATE[icon]@test.com",
	}

	testRegex(t, lre.GetEmail, matches, fails)
}
