package service

import (
	"context"
	"errors"
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

func (s *ProjectService) AddMember(ctx context.Context, projectID int64, currentUserID int64, newMember *model.ProjectMember) error {
	role, err := s.repo.GetMemberRole(ctx, projectID, currentUserID)
	if err != nil || role != "owner" {
		return errors.New("forbidden: only owner can add members")
	}
	if role != "owner" {
		return errors.New("forbidden: only owner can add members")
	}
	newMember.ProjectID = projectID
	if newMember.Role == "" {
		newMember.Role = "member" // default role jika tidak diisi
	}
	return s.repo.AddMember(ctx, newMember)
}
