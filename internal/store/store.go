// Package store persists tasks in a JSON file and exposes CRUD operations
// over them.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"task-tracker/internal/task"
)

const (
	indexNotFound = -1
	// 644 means read + write for file owner, read for all other users
	writeFilePermissions = 0o644
)

// Store keeps tasks in memory and persists them to a JSON file whose only
// top-level key is a "tasks" array.
type Store struct {
	path  string
	tasks []task.Task
	now   func() time.Time
}

// fileData is the on-disk representation of the store.
type fileData struct {
	Tasks []task.Task `json:"tasks"`
}

// NewStore returns a store bound to path. Call Load before using it.
func NewStore(path string) *Store {
	return &Store{path: path, now: time.Now}
}

// Load reads the backing file into memory. A missing file is not an error: it
// yields an empty task list so the first write creates the file.
func (s *Store) Load() error {
	s.tasks = []task.Task{}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("read %s: %w", s.path, err)
	}

	if len(data) == 0 {
		return nil
	}

	var fd fileData
	if err := json.Unmarshal(data, &fd); err != nil {
		return fmt.Errorf("parse %s: %w", s.path, err)
	}

	if fd.Tasks != nil {
		s.tasks = fd.Tasks
	}

	return nil
}

// Save writes the in-memory tasks to disk atomically via a temp file rename.
func (s *Store) save() error {
	tasks := s.tasks
	if tasks == nil {
		tasks = []task.Task{}
	}

	data, err := json.MarshalIndent(fileData{Tasks: tasks}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tasks: %w", err)
	}

	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, writeFilePermissions); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}

	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("replace %s: %w", s.path, err)
	}

	return nil
}

// Add creates a task with the given description and status and persists it.
func (s *Store) Add(description string, status task.Status) (task.Task, error) {
	description = strings.TrimSpace(description)
	if description == "" {
		return task.Task{}, errors.New("task description must not be empty")
	}

	if status == "" {
		status = task.StatusTodo
	}

	status, err := task.ParseStatus(string(status))
	if err != nil {
		return task.Task{}, err
	}

	now := s.now().UTC().Truncate(time.Second)
	t := task.Task{
		ID:          s.nextID(),
		Description: description,
		Status:      status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tasks = append(s.tasks, t)
	if err := s.save(); err != nil {
		return task.Task{}, err
	}

	return t, nil
}

// List returns every task ordered by id, optionally filtered by status.
func (s *Store) List(status *task.Status) []task.Task {
	out := make([]task.Task, 0, len(s.tasks))
	for _, t := range s.tasks {
		if status != nil && t.Status != *status {
			continue
		}
		out = append(out, t)
	}

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Get returns the task with the given id.
func (s *Store) Get(id int) (task.Task, error) {
	i := s.indexOf(id)
	if i < 0 {
		return task.Task{}, fmt.Errorf("%w: %d", task.ErrNotFound, id)
	}

	return s.tasks[i], nil
}

// Update changes the description and/or status of a task. Nil arguments are
// left untouched. It returns the updated task.
func (s *Store) Update(id int, description *string, status *task.Status) (task.Task, error) {
	i := s.indexOf(id)
	if i < 0 {
		return task.Task{}, fmt.Errorf("%w: %d", task.ErrNotFound, id)
	}

	var newDesc *string
	if description != nil {
		d := strings.TrimSpace(*description)
		if d == "" {
			return task.Task{}, errors.New("task description must not be empty")
		}
		newDesc = &d
	}

	var newStatus *task.Status
	if status != nil {
		st, err := task.ParseStatus(string(*status))
		if err != nil {
			return task.Task{}, err
		}
		newStatus = &st
	}

	if newDesc == nil && newStatus == nil {
		return task.Task{}, errors.New("nothing to update: no fields provided")
	}

	if newDesc != nil {
		s.tasks[i].Description = *newDesc
	}

	if newStatus != nil {
		s.tasks[i].Status = *newStatus
	}

	s.tasks[i].UpdatedAt = s.now().UTC().Truncate(time.Second)

	if err := s.save(); err != nil {
		return task.Task{}, err
	}

	return s.tasks[i], nil
}

// Delete removes the task with the given id and persists the change.
func (s *Store) Delete(id int) (task.Task, error) {
	i := s.indexOf(id)
	if i < 0 {
		return task.Task{}, fmt.Errorf("%w: %d", task.ErrNotFound, id)
	}

	deleted := s.tasks[i]
	s.tasks = append(s.tasks[:i], s.tasks[i+1:]...)
	if err := s.save(); err != nil {
		return task.Task{}, err
	}

	return deleted, nil
}

// indexOf returns the position of id, or -1 when absent.
func (s *Store) indexOf(id int) int {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			return i
		}
	}

	return indexNotFound
}

// nextID returns one past the highest existing id, so ids stay unique even
// after deletions in the middle of the list.
func (s *Store) nextID() int {
	highest := 0
	for _, t := range s.tasks {
		if t.ID > highest {
			highest = t.ID
		}
	}

	return highest + 1
}
