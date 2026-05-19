package slug

import (
	"strings"
)

// Slugify converts title into a filesystem-safe ticket filename slug.
//
// The slug is built from up to five non-empty tokens. Each token is lowercased,
// stripped to ASCII letters and digits, and joined with hyphens. Tokens that
// become empty after stripping are skipped and do not count toward the limit.
func Slugify(title string) string {
	const maxTokens = 5

	rawTokens := strings.Fields(title)

	parts := make([]string, 0, maxTokens)
	for _, token := range rawTokens {
		if len(parts) == maxTokens {
			break
		}

		lower := strings.ToLower(token)

		// Strip every character that is not [a-z0-9] (ASCII only).
		var b strings.Builder
		for _, r := range lower {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				b.WriteRune(r)
			}
		}
		cleaned := b.String()

		// Skip tokens that are empty after stripping; they do not
		// count toward the 5-word limit.
		if cleaned == "" {
			continue
		}

		parts = append(parts, cleaned)
	}

	return strings.Join(parts, "-")
}
