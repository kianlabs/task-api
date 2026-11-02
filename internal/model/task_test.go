package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateTask(t *testing.T) {
	valid := Task{
		Title:    "Belajar Go",
		Status:   StatusTodo,
		Priority: PriorityMedium,
		Deadline: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		mutate  func(*Task)
		wantErr bool
	}{
		{"valid", func(*Task) {}, false},
		{"empty title", func(t *Task) { t.Title = "  " }, true},
		{"long title", func(t *Task) { t.Title = string(make([]byte, 201)) }, true},
		{"invalid status", func(t *Task) { t.Status = "pending" }, true},
		{"invalid priority", func(t *Task) { t.Priority = "super" }, true},
		{"empty deadline", func(t *Task) { t.Deadline = time.Time{} }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := valid
			tt.mutate(&task)
			err := ValidateTask(task)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTask() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTaskNormalize(t *testing.T) {
	task := Task{Title: "  Judul  ", Description: "  isi  "}
	task.Normalize()

	if task.Title != "Judul" {
		t.Errorf("expected trimmed title, got %q", task.Title)
	}
	if task.Status != StatusTodo {
		t.Errorf("expected default status todo, got %q", task.Status)
	}
	if task.Priority != PriorityMedium {
		t.Errorf("expected default priority medium, got %q", task.Priority)
	}
}

func TestTaskJSONDeadline(t *testing.T) {
	task := Task{ID: 1, Deadline: time.Date(2026, 12, 5, 0, 0, 0, 0, time.UTC)}

	raw, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	if decoded["deadline"] != "2026-12-05" {
		t.Errorf("expected deadline 2026-12-05, got %v", decoded["deadline"])
	}

	var back Task
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if !back.Deadline.Equal(task.Deadline) {
		t.Errorf("deadline roundtrip mismatch: %v vs %v", back.Deadline, task.Deadline)
	}
}

func TestTaskJSONInvalidDeadline(t *testing.T) {
	var task Task
	if err := json.Unmarshal([]byte(`{"deadline":"05-12-2026"}`), &task); err == nil {
		t.Error("expected error for invalid deadline format")
	}
}

func TestTaskFilterNormalize(t *testing.T) {
	f := TaskFilter{}
	f.Normalize()

	if f.Page != 1 || f.Limit != 20 {
		t.Errorf("unexpected defaults: page=%d limit=%d", f.Page, f.Limit)
	}
	if f.Sort != "id" || f.Order != "asc" {
		t.Errorf("unexpected sort defaults: %s %s", f.Sort, f.Order)
	}

	big := TaskFilter{Page: -3, Limit: 500, Sort: "evil; DROP TABLE", Order: "DESC"}
	big.Normalize()
	if big.Page != 1 || big.Limit != 100 {
		t.Errorf("expected clamping, got page=%d limit=%d", big.Page, big.Limit)
	}
	if big.Sort != "id" {
		t.Errorf("expected whitelisted sort, got %q", big.Sort)
	}
	if big.Order != "desc" {
		t.Errorf("expected desc, got %q", big.Order)
	}
}

func TestTaskFilterOffset(t *testing.T) {
	f := TaskFilter{Page: 3, Limit: 20}
	if got := f.Offset(); got != 40 {
		t.Errorf("expected offset 40, got %d", got)
	}
}

func TestEnumHelpers(t *testing.T) {
	for _, s := range []string{StatusTodo, StatusInProgress, StatusDone} {
		if !IsValidStatus(s) {
			t.Errorf("expected %q valid", s)
		}
	}
	if IsValidStatus("pending") {
		t.Error("pending should be invalid")
	}

	for _, p := range []string{PriorityLow, PriorityMedium, PriorityHigh, PriorityUrgent} {
		if !IsValidPriority(p) {
			t.Errorf("expected %q valid", p)
		}
	}
	if IsValidPriority("urgentt") {
		t.Error("urgentt should be invalid")
	}
}
