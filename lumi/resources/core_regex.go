package resources

func (p *lumiRegex) id() (string, error) {
	return "time", nil
}

// A ton of glory goes to:
// - https://ihateregex.io/expr where many of these regexes come from

func (p *lumiRegex) GetIpv4() (string, error) {
	return "(\\b25[0-5]|\\b2[0-4][0-9]|\\b[01]?[0-9][0-9]?)(\\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}", nil
}

func (p *lumiRegex) GetIpv6() (string, error) {
	// This needs a better approach, possibly using advanced regex features if we can...
	return "(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))", nil
}

// TODO: needs to be much more precise
func (p *lumiRegex) GetUrl() (string, error) {
	return "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{1,256}\\.[a-zA-Z0-9()]{1,6}\\b([-a-zA-Z0-9()!@:%_\\+.~#?&\\/\\/=]*)", nil
}

// TODO: can't figure this one out yet, needs work before getting exposed
// Adopted from:
//   https://stackoverflow.com/a/20046959/1195583
const reDomain = "(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\\.([a-zA-Z]{2,}|[a-zA-Z0-9-]{2,30}\\.[a-zA-Z]{2,3})"

func (p *lumiRegex) GetDomain() (string, error) {
	return reDomain, nil
}

const reAtextAscii = "[a-z0-9!#$%&'*+-/=?^_`{|}~]"
const reNonAscii = "[\\xC0-\\xDF][\\x80-\\xBF]|[\\xE0-\\xEF][\\x80-\\xBF]{2}|[\\xF0-\\xF7][\\x80-\\xBF]{3}"
const reAtext = "(" + reAtextAscii + "|" + reNonAscii + ")+"
const reEmailLocal = reAtext + "(\\." + reAtext + ")*"

// vv TODO: needs work, amongst others: UTF8 support?, IPv4 + IPv6, comments
// also: how is this different from domain in general?
const reEmailDomain = "[A-Za-z0-9]{1,63}(\\.[A-Za-z0-9]{1,63})+"
const reEmail = reEmailLocal + "@" + reEmailDomain

// TODO: this needs serious work! re-use aspects from the domain recognition
func (p *lumiRegex) GetEmail() (string, error) {
	// overall:     https://en.wikipedia.org/wiki/Email_address
	// utf8 email:  https://datatracker.ietf.org/doc/html/rfc6531
	// utf8 coding: https://en.wikipedia.org/wiki/UTF-8
	// atext:       https://datatracker.ietf.org/doc/html/rfc5322#section-3.2.3
	//
	// local-part@domain
	// - unquoted:
	//     [a-z0-9!#$%&'*+-/=?^_`{|}~] and '.' (not first, not last, not in sequence)
	//     any unicode above ascii, encoded as UTF8
	// - max: 64 chars
	return reEmail, nil
}

func (p *lumiRegex) GetMac() (string, error) {
	return "[a-fA-F0-9]{2}(:[a-fA-F0-9]{2}){5}", nil
}

func (p *lumiRegex) GetUuid() (string, error) {
	return "[0-9a-fA-F]{8}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{4}\\b-[0-9a-fA-F]{12}", nil
}

func (p *lumiRegex) GetEmoji() (string, error) {
	// weather:  02600 ☀  - 027BF ➿
	// emoji:    1F300 🌀 - 1F6FC 🛼
	// extras:   1F900 🤀  - 1F9FF 🧿
	// more:     1FA70 🩰 - 1FAF6 heart hands
	return "[☀-➿🌀-🛼🤀-🧿🩰-🫶]", nil
}

func (p *lumiRegex) GetSemver() (string, error) {
	return "(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?", nil
}

func (p *lumiRegex) GetCreditCard() (string, error) {
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
