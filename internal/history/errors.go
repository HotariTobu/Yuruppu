package history

import "errors"

// ErrNilStorage is returned when storage is nil.
var ErrNilStorage = errors.New("storage cannot be nil")

// ValidationError is returned when input validation fails.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// ReadError is returned when reading history fails.
type ReadError struct {
	Message string
}

func (e *ReadError) Error() string {
	return e.Message
}

// WriteError is returned when writing history fails.
type WriteError struct {
	Message string
}

func (e *WriteError) Error() string {
	return e.Message
}
