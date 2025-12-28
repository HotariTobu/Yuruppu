package history

// StorageReadError is returned when reading from storage fails.
type StorageReadError struct {
	Message string
}

func (e *StorageReadError) Error() string {
	return e.Message
}

// StorageWriteError is returned when writing to storage fails.
type StorageWriteError struct {
	Message string
}

func (e *StorageWriteError) Error() string {
	return e.Message
}

// StorageTimeoutError is returned when storage operation times out.
type StorageTimeoutError struct {
	Message string
}

func (e *StorageTimeoutError) Error() string {
	return e.Message
}
