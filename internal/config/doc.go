// Package config loads Clinban project configuration.
//
// A project may define a .clinban TOML file at its root. The file can set the
// active ticket directory and archive directory; omitted fields fall back to
// sensible defaults. Paths in .clinban are resolved relative to the project
// root unless they are already absolute.
//
// The package performs only configuration loading and path resolution. It does
// not validate that directories exist and does not perform any filesystem
// operations beyond reading .clinban.
package config
