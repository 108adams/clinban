// Package ticket defines the Clinban ticket schema and Markdown
// frontmatter parser.
//
// A ticket is a Markdown document whose first section is YAML frontmatter. The
// frontmatter contains schema fields such as id, status, type, title, tags,
// created, and updated. The remaining Markdown body is preserved as freeform
// ticket content.
//
// This package owns syntax-level parsing and marshaling. It does not check
// business rules such as valid workflow transitions or repository-wide ID
// uniqueness; those are handled by package fsm and package lint.
package ticket
