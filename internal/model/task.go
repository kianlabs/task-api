package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	StatusTodo       = "todo"
	StatusInProgress = "in_progress"
	StatusDone       = "done"

	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

const dateFormat = "2006-01-02"

var (
	validStatuses = map[string]struct{}{
		StatusTodo: {}, StatusInProgress: {}, StatusDone: {},
	}
	validPriorities = map[string]struct{}{
		PriorityLow: {}, PriorityMedium: {}, PriorityHigh: {}, PriorityUrgent: {},
	}
)

type Task struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	ProjectID   *int64    `json:"project_id"`
	AssigneeID  *int64    `json:"assignee_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	Deadline    time.Time `json:"deadline"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (t *Task) Normalize() {
	t.Title = strings.TrimSpace(t.Title)
	t.Description = strings.TrimSpace(t.Description)
	t.Status = strings.TrimSpace(t.Status)
	t.Priority = strings.TrimSpace(t.Priority)
	if t.Status == "" {
		t.Status = StatusTodo
	}
	if t.Priority == "" {
		t.Priority = PriorityMedium
	}
}

func ValidateTask(t Task) error {
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("title is required")
	}
	if len(t.Title) > 200 {
		return errors.New("title must be at most 200 characters")
	}
	if len(t.Description) > 5000 {
		return errors.New("description must be at most 5000 characters")
	}
	if _, ok := validStatuses[t.Status]; !ok {
		return fmt.Errorf("status must be one of: todo, in_progress, done")
	}
	if _, ok := validPriorities[t.Priority]; !ok {
		return fmt.Errorf("priority must be one of: low, medium, high, urgent")
	}
	if t.Deadline.IsZero() {
		return errors.New("deadline is required")
	}
	return nil
}

func (t Task) MarshalJSON() ([]byte, error) {
	type Alias Task
	return json.Marshal(struct {
		Alias
		Deadline string `json:"deadline"`
	}{
		Alias:    Alias(t),
		Deadline: t.Deadline.Format(dateFormat),
	})
}

func (t *Task) UnmarshalJSON(data []byte) error {
	type Alias Task
	aux := struct {
		*Alias
		Deadline string `json:"deadline"`
	}{Alias: (*Alias)(t)}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.Deadline == "" {
		t.Deadline = time.Time{}
		return nil
	}
	parsed, err := time.Parse(dateFormat, aux.Deadline)
	if err != nil {
		return fmt.Errorf("deadline must be YYYY-MM-DD: %w", err)
	}
	t.Deadline = parsed
	return nil
}

type TaskFilter struct {
	UserID    int64
	ProjectID *int64
	Status    string
	Priority  string
	Search    string
	Sort      string
	Order     string
	Page      int
	Limit     int
}

func (f *TaskFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	switch f.Sort {
	case "deadline", "priority", "created_at", "id":
	default:
		f.Sort = "id"
	}
	if strings.EqualFold(f.Order, "desc") {
		f.Order = "desc"
	} else {
		f.Order = "asc"
	}
	f.Search = strings.TrimSpace(f.Search)
	f.Status = strings.TrimSpace(f.Status)
	f.Priority = strings.TrimSpace(f.Priority)
}

func (f TaskFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

func IsValidStatus(s string) bool {
	_, ok := validStatuses[s]
	return ok
}

func IsValidPriority(p string) bool {
	_, ok := validPriorities[p]
	return ok
}
