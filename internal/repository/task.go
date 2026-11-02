package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"task-api/internal/model"
)

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// accessPredicate membatasi task yang boleh dilihat user: miliknya, yang
// di-assign kepadanya, atau task dari project tempat ia menjadi anggota.
const accessPredicate = `(t.user_id = $1
	OR t.assignee_id = $1
	OR (t.project_id IS NOT NULL AND t.project_id IN (
		SELECT pm.project_id FROM project_members pm WHERE pm.user_id = $1
	)))`

const taskColumns = `t.id, t.user_id, t.project_id, t.assignee_id, t.title, t.description,
	t.status, t.priority, t.deadline, t.created_at, t.updated_at`

func (r *TaskRepository) Create(ctx context.Context, task *model.Task) error {
	query := `INSERT INTO tasks (user_id, project_id, assignee_id, title, description, status, priority, deadline, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP)
	RETURNING id, created_at, updated_at`

	return r.db.QueryRowContext(ctx, query,
		task.UserID, task.ProjectID, task.AssigneeID, task.Title,
		task.Description, task.Status, task.Priority, task.Deadline,
	).Scan(&task.ID, &task.CreatedAt, &task.UpdatedAt)
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*model.Task, error) {
	query := `SELECT ` + taskColumns + ` FROM tasks t WHERE t.id = $1`

	var t model.Task
	var projectID, assigneeID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.UserID, &projectID, &assigneeID, &t.Title, &t.Description,
		&t.Status, &t.Priority, &t.Deadline, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if projectID.Valid {
		t.ProjectID = &projectID.Int64
	}
	if assigneeID.Valid {
		t.AssigneeID = &assigneeID.Int64
	}
	return &t, nil
}

func (r *TaskRepository) List(ctx context.Context, f model.TaskFilter) ([]model.Task, int, error) {
	conditions := []string{accessPredicate}
	args := []any{f.UserID}

	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if f.ProjectID != nil {
		conditions = append(conditions, "t.project_id = "+addArg(*f.ProjectID))
	}
	if f.Status != "" {
		conditions = append(conditions, "t.status = "+addArg(f.Status))
	}
	if f.Priority != "" {
		conditions = append(conditions, "t.priority = "+addArg(f.Priority))
	}
	if f.Search != "" {
		pattern := "%" + f.Search + "%"
		placeholder := addArg(pattern)
		conditions = append(conditions, "(t.title ILIKE "+placeholder+" OR t.description ILIKE "+placeholder+")")
	}

	where := strings.Join(conditions, " AND ")

	var total int
	countQuery := `SELECT COUNT(*) FROM tasks t WHERE ` + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderExpr := taskSortExpression(f.Sort)
	order := "ASC"
	if f.Order == "desc" {
		order = "DESC"
	}

	query := fmt.Sprintf(`SELECT %s FROM tasks t WHERE %s ORDER BY %s %s, t.id ASC LIMIT %s OFFSET %s`,
		taskColumns, where, orderExpr, order, addArg(f.Limit), addArg(f.Offset()))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		var projectID, assigneeID sql.NullInt64
		if err := rows.Scan(
			&t.ID, &t.UserID, &projectID, &assigneeID, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.Deadline, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if projectID.Valid {
			t.ProjectID = &projectID.Int64
		}
		if assigneeID.Valid {
			t.AssigneeID = &assigneeID.Int64
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

func taskSortExpression(sort string) string {
	switch sort {
	case "deadline":
		return "t.deadline"
	case "priority":
		return `CASE t.priority WHEN 'urgent' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 ELSE 1 END`
	case "created_at":
		return "t.created_at"
	default:
		return "t.id"
	}
}

func (r *TaskRepository) Update(ctx context.Context, t *model.Task) (bool, error) {
	query := `UPDATE tasks
	SET title = $1, description = $2, status = $3, priority = $4, deadline = $5,
	    project_id = $6, assignee_id = $7, updated_at = CURRENT_TIMESTAMP
	WHERE id = $8`

	res, err := r.db.ExecContext(ctx, query,
		t.Title, t.Description, t.Status, t.Priority, t.Deadline,
		t.ProjectID, t.AssigneeID, t.ID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if affected == 0 {
		return false, nil
	}
	if err := r.db.QueryRowContext(ctx, `SELECT created_at, updated_at FROM tasks WHERE id = $1`, t.ID).
		Scan(&t.CreatedAt, &t.UpdatedAt); err != nil {
		return false, err
	}
	return true, nil
}

func (r *TaskRepository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *TaskRepository) Stats(ctx context.Context, userID int64) (model.DashboardStats, error) {
	query := `SELECT
		COUNT(*),
		COUNT(*) FILTER (WHERE t.status = 'todo'),
		COUNT(*) FILTER (WHERE t.status = 'in_progress'),
		COUNT(*) FILTER (WHERE t.status = 'done'),
		COUNT(*) FILTER (WHERE t.priority = 'low'),
		COUNT(*) FILTER (WHERE t.priority = 'medium'),
		COUNT(*) FILTER (WHERE t.priority = 'high'),
		COUNT(*) FILTER (WHERE t.priority = 'urgent'),
		COUNT(*) FILTER (WHERE t.deadline < CURRENT_DATE AND t.status <> 'done'),
		COUNT(*) FILTER (WHERE t.deadline = CURRENT_DATE AND t.status <> 'done')
	FROM tasks t WHERE ` + accessPredicate

	var stats model.DashboardStats
	var todo, inProgress, done, low, medium, high, urgent int
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalTasks,
		&todo, &inProgress, &done,
		&low, &medium, &high, &urgent,
		&stats.Overdue, &stats.DueToday,
	)
	if err != nil {
		return model.DashboardStats{}, err
	}

	stats.ByStatus = map[string]int{
		model.StatusTodo: todo, model.StatusInProgress: inProgress, model.StatusDone: done,
	}
	stats.ByPriority = map[string]int{
		model.PriorityLow: low, model.PriorityMedium: medium,
		model.PriorityHigh: high, model.PriorityUrgent: urgent,
	}

	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM project_members WHERE user_id = $1`, userID,
	).Scan(&stats.Projects); err != nil {
		return model.DashboardStats{}, err
	}

	return stats, nil
}

func (r *TaskRepository) DueReminders(ctx context.Context) ([]model.ReminderCount, error) {
	query := `SELECT COALESCE(t.assignee_id, t.user_id) AS responsible_id,
		COUNT(*) FILTER (WHERE t.deadline < CURRENT_DATE),
		COUNT(*) FILTER (WHERE t.deadline = CURRENT_DATE)
	FROM tasks t
	WHERE t.status <> 'done' AND t.deadline <= CURRENT_DATE
	GROUP BY responsible_id
	ORDER BY responsible_id`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reminders := []model.ReminderCount{}
	for rows.Next() {
		var r model.ReminderCount
		if err := rows.Scan(&r.UserID, &r.Overdue, &r.DueToday); err != nil {
			return nil, err
		}
		reminders = append(reminders, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reminders, nil
}
