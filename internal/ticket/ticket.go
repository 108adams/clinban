package ticket

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ErrMissingFrontmatter is returned by Parse when content does not start with a
// complete --- fenced frontmatter block.
var ErrMissingFrontmatter = errors.New("ticket: parse: missing frontmatter")

// Ticket is the in-memory representation of a Clinban ticket file.
//
// The exported fields map directly to the public YAML frontmatter schema except
// for Body, which contains the Markdown content after the closing frontmatter
// fence.
type Ticket struct {
	// ID is the zero-padded four-digit ticket identifier, for example "0042".
	// It is derived from the filename by the store layer and is never stored in YAML frontmatter.
	ID string
	// Status is the current workflow state.
	Status Status `yaml:"status"`
	// Type is the ticket's controlled category.
	Type Type `yaml:"type"`
	// Title is the short human-readable ticket summary.
	Title string `yaml:"title"`
	// Tags are optional free-form labels.
	Tags []string `yaml:"tags"`
	// Created is the timestamp assigned when Clinban created or registered the ticket.
	Created time.Time `yaml:"created"`
	// Updated is refreshed by Clinban writes that modify ticket content or state.
	Updated time.Time `yaml:"updated"`
	// Body is the Markdown content after the YAML frontmatter.
	Body string
}

// frontmatter is the YAML-encodable shape of a ticket's header fields.
// Tags uses the flow style so that an empty slice serialises as `tags: []`
// rather than being omitted or rendered as a multi-line block.
// Field order determines the YAML serialisation order: title is first to make
// the human-facing summary immediately visible at the top of the file.
type frontmatter struct {
	Title   string    `yaml:"title"`
	Status  Status    `yaml:"status"`
	Type    Type      `yaml:"type"`
	Tags    []string  `yaml:"tags,flow"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
}

const fence = "---"

// Parse decodes a Markdown ticket file into a Ticket.
//
// The expected format is:
//
//	---\n
//	<yaml fields>\n
//	---\n
//	<optional body>
//
// Parse returns ErrMissingFrontmatter when the opening or closing fence is
// absent. It returns a wrapped YAML error when the frontmatter block is
// syntactically malformed.
func Parse(content []byte) (*Ticket, error) {
	// Normalise CRLF to LF so editors on Windows or with CRLF mode don't
	// produce a silent parse failure.
	s := strings.ReplaceAll(string(content), "\r\n", "\n")

	// The document must start with the opening fence line.
	const openFence = fence + "\n"
	if !strings.HasPrefix(s, openFence) {
		return nil, ErrMissingFrontmatter
	}

	// Strip the opening fence and find the closing fence.
	// Accept both "\n---\n" (normal) and "\n---" at end of file (editors that
	// strip the trailing newline on save).
	rest := s[len(openFence):]
	const closeFenceNL = "\n" + fence + "\n"
	const closeFenceEOF = "\n" + fence
	idx := strings.Index(rest, closeFenceNL)
	var body string
	if idx != -1 {
		body = rest[idx+len(closeFenceNL):]
	} else if strings.HasSuffix(rest, closeFenceEOF) {
		idx = len(rest) - len(closeFenceEOF)
		body = ""
	} else {
		return nil, ErrMissingFrontmatter
	}

	yamlBlock := rest[:idx]

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
		// ID is intentionally not set here — it is derived from the filename by
		// the store layer (store.ReadTicket). Parse always returns t.ID == "".
		Status:  fm.Status,
		Type:    fm.Type,
		Title:   fm.Title,
		Tags:    tags,
		Created: fm.Created,
		Updated: fm.Updated,
		Body:    body,
	}, nil
}

// Marshal encodes t as a Markdown ticket file.
//
// The output contains --- fenced YAML frontmatter followed by t.Body. Tags are
// always emitted, and an empty tag list is serialized as tags: [].
func Marshal(t *Ticket) ([]byte, error) {
	tags := t.Tags
	if tags == nil {
		tags = []string{}
	}

	fm := frontmatter{
		Title: t.Title,
		// ID is not stored in frontmatter — it is derived from the filename.
		Status:  t.Status,
		Type:    t.Type,
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
