package task

import (
	"fmt"
	"strings"
)

// Status is the lifecycle state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in progress"
	StatusDone       Status = "done"
)

// Statuses lists every valid status in display order.
var Statuses = []Status{StatusTodo, StatusInProgress, StatusDone}

// ParseStatus converts a user-supplied string into a Status. It accepts
// case-insensitive input and common spellings such as "in-progress" or
// "in_progress" for StatusInProgress.
func ParseStatus(s string) (Status, error) {
	switch normalize(s) {
	case "todo":
		return StatusTodo, nil
	case "in progress", "in-progress", "in_progress", "progress", "doing":
		return StatusInProgress, nil
	case "done", "complete", "completed":
		return StatusDone, nil
	default:
		return "", fmt.Errorf("invalid status %q (want one of: %s)", s, StatusList())
	}
}

// StatusList renders the valid statuses as a comma-separated list.
func StatusList() string {
	parts := make([]string, len(Statuses))
	for i, s := range Statuses {
		parts[i] = string(s)
	}
	return strings.Join(parts, ", ")
}

func normalize(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(s), " "))
}
