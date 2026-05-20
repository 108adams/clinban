package main

import "fmt"

// ExitError carries a process exit code alongside a cause. Commands return it
// instead of calling os.Exit directly so that Execute can map it to the right
// exit code and deferred cleanup still runs.
type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string { return fmt.Sprintf("%v", e.Err) }
func (e ExitError) Unwrap() error { return e.Err }
