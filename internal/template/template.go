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

// New renders the embedded new-ticket template for id and now.
//
// The returned bytes are a complete Markdown ticket file with system-owned
// fields pre-filled. User-owned fields such as title and type are intentionally
// blank so the interactive creation flow can detect an unchanged template.
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
