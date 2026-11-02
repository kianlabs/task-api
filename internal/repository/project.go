package repository

import (
	"context"
	"database/sql"
	"task-api/internal/model"
)

type ProjectRepository struct {
	db *sql.DB
}

func NewProjectRepository(db *sql.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, project *model.Project, userID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	projectQuery := `INSERT INTO projects (name, description) VALUES ($1, $2) RETURNING id, created_at`
	err = tx.QueryRowContext(ctx, projectQuery, project.Name, project.Description).Scan(&project.ID, &project.CreatedAt)
	if err != nil {
		return err
	}
	memberQuery := `INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, 'owner')`
	_, err = tx.ExecContext(ctx, memberQuery, project.ID, userID)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *ProjectRepository) GetByID(ctx context.Context, id int64, userID int64) (*model.Project, error) {
	var p model.Project

	query := `SELECT p.id, p.name, p.description, p.created_at
	FROM projects p
	JOIN project_members pm ON p.id = pm.project_id
	WHERE p.id = $1 AND pm.user_id = $2`
	err := r.db.QueryRowContext(ctx, query, id, userID).Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) ListByUserID(ctx context.Context, userID int64) ([]model.Project, error) {
	query := `SELECT p.id, p.name, p.description, p.created_at
	FROM projects p
	JOIN project_members pm ON p.id = pm.project_id
	WHERE pm.user_id = $1
	ORDER BY p.id`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	projects := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return projects, nil

}

func (r *ProjectRepository) GetMemberRole(ctx context.Context, projectID int64, userID int64) (string, error) {
	query := `SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`
	var role string
	err := r.db.QueryRowContext(ctx, query, projectID, userID).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *ProjectRepository) AddMember(ctx context.Context, member *model.ProjectMember) error {
	query := `INSERT INTO project_members (project_id, user_id, role)
	VALUES ($1, $2, $3)
	RETURNING id, created_at`
	return r.db.QueryRowContext(ctx, query, member.ProjectID, member.UserID, member.Role).Scan(&member.ID, &member.CreatedAt)
}
