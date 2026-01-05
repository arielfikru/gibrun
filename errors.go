package gibrun

import "errors"

// Standard errors for gibrun operations.
var (
	// ErrNilValue is returned when attempting to store a nil value.
	ErrNilValue = errors.New("gibrun: cannot store nil value")

	// ErrNilPointer is returned when attempting to bind to a nil pointer.
	ErrNilPointer = errors.New("gibrun: cannot bind to nil pointer")
)
