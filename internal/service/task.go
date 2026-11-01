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

func (s *TaskService) GetTasks(ctx context.Context, userID int64) ([]model.Task, error) {
	return s.repo.GetByUserID(ctx, userID)
}

func (s *TaskService) GetTaskByID(ctx context.Context, id int, userID int64) (*model.Task, error) {
	return s.repo.GetByID(ctx, id, userID)
}

func (s *TaskService) UpdateTask(ctx context.Context, id int, userID int64, t *model.Task) (bool, error) {
	if err := model.ValidateTask(*t); err != nil {
		return false, err
	}
	return s.repo.Update(ctx, id, userID, t)
}

func (s *TaskService) DeleteTask(ctx context.Context, id int, userID int64) (bool, error) {
	return s.repo.Delete(ctx, id, userID)
}
