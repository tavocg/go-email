package email

import (
	"regexp"
	"strings"
)

const StrictParserError = errStr("email does not meet StrictParser requirements")

// strictParserEmailRegexp accepts a conservative ASCII email format:
//   - exactly one "@"
//   - no consecutive "."
//   - local part is 1 to 16 dot-separated segments
//   - each local segment is 1 to 30 characters
//   - local segments must start and end with an ASCII letter or digit
//   - local segment middle characters may be ASCII letters, digits, "+", "-", "_" or "."
//   - domain is 1 to 4 dot-separated labels followed by a TLD
//   - each non-TLD domain label is 1 to 32 characters
//   - TLD must be 2 to 16 ASCII letters
//   - domain labels must start and end with an ASCII letter or digit
//   - domain label middle characters may be ASCII letters, digits, or "-"
//
// It does not enforce:
//   - total email length
//   - total local-part length, because multiple 30-character local segments are allowed
//   - real RFC/DNS maximum domain length
//   - whether the domain exists
//   - whether the TLD is real
//   - whether the address is deliverable
var strictParserEmailRegexp = regexp.MustCompile(`^(?:[A-Za-z0-9](?:[A-Za-z0-9+\-_]{0,28}[A-Za-z0-9])?(?:\.(?:[A-Za-z0-9](?:[A-Za-z0-9+\-_]{0,28}[A-Za-z0-9])?)){0,15})@(?:[A-Za-z0-9](?:[A-Za-z0-9-]{0,30}[A-Za-z0-9])?\.){1,4}[A-Za-z]{2,16}$`)

// StrictParser trims surrounding whitespace and validates the address against
// the package's strict rules.
func StrictParser(email string) (valid *validAddress, err error) {
	email = strings.TrimSpace(email)
	if len(email) <= 63 && strictParserEmailRegexp.MatchString(email) {
		v := validAddress(email)
		return &v, nil
	}
	return nil, StrictParserError
}
