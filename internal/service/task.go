package service

import (
	"context"
	"database/sql"
	"errors"

	"task-api/internal/model"
	"task-api/internal/repository"
)

type TaskService struct {
	repo        *repository.TaskRepository
	projectRepo *repository.ProjectRepository
	userRepo    *repository.UserRepository
}

func NewTaskService(repo *repository.TaskRepository, projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository) *TaskService {
	return &TaskService{repo: repo, projectRepo: projectRepo, userRepo: userRepo}
}

// CreateTask membuat task milik ownerID. Bila project diisi, owner wajib
// anggota project tersebut.
func (s *TaskService) CreateTask(ctx context.Context, ownerID int64, t *model.Task) error {
	t.UserID = ownerID
	t.Normalize()

	if err := model.ValidateTask(*t); err != nil {
		return invalid(err.Error())
	}
	if t.ProjectID != nil {
		member, err := s.projectRepo.IsMember(ctx, *t.ProjectID, ownerID)
		if err != nil {
			return err
		}
		if !member {
			return forbidden("you are not a member of this project")
		}
	}
	if err := s.validateAssignment(ctx, t.ProjectID, t.AssigneeID); err != nil {
		return err
	}
	return s.repo.Create(ctx, t)
}

func (s *TaskService) GetTask(ctx context.Context, id, userID int64) (*model.Task, error) {
	t, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("task not found")
	}
	if err != nil {
		return nil, err
	}

	allowed, err := s.canRead(ctx, t, userID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		// Sengaja 404 agar keberadaan resource tidak bocor.
		return nil, notFound("task not found")
	}
	return t, nil
}

func (s *TaskService) ListTasks(ctx context.Context, f model.TaskFilter) ([]model.Task, int, error) {
	f.Normalize()
	return s.repo.List(ctx, f)
}

// ListProjectTasks memastikan pemanggil anggota project sebelum membaca task.
func (s *TaskService) ListProjectTasks(ctx context.Context, projectID, userID int64, f model.TaskFilter) ([]model.Task, int, error) {
	member, err := s.projectRepo.IsMember(ctx, projectID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !member {
		return nil, 0, forbidden("you are not a member of this project")
	}

	f.UserID = userID
	f.ProjectID = &projectID
	f.Normalize()
	return s.repo.List(ctx, f)
}

func (s *TaskService) UpdateTask(ctx context.Context, id, userID int64, updated *model.Task) (*model.Task, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("task not found")
	}
	if err != nil {
		return nil, err
	}

	allowed, err := s.canWrite(ctx, existing, userID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, forbidden("you cannot modify this task")
	}

	updated.Normalize()
	if err := model.ValidateTask(*updated); err != nil {
		return nil, invalid(err.Error())
	}
	if err := s.validateAssignment(ctx, updated.ProjectID, updated.AssigneeID); err != nil {
		return nil, err
	}

	updated.ID = id
	updated.UserID = existing.UserID
	found, err := s.repo.Update(ctx, updated)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, notFound("task not found")
	}
	return updated, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, id, userID int64) error {
	existing, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound("task not found")
	}
	if err != nil {
		return err
	}

	allowed, err := s.canWrite(ctx, existing, userID)
	if err != nil {
		return err
	}
	if !allowed {
		return forbidden("you cannot delete this task")
	}

	found, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return notFound("task not found")
	}
	return nil
}

func (s *TaskService) Dashboard(ctx context.Context, userID int64) (model.DashboardStats, error) {
	return s.repo.Stats(ctx, userID)
}

func (s *TaskService) canRead(ctx context.Context, t *model.Task, userID int64) (bool, error) {
	if t.UserID == userID {
		return true, nil
	}
	if t.AssigneeID != nil && *t.AssigneeID == userID {
		return true, nil
	}
	if t.ProjectID != nil {
		return s.projectRepo.IsMember(ctx, *t.ProjectID, userID)
	}
	return false, nil
}

// canWrite: pemilik task, atau owner/admin project terkait.
func (s *TaskService) canWrite(ctx context.Context, t *model.Task, userID int64) (bool, error) {
	if t.UserID == userID {
		return true, nil
	}
	if t.ProjectID == nil {
		return false, nil
	}
	role, err := s.projectRepo.GetMemberRole(ctx, *t.ProjectID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return role == model.RoleOwner || role == model.RoleAdmin, nil
}

// validateAssignment memastikan project/assignee yang dipilih valid dan
// konsisten satu sama lain.
func (s *TaskService) validateAssignment(ctx context.Context, projectID, assigneeID *int64) error {
	if projectID != nil {
		if _, err := s.projectRepo.GetByID(ctx, *projectID); errors.Is(err, sql.ErrNoRows) {
			return invalid("project not found")
		} else if err != nil {
			return err
		}
	}

	if assigneeID == nil {
		return nil
	}

	exists, err := s.userRepo.Exists(ctx, *assigneeID)
	if err != nil {
		return err
	}
	if !exists {
		return invalid("assignee not found")
	}

	if projectID != nil {
		member, err := s.projectRepo.IsMember(ctx, *projectID, *assigneeID)
		if err != nil {
			return err
		}
		if !member {
			return invalid("assignee is not a member of the project")
		}
	}
	return nil
}
