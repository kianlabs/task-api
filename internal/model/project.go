package model

import (
	"errors"
	"strings"
	"time"
)

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type Project struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	OwnerID     int64     `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProjectMember struct {
	ProjectID int64     `json:"project_id"`
	UserID    int64     `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectFilter struct {
	UserID int64
	Page   int
	Limit  int
}

func (f *ProjectFilter) Normalize() {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit <= 0 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
}

func (f ProjectFilter) Offset() int {
	return (f.Page - 1) * f.Limit
}

func (p *Project) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
}

func ValidateProject(p Project) error {
	if p.Name == "" {
		return errors.New("project name is required")
	}
	if len(p.Name) > 120 {
		return errors.New("project name must be at most 120 characters")
	}
	if len(p.Description) > 2000 {
		return errors.New("description must be at most 2000 characters")
	}
	return nil
}

func IsValidRole(role string) bool {
	switch role {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	}
	return false
}
