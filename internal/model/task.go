package model

import "fmt"

type Task struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Deadline string `json:"deadline"`
}

func ValidateTask(t Task) error {
	if t.Title == "" {
		return fmt.Errorf("Title is required")
	}
	if t.Status == "" {
		return fmt.Errorf("Status is required")
	}
	if t.Deadline == "" {
		return fmt.Errorf("Deadline is required")
	}
	return nil
}
