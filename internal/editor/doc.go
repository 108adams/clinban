// Package editor opens ticket files in the user's configured editor.
//
// The package is deliberately small: it resolves the EDITOR environment
// variable, falls back to vi when EDITOR is unset, and runs the editor as a
// child process with the caller's standard input, output, and error streams.
// Callers remain responsible for reading, parsing, validating, and writing the
// file after the editor exits.
package editor
