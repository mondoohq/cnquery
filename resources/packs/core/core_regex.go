package core

import "regexp"

func (p *mqlRegex) id() (string, error) {
	return "time", nil
}

// A ton of glory goes to:
// - https://ihateregex.io/expr where many of these regexes come from

func (p *mqlRegex) GetIpv4() (string, error) {
	return "(\\b25[0-5]|\\b2[0-4][0-9]|\\b[01]?[0-9][0-9]?)(\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}", nil
}

func (p *mqlRegex) GetIpv6() (string, error) {
	// This needs a better approach, possibly using advanced regex features if we can...
	return "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))", nil
}

// TODO: needs to be much more precise
func (p *mqlRegex) GetUrl() (string, error) {
	return "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\\b([-a-zA-Z0-9()!@:%_\\+.~#?&\\/\\/=]*)", nil
}

// TODO: can't figure this one out yet, needs work before getting exposed
// Adopted from:
//   https://stackoverflow.com/a/20046959/1195583
// Note:
// - there is a difference between Domain names and Host names, see:
//   https://stackoverflow.com/questions/2180465/can-domain-name-subdomains-have-an-underscore-in-it
//   - For example, in the case of emails and URLs we use internet domain names
//     ie host names
// - the reNoTldHostname allows for domain names with no TLD, even though this
//   is discouraged (and it kind of matches all kinds of things). Useful
//   for e.g. email regex
const reLDHLabel = "([0-9][a-zA-Z]|[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9]|[a-zA-Z][0-9]|[a-zA-Z]{1,2})"
const reUrlDomain = reLDHLabel + "(\\." + reLDHLabel + ")+"
const reNoTldHostname = reLDHLabel + "(\\." + reLDHLabel + ")*"

var rexUrlDomain = regexp.MustCompile(reUrlDomain)

// const reDomainLabel = "... needs work"

func (p *mqlRegex) GetDomain() (string, error) {
	return reUrlDomain, nil
}

// Email Regex
// ===========
// overall:     https://en.wikipedia.org/wiki/Email_address
//   addr-spec       =   local-part "@" domain
//   local-part      =   dot-atom / quoted-string / obs-local-part
//
// utf8 email:  https://datatracker.ietf.org/doc/html/rfc6531
// utf8 coding: https://en.wikipedia.org/wiki/UTF-8
//
// Unquoted:
//   Atext:       https://datatracker.ietf.org/doc/html/rfc5322#section-3.2.3
//   [a-z0-9!#$%&'*+-/=?^_`{|}~] and '.' (not first, not last, not in sequence)
//   any unicode above ascii, encoded as UTF8
//
// Quoted:
//   https://datatracker.ietf.org/doc/html/rfc5321#section-4.1.2
//   https://datatracker.ietf.org/doc/html/rfc6531#section-3.3
//   Qtext = %d32-33 / %d35-91 / %d93-126 / UTF8-nonascii
//
// Domain:
//   https://datatracker.ietf.org/doc/html/rfc5322#section-3.4.1
//   Dtext = %d33-90 / %d94-126 / obs-dtext
//   Weird: dtext may be empty, which is very weird. Implementing it with
//   this constraint in place, but it may need review.
//
//   Additionally: it's not in these RFCs, but the domain is further resricted
//   by https://datatracker.ietf.org/doc/html/rfc3696. It is also not a domain
//   name in the context of DNS, see these clarifications:
//   - https://www.rfc-editor.org/rfc/rfc2181#section-11
//   - https://stackoverflow.com/questions/2180465/can-domain-name-subdomains-have-an-underscore-in-it
//
// Limitation: I suspect we may also need to support rfc5322, which includes
// more characters in its qtext definition. However this document and the wiki
// are at odds with each other and I can't make heads or tails out of it
// (eg the wiki says qtext support HT, but rfc5322 clearly says it doesn't).
// This needs follow-up work, but it's also an extreme edge-case afaics.
//
// Limitation: We do not check the length of the individual parts ie:
// - local part can be up to 64 octets
// - domain can be up to 255 octets
// - also domain labels may only be up to 63 octets
//
// TODO: IPv4 + IPv6 domains, comments
const reAtextAscii = "[a-z0-9!#$%&'*+-/=?^_`{|}~]"
const reUtf8NonAscii = "[\\xC0-\\xDF][\\x80-\\xBF]|[\\xE0-\\xEF][\\x80-\\xBF]{2}|[\\xF0-\\xF7][\\x80-\\xBF]{3}"
const reQtextAscii = "[ !#-\\[\\]-~]"
const reQtext = "\"(" + reQtextAscii + "|" + reUtf8NonAscii + "){1,63}\""
const reAtext = "(" + reAtextAscii + "|" + reUtf8NonAscii + "){1,63}"
const reDotAtom = reAtext + "(\\." + reAtext + ")*"
const reEmailLocal = "(" + reQtext + "|" + reDotAtom + ")"
const dText = "[!-Z^-~]"
const reDomainLiteral = "\\[" + dText + "{0,255}\\]"
const reEmailDomain = "(" + reNoTldHostname + "|" + reDomainLiteral + ")"

const reEmail = reEmailLocal + "@" + reEmailDomain

// TODO: this needs serious work! re-use aspects from the domain recognition
func (p *mqlRegex) GetEmail() (string, error) {
	return reEmail, nil
}

func (p *mqlRegex) GetMac() (string, error) {
	return "[a-fA-F0-9]{2}(:[a-fA-F0-9]{2}){5}", nil
}

func (p *mqlRegex) GetUuid() (string, error) {
	return "[0-9a-fA-F]{8}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{12}", nil
}

func (p *mqlRegex) GetEmoji() (string, error) {
	// weather:  02600 ☀  - 027BF ➿
	// emoji:    1F300 🌀 - 1F6FC 🛼
	// extras:   1F900 🤀  - 1F9FF 🧿
	// more:     1FA70 🩰 - 1FAF6 heart hands
	return "[☀-➿🌀-🛼🤀-🧿🩰-🫶]", nil
}

func (p *mqlRegex) GetSemver() (string, error) {
	return "(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?", nil
}

func (p *mqlRegex) GetCreditCard() (string, error) {
	// For a complete list see:
	// https://stackoverflow.com/questions/9315647/regex-credit-card-number-tests
	return "(^|[^0-9])(" +
		"(4[0-9]{12}(?:[0-9]{3})?)|" + // VISA
		"(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14})" + // VISA Master Card
		"((?:5[1-5][0-9]{2}|222[1-9]|22[3-9][0-9]|2[3-6][0-9]{2}|27[01][0-9]|2720)[0-9]{12})|" + // Mastercard?
		"(3[47][0-9]{13})|" + // Amex Card
		"(3(?:0[0-5]|[68][0-9])[0-9]{11})|" + // Diner's Club
		"(6(?:011|5[0-9]{2})[0-9]{12})|" + // Discover?
		"((?:2131|1800|35\\d{3})\\d{11})" + // JCB card
		")($|[^0-9])", nil
}
