package template

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed new.md
var newMD string

// templateData holds the values substituted into the ticket template.
type templateData struct {
	Now   time.Time
	Type  string
	Title string
}

// New renders the embedded new-ticket template for now, defaultType, and title.
//
// The returned bytes are a complete Markdown ticket file with system-owned
// fields pre-filled. The title field is pre-filled with the provided title
// value, encoded as a YAML-safe scalar so that any string — including those
// containing quotes, colons, backslashes, or newlines — round-trips through
// ticket.Parse without corruption. Passing an empty string produces title: "".
// The ticket ID is not included in the template; callers are responsible for
// setting t.ID after parsing the returned bytes.
func New(now time.Time, defaultType, title string) ([]byte, error) {
	tmpl, err := template.New("ticket").Funcs(template.FuncMap{
		"yamlstr": func(s string) (string, error) {
			b, err := yaml.Marshal(s)
			if err != nil {
				return "", err
			}
			return strings.TrimSuffix(string(b), "\n"), nil
		},
	}).Parse(newMD)
	if err != nil {
		return nil, fmt.Errorf("template: parse: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{Now: now, Type: defaultType, Title: title}); err != nil {
		return nil, fmt.Errorf("template: execute: %w", err)
	}

	return buf.Bytes(), nil
}
