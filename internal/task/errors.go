package task

import "errors"

// ErrNotFound is returned when no task matches the requested id.
var ErrNotFound = errors.New("task not found")
