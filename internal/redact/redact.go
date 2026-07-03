// Package redact scrubs secret values out of strings destined for logs, error
// messages, or frontend event payloads.
//
// The primary, testable guarantee is VALUE-based: given the actual secret
// strings (a host password / key passphrase) that are in scope at an error
// site, Redact replaces every occurrence with a fixed placeholder. This is a
// real boundary you can assert with a sentinel — unlike pattern guessing, it
// does not depend on the secret appearing in a particular textual shape.
//
// As defence in depth it also strips any embedded PEM private-key block, since
// such material must never reach a log or the UI regardless of source.
package redact

import (
	"regexp"
	"strings"
)

// Placeholder is what every redacted secret is replaced with.
const Placeholder = "***"

// pemBlock matches an inline PEM private-key block (any key type).
var pemBlock = regexp.MustCompile(`(?s)-----BEGIN [^-]*PRIVATE KEY-----.*?-----END [^-]*PRIVATE KEY-----`)

// String replaces every occurrence of each non-empty secret in s with the
// placeholder, then strips any embedded PEM private-key block. Secrets shorter
// than 4 characters are ignored to avoid mangling unrelated text (a real
// password/passphrase is longer, and scrubbing e.g. "a" would corrupt output).
func String(s string, secrets ...string) string {
	for _, sec := range secrets {
		if len(sec) < 4 {
			continue
		}
		s = strings.ReplaceAll(s, sec, Placeholder)
	}
	s = pemBlock.ReplaceAllString(s, "-----REDACTED PRIVATE KEY-----")
	return s
}

// Error is a convenience wrapper: it redacts an error's message and returns the
// scrubbed text, or "" when err is nil.
func Error(err error, secrets ...string) string {
	if err == nil {
		return ""
	}
	return String(err.Error(), secrets...)
}
