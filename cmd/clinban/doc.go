// Command clinban manages repository-local kanban tickets stored as Markdown
// files with YAML frontmatter.
//
// Clinban is intentionally filesystem-native: tickets live beside the code they
// describe, active tickets are listed from the configured ticket directory, and
// completed tickets may be moved to an archive directory. Human users interact
// through subcommands such as new, list, show, edit, move, archive, register,
// and lint. Automated users can create or modify ticket files directly and use
// lint as the schema integrity check.
//
// The command package is kept thin. Command handlers parse flags, print CLI
// output, and coordinate the internal packages that own schema parsing,
// validation, filesystem storage, state transitions, editor invocation, and
// slug generation.
package main
