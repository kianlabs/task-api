package model

import (
	"errors"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (req *LoginRequest) Validate() error {
	if req.Email == "" {
		return errors.New("Email is required")
	}
	if len(req.Password) < 6 {
		return errors.New("Password must be at least 6 characters")
	}
	return nil
}

func (req *RegisterRequest) Validate() error {
	if req.Email == "" {
		return errors.New("Email is required")
	}
	if len(req.Password) < 6 {
		return errors.New("Password must be at least 6 characters")
	}
	return nil
}
