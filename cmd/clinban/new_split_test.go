package main

import (
	"testing"
)

// TestSplitRawBody covers all done-criteria cases from TASK-003.
func TestSplitRawBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		wantTitle string
		wantBody  string
	}{
		{
			name:      "empty input",
			input:     "",
			wantTitle: "",
			wantBody:  "",
		},
		{
			name:      "no hash — body only",
			input:     "just body",
			wantTitle: "",
			wantBody:  "just body",
		},
		{
			name:      "title and body separated by hash",
			input:     "title # body",
			wantTitle: "title",
			wantBody:  "body",
		},
		{
			name:      "only first hash splits — subsequent hashes stay in body",
			input:     "title # body with # hashes",
			wantTitle: "title",
			wantBody:  "body with # hashes",
		},
		{
			name:      "hash at end — empty body",
			input:     "title #",
			wantTitle: "title",
			wantBody:  "",
		},
		{
			name:      "hash at start — empty title",
			input:     "# body only",
			wantTitle: "",
			wantBody:  "body only",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotTitle, gotBody := splitRawBody(tc.input)

			if gotTitle != tc.wantTitle {
				t.Errorf("splitRawBody(%q) title = %q, want %q", tc.input, gotTitle, tc.wantTitle)
			}
			if gotBody != tc.wantBody {
				t.Errorf("splitRawBody(%q) body = %q, want %q", tc.input, gotBody, tc.wantBody)
			}
		})
	}
}
