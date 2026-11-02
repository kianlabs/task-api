package service

import (
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"task-api/internal/model"
	"task-api/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetProfile(ctx context.Context, id int64) (*model.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("user not found")
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) UpdateEmail(ctx context.Context, id int64, email string) (*model.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, invalid("email is not valid")
	}

	if err := s.repo.UpdateEmail(ctx, id, email); errors.Is(err, repository.ErrDuplicate) {
		return nil, conflict("email is already registered")
	} else if err != nil {
		return nil, err
	}

	return s.GetProfile(ctx, id)
}

func (s *UserService) ChangePassword(ctx context.Context, id int64, current, next string) error {
	if len(next) < 6 {
		return invalid("new password must be at least 6 characters")
	}

	u, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return notFound("user not found")
	}
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(current)); err != nil {
		return forbidden("current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(next), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordHash(ctx, id, string(hash))
}
