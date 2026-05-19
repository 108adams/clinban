// Package template renders the initial Markdown file used by interactive ticket
// creation.
//
// The package embeds the new-ticket template and substitutes the system-owned
// fields that Clinban controls: ID, created timestamp, and updated timestamp.
// User-owned fields are left blank for the editor workflow.
package template
