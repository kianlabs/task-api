package model

type ReminderCount struct {
	UserID   int64 `json:"user_id"`
	Overdue  int   `json:"overdue"`
	DueToday int   `json:"due_today"`
}
