package template

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
	"time"
)

//go:embed new.md
var newMD string

// templateData holds the values substituted into the ticket template.
type templateData struct {
	ID  int
	Now time.Time
}

// New returns the embedded new-ticket template rendered with id and now
// pre-filled. The title and type fields are left as empty strings for the
// user to complete in their editor.
func New(id int, now time.Time) ([]byte, error) {
	tmpl, err := template.New("ticket").Parse(newMD)
	if err != nil {
		return nil, fmt.Errorf("template: parse: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, templateData{ID: id, Now: now}); err != nil {
		return nil, fmt.Errorf("template: execute: %w", err)
	}

	return buf.Bytes(), nil
}
