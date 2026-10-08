package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"task-tracker/internal/task"
)

// fixedTime is a stable clock so timestamp assertions are deterministic.
var fixedTime = time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)

// newTestStore returns a loaded store backed by a temp file, with a frozen clock.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	s := NewStore(filepath.Join(t.TempDir(), "tasks.json"))
	if err := s.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	s.now = func() time.Time { return fixedTime }
	return s
}

func TestLoadMissingFileStartsEmpty(t *testing.T) {
	s := newTestStore(t)

	if got := s.List(nil); len(got) != 0 {
		t.Fatalf("List on fresh store = %d tasks, want 0", len(got))
	}
	if _, err := os.Stat(s.path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Load created the file; stat err = %v", err)
	}
}

func TestAddAssignsIDsAndDefaultsStatus(t *testing.T) {
	s := newTestStore(t)

	first, err := s.Add("first task", "")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if first.ID != 1 {
		t.Errorf("first id = %d, want 1", first.ID)
	}
	if first.Status != task.StatusTodo {
		t.Errorf("default status = %q, want %q", first.Status, task.StatusTodo)
	}
	if !first.CreatedAt.Equal(fixedTime) || !first.UpdatedAt.Equal(fixedTime) {
		t.Errorf("timestamps = %v/%v, want %v", first.CreatedAt, first.UpdatedAt, fixedTime)
	}

	second, err := s.Add("second task", task.StatusInProgress)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if second.ID != 2 {
		t.Errorf("second id = %d, want 2", second.ID)
	}
	if second.Status != task.StatusInProgress {
		t.Errorf("status = %q, want %q", second.Status, task.StatusInProgress)
	}
}

func TestAddRejectsBadInput(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Add("   ", task.StatusTodo); err == nil {
		t.Error("empty description accepted, want error")
	}
	if _, err := s.Add("valid", task.Status("nonsense")); err == nil {
		t.Error("invalid status accepted, want error")
	}
	if got := s.List(nil); len(got) != 0 {
		t.Errorf("rejected adds still persisted %d tasks", len(got))
	}
}

