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

const projectColumns = `p.id, p.name, p.description, p.owner_id, p.created_at, p.updated_at`

func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	projectQuery := `INSERT INTO projects (name, description, owner_id, updated_at)
	VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
	RETURNING id, created_at, updated_at`
	err = tx.QueryRowContext(ctx, projectQuery, project.Name, project.Description, project.OwnerID).
		Scan(&project.ID, &project.CreatedAt, &project.UpdatedAt)
	if err != nil {
		return err
	}

	memberQuery := `INSERT INTO project_members (project_id, user_id, role) VALUES ($1, $2, 'owner')`
	if _, err := tx.ExecContext(ctx, memberQuery, project.ID, project.OwnerID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *ProjectRepository) GetByID(ctx context.Context, id int64) (*model.Project, error) {
	var p model.Project
	query := `SELECT ` + projectColumns + ` FROM projects p WHERE p.id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ProjectRepository) ListByUserID(ctx context.Context, f model.ProjectFilter) ([]model.Project, int, error) {
	var total int
	countQuery := `SELECT COUNT(*) FROM project_members pm WHERE pm.user_id = $1`
	if err := r.db.QueryRowContext(ctx, countQuery, f.UserID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `SELECT ` + projectColumns + `
	FROM projects p
	JOIN project_members pm ON p.id = pm.project_id
	WHERE pm.user_id = $1
	ORDER BY p.id
	LIMIT $2 OFFSET $3`

	rows, err := r.db.QueryContext(ctx, query, f.UserID, f.Limit, f.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	projects := []model.Project{}
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, err
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return projects, total, nil
}

func (r *ProjectRepository) Update(ctx context.Context, p *model.Project) (bool, error) {
	query := `UPDATE projects
	SET name = $1, description = $2, updated_at = CURRENT_TIMESTAMP
	WHERE id = $3`
	res, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.ID)
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
	if err := r.db.QueryRowContext(ctx, `SELECT created_at, updated_at FROM projects WHERE id = $1`, p.ID).
		Scan(&p.CreatedAt, &p.UpdatedAt); err != nil {
		return false, err
	}
	return true, nil
}

func (r *ProjectRepository) Delete(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *ProjectRepository) GetMemberRole(ctx context.Context, projectID, userID int64) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *ProjectRepository) IsMember(ctx context.Context, projectID, userID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM project_members WHERE project_id = $1 AND user_id = $2)`,
		projectID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *ProjectRepository) ListMembers(ctx context.Context, projectID int64) ([]model.ProjectMember, error) {
	query := `SELECT pm.project_id, pm.user_id, u.email, pm.role, pm.created_at
	FROM project_members pm
	JOIN users u ON u.id = pm.user_id
	WHERE pm.project_id = $1
	ORDER BY pm.created_at, pm.user_id`

	rows, err := r.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []model.ProjectMember{}
	for rows.Next() {
		var m model.ProjectMember
		if err := rows.Scan(&m.ProjectID, &m.UserID, &m.Email, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func (r *ProjectRepository) AddMember(ctx context.Context, member *model.ProjectMember) error {
	query := `INSERT INTO project_members (project_id, user_id, role)
	VALUES ($1, $2, $3)
	RETURNING created_at`
	if err := r.db.QueryRowContext(ctx, query, member.ProjectID, member.UserID, member.Role).
		Scan(&member.CreatedAt); err != nil {
		return mapPgError(err)
	}
	// Lengkapi email untuk response.
	return r.db.QueryRowContext(ctx, `SELECT email FROM users WHERE id = $1`, member.UserID).
		Scan(&member.Email)
}

func (r *ProjectRepository) UpdateMemberRole(ctx context.Context, projectID, userID int64, role string) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE project_members SET role = $1 WHERE project_id = $2 AND user_id = $3`,
		role, projectID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (r *ProjectRepository) RemoveMember(ctx context.Context, projectID, userID int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM project_members WHERE project_id = $1 AND user_id = $2`,
		projectID, userID,
	)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}
