package slug_test

import (
	"testing"

	"clinban/internal/slug"
)

// Done-criteria constants from the task spec.
const (
	titleFiveWords     = "Fix login timeout on staging"
	wantFiveWords      = "fix-login-timeout-on-staging"
	titleFewerThanFive = "One two"
	wantFewerThanFive  = "one-two"
	titleSpecialChars  = "Hello, World! (urgent)"
	wantSpecialChars   = "hello-world-urgent"
	titleEmpty         = ""
	wantEmpty          = ""
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		// Done-criteria cases (P0)
		{
			name:  "five words",
			input: titleFiveWords,
			want:  wantFiveWords,
		},
		{
			name:  "fewer than five words",
			input: titleFewerThanFive,
			want:  wantFewerThanFive,
		},
		{
			name:  "special characters stripped",
			input: titleSpecialChars,
			want:  wantSpecialChars,
		},
		{
			name:  "empty string",
			input: titleEmpty,
			want:  wantEmpty,
		},
		// P1 — Business logic
		{
			name:  "truncates to first five non-empty tokens",
			input: "one two three four five six seven",
			want:  "one-two-three-four-five",
		},
		{
			name:  "single word",
			input: "Hello",
			want:  "hello",
		},
		{
			name:  "all uppercase",
			input: "FIX LOGIN TIMEOUT ON STAGING NOW",
			want:  "fix-login-timeout-on-staging",
		},
		{
			name:  "mixed case with digits",
			input: "Fix123 Login2 Timeout3",
			want:  "fix123-login2-timeout3",
		},
		// P2 — Edge cases
		{
			name:  "token all special chars skipped not counted in limit",
			input: "one !!! two ??? three four five six",
			want:  "one-two-three-four-five",
		},
		{
			name:  "token all special chars skipped counted check at limit",
			input: "one !!! two ??? three *** four",
			want:  "one-two-three-four",
		},
		{
			name:  "leading and trailing whitespace",
			input: "  Fix login  ",
			want:  "fix-login",
		},
		{
			name:  "digits only title",
			input: "1234 5678",
			want:  "1234-5678",
		},
		{
			name:  "all special characters in title",
			input: "!!! ??? ---",
			want:  "",
		},
		{
			name:  "exactly five words already",
			input: "a b c d e",
			want:  "a-b-c-d-e",
		},
		{
			name:  "tabs and multiple spaces as separators",
			input: "one\ttwo   three",
			want:  "one-two-three",
		},
		{
			name:  "unicode accented letters are stripped (only a-z0-9 allowed)",
			input: "Café résumé",
			want:  "caf-rsum",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := slug.Slugify(tc.input)

			if got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
