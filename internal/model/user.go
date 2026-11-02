package model

import (
	"errors"
	"net/mail"
	"strings"
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validateCredentials(email, password string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("email is not valid")
	}
	if len(password) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if len(password) > 72 {
		// Batas bcrypt: byte di atas 72 diabaikan diam-diam.
		return errors.New("password must be at most 72 characters")
	}
	return nil
}

func (req *LoginRequest) Normalize() {
	req.Email = normalizeEmail(req.Email)
}

func (req *LoginRequest) Validate() error {
	return validateCredentials(req.Email, req.Password)
}

func (req *RegisterRequest) Normalize() {
	req.Email = normalizeEmail(req.Email)
}

func (req *RegisterRequest) Validate() error {
	return validateCredentials(req.Email, req.Password)
}
