package service

import (
	"context"
	"task-api/internal/model"
	"task-api/internal/repository"
)

type ProjectService struct {
	repo *repository.ProjectRepository
}

func NewProjectService(repo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{repo: repo}
}

func (s *ProjectService) CreateProject(ctx context.Context, p *model.Project, userID int64) error {
	if err := model.ValidateProject(*p); err != nil {
		return err
	}
	return s.repo.Create(ctx, p, userID)
}

func (s *ProjectService) GetProjectByID(ctx context.Context, id int64, userID int64) (*model.Project, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *ProjectService) GetProjects(ctx context.Context, userID int64) ([]model.Project, error) {
	return s.repo.ListByUserID(ctx, userID)
}
