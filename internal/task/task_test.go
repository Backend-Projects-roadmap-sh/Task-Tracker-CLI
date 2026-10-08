package task

import (
	"errors"
	"strings"
	"testing"
)

// Validate reports whether the task holds a usable description and status.
func (t Task) Validate() error {
	if strings.TrimSpace(t.Description) == "" {
		return errors.New("task description must not be empty")
	}
	if _, err := ParseStatus(string(t.Status)); err != nil {
		return err
	}
	return nil
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		in      string
		want    Status
		wantErr bool
	}{
		{in: "todo", want: StatusTodo},
		{in: "TODO", want: StatusTodo},
		{in: "  Todo  ", want: StatusTodo},
		{in: "in progress", want: StatusInProgress},
		{in: "In Progress", want: StatusInProgress},
		{in: "in-progress", want: StatusInProgress},
		{in: "in_progress", want: StatusInProgress},
		{in: "done", want: StatusDone},
		{in: "done", want: StatusDone},
		{in: "blocked", wantErr: true},
		{in: "", wantErr: true},
	}

	for _, tt := range tests {
		got, err := ParseStatus(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseStatus(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseStatus(%q) unexpected error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseStatus(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestTaskValidate(t *testing.T) {
	valid := Task{ID: 1, Description: "write docs", Status: StatusTodo}
	if err := valid.Validate(); err != nil {
		t.Errorf("valid task rejected: %v", err)
	}

	blank := Task{ID: 1, Description: "   ", Status: StatusTodo}
	if err := blank.Validate(); err == nil {
		t.Error("blank description accepted, want error")
	}

	badStatus := Task{ID: 1, Description: "write docs", Status: Status("nope")}
	if err := badStatus.Validate(); err == nil {
		t.Error("invalid status accepted, want error")
	}
}
