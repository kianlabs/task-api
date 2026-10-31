package service

import (
	"context"
	"task-api/internal/model"
	"task-api/internal/repository"
)

type TaskService struct {
	repo *repository.TaskRepository
}

func NewTaskService(repo *repository.TaskRepository) *TaskService {
	return &TaskService{repo: repo}
}

func (s *TaskService) CreateTask(ctx context.Context, task *model.Task) error {
	if err := model.ValidateTask(*task); err != nil {
		return err
	}
	return s.repo.Create(ctx, task)
}