func TestIDsAreNotReusedAfterDelete(t *testing.T) {
	s := newTestStore(t)

	if _, err := s.Add("a", task.StatusTodo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("b", task.StatusTodo); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Delete(1); err != nil {
		t.Fatal(err)
	}

	third, err := s.Add("c", task.StatusTodo)
	if err != nil {
		t.Fatal(err)
	}
	if third.ID != 3 {
		t.Errorf("id after delete = %d, want 3", third.ID)
	}
}

func TestGet(t *testing.T) {
	s := newTestStore(t)
	added, err := s.Add("find me", task.StatusDone)
	if err != nil {
		t.Fatal(err)
	}

	got, err := s.Get(added.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Description != "find me" {
		t.Errorf("description = %q, want %q", got.Description, "find me")
	}

	if _, err := s.Get(999); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Get(999) err = %v, want ErrNotFound", err)
	}
}

func TestListFiltersAndSortsByID(t *testing.T) {
	s := newTestStore(t)
	mustAdd(t, s, "one", task.StatusTodo)
	mustAdd(t, s, "two", task.StatusDone)
	mustAdd(t, s, "three", task.StatusTodo)

	all := s.List(nil)
	if len(all) != 3 {
		t.Fatalf("List(nil) = %d tasks, want 3", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID > all[i].ID {
			t.Errorf("List not sorted by id: %d before %d", all[i-1].ID, all[i].ID)
		}
	}

	todo := task.StatusTodo
	filtered := s.List(&todo)
	if len(filtered) != 2 {
		t.Fatalf("List(todo) = %d tasks, want 2", len(filtered))
	}
	for _, tk := range filtered {
		if tk.Status != task.StatusTodo {
			t.Errorf("filter leaked status %q", tk.Status)
		}
	}

	done := task.StatusDone
	if got := s.List(&done); len(got) != 1 || got[0].Description != "two" {
		t.Errorf("List(done) = %+v, want just \"two\"", got)
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	added := mustAdd(t, s, "draft", task.StatusTodo)

	later := fixedTime.Add(2 * time.Hour)
	s.now = func() time.Time { return later }

	desc := "final"
	status := task.StatusDone
	updated, err := s.Update(added.ID, &desc, &status)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Description != "final" || updated.Status != task.StatusDone {
		t.Errorf("updated = %+v, want description \"final\" and status done", updated)
	}
	if !updated.CreatedAt.Equal(fixedTime) {
		t.Errorf("CreatedAt changed to %v, want %v", updated.CreatedAt, fixedTime)
	}
	if !updated.UpdatedAt.Equal(later) {
		t.Errorf("UpdatedAt = %v, want %v", updated.UpdatedAt, later)
	}

	// A nil description must leave the existing text alone.
	progress := task.StatusInProgress
	partial, err := s.Update(added.ID, nil, &progress)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if partial.Description != "final" {
		t.Errorf("description = %q, want it preserved as \"final\"", partial.Description)
	}
	if partial.Status != task.StatusInProgress {
		t.Errorf("status = %q, want %q", partial.Status, task.StatusInProgress)
	}
}

func TestUpdateRejectsBadInput(t *testing.T) {
	s := newTestStore(t)
	added := mustAdd(t, s, "draft", task.StatusTodo)

	if _, err := s.Update(added.ID, nil, nil); err == nil {
		t.Error("Update with no fields accepted, want error")
	}
	blank := "  "
	if _, err := s.Update(added.ID, &blank, nil); err == nil {
		t.Error("Update with blank description accepted, want error")
	}
	bad := task.Status("nope")
	if _, err := s.Update(added.ID, nil, &bad); err == nil {
		t.Error("Update with invalid status accepted, want error")
	}
	if _, err := s.Update(404, nil, nil); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("Update(404) err = %v, want ErrNotFound", err)
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	added := mustAdd(t, s, "remove me", task.StatusTodo)

	deleted, err := s.Delete(added.ID)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted.ID != added.ID {
		t.Errorf("deleted id = %d, want %d", deleted.ID, added.ID)
	}
	if got := s.List(nil); len(got) != 0 {
		t.Errorf("List after delete = %d tasks, want 0", len(got))
	}
	if _, err := s.Delete(added.ID); !errors.Is(err, task.ErrNotFound) {
		t.Errorf("second Delete err = %v, want ErrNotFound", err)
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")

	s := NewStore(path)
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	s.now = func() time.Time { return fixedTime }
	added := mustAdd(t, s, "persist me", task.StatusDone)

	reloaded := NewStore(path)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := reloaded.Get(added.ID)
	if err != nil {
		t.Fatalf("Get after reload: %v", err)
	}
	if got.Description != "persist me" || got.Status != task.StatusDone {
		t.Errorf("reloaded = %+v, want description \"persist me\" and status done", got)
	}
	if !got.CreatedAt.Equal(fixedTime) || !got.UpdatedAt.Equal(fixedTime) {
		t.Errorf("timestamps did not survive round trip: %+v", got)
	}
}

func TestFileFormatIsTasksArray(t *testing.T) {
	s := newTestStore(t)
	mustAdd(t, s, "shape check", task.StatusTodo)

	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("file is not valid JSON object: %v", err)
	}
	if _, ok := raw["tasks"]; !ok {
		t.Fatalf("file has no \"tasks\" key, got keys %v", raw)
	}

	var arr []task.Task
	if err := json.Unmarshal(raw["tasks"], &arr); err != nil {
		t.Fatalf("\"tasks\" is not an array: %v", err)
	}
	if len(arr) != 1 || arr[0].Description != "shape check" {
		t.Errorf("tasks array = %+v, want one task \"shape check\"", arr)
	}
}

func TestSaveWithNoTasksWritesEmptyArray(t *testing.T) {
	s := newTestStore(t)
	if err := s.save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["tasks"]) != "[]" {
		t.Errorf("empty store serialized as %s, want []", raw["tasks"])
	}
}

func TestLoadRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(path)
	if err := s.Load(); err == nil {
		t.Error("Load accepted corrupt JSON, want error")
	}
}

func TestEmptyFileLoadsAsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.json")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewStore(path)
	if err := s.Load(); err != nil {
		t.Fatalf("Load of empty file: %v", err)
	}
	if got := s.List(nil); len(got) != 0 {
		t.Errorf("List = %d tasks, want 0", len(got))
	}
}

func mustAdd(t *testing.T, s *Store, desc string, status task.Status) task.Task {
	t.Helper()
	tk, err := s.Add(desc, status)
	if err != nil {
		t.Fatalf("Add(%q): %v", desc, err)
	}
	return tk
}
