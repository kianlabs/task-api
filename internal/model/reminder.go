package model

// ReminderCount merangkum task yang butuh perhatian untuk satu user.
type ReminderCount struct {
	UserID   int64 `json:"user_id"`
	Overdue  int   `json:"overdue"`
	DueToday int   `json:"due_today"`
}
