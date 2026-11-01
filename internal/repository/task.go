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

func (r *TaskRepository) GetByUserID(ctx context.Context, userID int64) ([]model.Task, error) {
	query := `SELECT id, user_id, title, status, deadline FROM tasks WHERE user_id = $1 ORDER BY id`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.Deadline); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int, userID int64) (*model.Task, error) {
	var t model.Task
	query := `SELECT id, user_id, title, status, deadline FROM tasks WHERE id = $1 AND user_id = $2`
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&t.ID, &t.UserID, &t.Title, &t.Status, &t.Deadline)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TaskRepository) Update(ctx context.Context, id int, userID int64, t *model.Task) (bool, error) {
	query := `UPDATE tasks SET title = $1, status = $2, deadline = $3 WHERE id = $4 AND user_id = $5`
	res, err := r.db.ExecContext(ctx, query, t.Title, t.Status, t.Deadline, id, userID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id int, userID int64) (bool, error) {
	query := `DELETE FROM tasks WHERE id = $1 AND user_id = $2`
	res, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return false, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return affected > 0, nil
}
