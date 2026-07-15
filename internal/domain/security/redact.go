package security

import "regexp"

var (
	emailPattern = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phonePattern = regexp.MustCompile(`\+?\d[\d\-\s().]{7,}\d`)
)

// Redact masks email- and phone-like substrings in s. It is applied as a
// last line of defense at the CLI's error-output boundary (see
// internal/cli.Run) so a stray contact value can never reach logs, errors
// or doctor output even if a call site forgot to keep it out — see the
// data-ownership boundary in ARCHITECTURE.md and docs/PRIVACY.md.
func Redact(s string) string {
	s = emailPattern.ReplaceAllString(s, "[redacted-email]")
	s = phonePattern.ReplaceAllString(s, "[redacted-phone]")
	return s
}
