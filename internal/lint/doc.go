// Package lint validates parsed Clinban tickets against the public ticket
// schema.
//
// Lint operates on an already parsed ticket. Syntax and frontmatter decoding
// errors belong to package ticket; lint handles semantic schema checks such as
// required fields, legal status and type values, filename-to-ID consistency,
// timestamp presence, tag contents, and repository-wide ID uniqueness.
//
// The package has no filesystem dependency. Callers provide the filename and
// the set of known repository IDs as context for rules that need repository
// information.
package lint
