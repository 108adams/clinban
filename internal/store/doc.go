// Package store provides filesystem-backed storage for Clinban tickets.
//
// Store knows the active ticket directory and archive directory. It scans ticket
// filenames for IDs, locates tickets by ID, reads and writes Markdown ticket
// files, lists active and archived records, and moves files between active and
// archive directories.
//
// Writes are performed by writing a temporary file in the target directory and
// renaming it into place, so readers never observe a partially written final
// file during normal operation. The package does not enforce workflow
// transitions or schema rules; callers combine it with package fsm and package
// lint for those concerns.
package store
