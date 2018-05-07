package minit

import "errors"

var (
	// ErrURLSchemeNotSupported url scheme not supported
	ErrURLSchemeNotSupported = errors.New("URL scheme not supported")
)

// Command command
type Command struct {
	Cmd []string // must not be empty
	Env []string
	Pty bool
}
