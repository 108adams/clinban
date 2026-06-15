// Package board provides the canonical board display ordering for Clinban tickets.
//
// It is the single source of truth for the status rank used by both the
// "clinban list" command and the TUI board view.  Callers sort their own
// slices via sort.SliceStable and board.Less.
package board
