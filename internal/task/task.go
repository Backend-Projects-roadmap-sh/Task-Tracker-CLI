// Package task provides the domain model for the task tracker: the Task type,
// its Status values, and the parsing and validation rules around them.
package task

import (
	"time"
)

// Task is a single tracked unit of work.
type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	Status      Status    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
