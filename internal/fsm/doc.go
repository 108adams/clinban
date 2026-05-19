// Package fsm defines the workflow state machine for Clinban tickets.
//
// The state machine is enforced by the CLI move command for human-driven
// transitions. It is intentionally independent from storage and parsing so that
// the allowed transition table can be tested and reasoned about without a
// filesystem.
package fsm
