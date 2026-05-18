package ticket

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrMissingFrontmatter is returned by Parse when the content does not contain
// a valid --- fenced frontmatter block.
var ErrMissingFrontmatter = errors.New("ticket: parse: missing frontmatter")

// Ticket is the in-memory representation of a clinban ticket file.
// Fields map directly to the YAML frontmatter schema.
// Body holds the markdown content after the closing --- fence.
type Ticket struct {
	ID      string    `yaml:"id"`
	Status  Status    `yaml:"status"`
	Type    Type      `yaml:"type"`
	Title   string    `yaml:"title"`
	Tags    []string  `yaml:"tags"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
	Body    string    // markdown body; not part of YAML frontmatter
}

// frontmatter is the YAML-encodable shape of a ticket's header fields.
// Tags uses the flow style so that an empty slice serialises as `tags: []`
// rather than being omitted or rendered as a multi-line block.
type frontmatter struct {
	ID      string    `yaml:"id"`
	Status  Status    `yaml:"status"`
	Type    Type      `yaml:"type"`
	Title   string    `yaml:"title"`
	Tags    []string  `yaml:"tags,flow"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
}

const fence = "---"

// Parse splits content on --- fences, decodes the YAML frontmatter block,
// and captures the remainder as Body.
//
// The expected format is:
//
//	---\n
//	<yaml fields>\n
//	---\n
//	<optional body>
//
// Returns ErrMissingFrontmatter if the opening or closing fence is absent.
// Returns a wrapped yaml decode error if the frontmatter block is malformed.
func Parse(content []byte) (*Ticket, error) {
	s := string(content)

	// The document must start with the opening fence line.
	const openFence = fence + "\n"
	if !strings.HasPrefix(s, openFence) {
		return nil, ErrMissingFrontmatter
	}

	// Strip the opening fence and find the closing fence.
	rest := s[len(openFence):]
	const closeFence = "\n" + fence + "\n"
	idx := strings.Index(rest, closeFence)
	if idx == -1 {
		return nil, ErrMissingFrontmatter
	}

	yamlBlock := rest[:idx]
	body := rest[idx+len(closeFence):]

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("ticket: parse: %w", err)
	}

	// Ensure Tags is never nil so callers can always range over it safely.
	tags := fm.Tags
	if tags == nil {
		tags = []string{}
	}

	return &Ticket{
		ID:      fm.ID,
		Status:  fm.Status,
		Type:    fm.Type,
		Title:   fm.Title,
		Tags:    tags,
		Created: fm.Created,
		Updated: fm.Updated,
		Body:    body,
	}, nil
}

// Marshal serialises the ticket back to --- fenced YAML frontmatter followed
// by the Body. The output is suitable for writing directly to a .md file.
//
// Tags serialises as `tags: []` when empty (flow style), matching the schema
// contract that the field is always present.
func Marshal(t *Ticket) ([]byte, error) {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}

	fm := frontmatter{
		ID:      t.ID,
		Status:  t.Status,
		Type:    t.Type,
		Title:   t.Title,
		Tags:    tags,
		Created: t.Created,
		Updated: t.Updated,
	}

	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return nil, fmt.Errorf("ticket: marshal: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(fence + "\n")
	buf.Write(yamlBytes)
	buf.WriteString(fence + "\n")
	buf.WriteString(t.Body)

	return buf.Bytes(), nil
}
