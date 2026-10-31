package repository

import (
	"context"
	"database/sql"
	"task-api/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	query := `INSERT INTO tasks (user_id, title, status, deadline)
	VALUES ($1, $2, $3, $4)
	RETURNING id`

	return r.db.QueryRowContext(ctx, query, task.UserID, task.Title, task.Status, task.Deadline).Scan(&task.ID)
}
