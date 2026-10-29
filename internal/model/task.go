package model

import (
	"encoding/json"
	"fmt"
	"time"
)

type Task struct {
	ID       int       `json:"id"`
	Title    string    `json:"title"`
	Status   string    `json:"status"`
	Deadline time.Time `json:"deadline"`
}

func ValidateTask(t Task) error {
	if t.Title == "" {
		return fmt.Errorf("Title is required")
	}
	if t.Status == "" {
		return fmt.Errorf("Status is required")
	}
	if t.Deadline.IsZero() {
		return fmt.Errorf("Deadline is required")
	}
	return nil
}

const dateFormat = "2006-01-02"

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
