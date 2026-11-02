package service

import (
	"context"
	"database/sql"
	"errors"

	"task-api/internal/model"
	"task-api/internal/repository"
)

type ProjectService struct {
	repo     *repository.ProjectRepository
	userRepo *repository.UserRepository
}

func NewProjectService(repo *repository.ProjectRepository, userRepo *repository.UserRepository) *ProjectService {
	return &ProjectService{repo: repo, userRepo: userRepo}
}

func (s *ProjectService) CreateProject(ctx context.Context, ownerID int64, p *model.Project) error {
	p.Normalize()
	if err := model.ValidateProject(*p); err != nil {
		return invalid(err.Error())
	}
	p.OwnerID = ownerID
	return s.repo.Create(ctx, p)
}

func (s *ProjectService) GetProject(ctx context.Context, id, userID int64) (*model.Project, error) {
	p, err := s.loadProject(ctx, id)
	if err != nil {
		return nil, err
	}

	member, err := s.repo.IsMember(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		// 404 agar keberadaan project tidak bocor ke non-member.
		return nil, notFound("project not found")
	}
	return p, nil
}

func (s *ProjectService) ListProjects(ctx context.Context, f model.ProjectFilter) ([]model.Project, int, error) {
	f.Normalize()
	return s.repo.ListByUserID(ctx, f)
}

func (s *ProjectService) UpdateProject(ctx context.Context, id, userID int64, updated *model.Project) (*model.Project, error) {
	if _, err := s.requireRole(ctx, id, userID, model.RoleAdmin); err != nil {
		return nil, err
	}
	updated.Normalize()
	if err := model.ValidateProject(*updated); err != nil {
		return nil, invalid(err.Error())
	}

	updated.ID = id
	found, err := s.repo.Update(ctx, updated)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, notFound("project not found")
	}
	return updated, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, id, userID int64) error {
	role, err := s.requireRole(ctx, id, userID, model.RoleOwner)
	if err != nil {
		return err
	}
	if role != model.RoleOwner {
		return forbidden("only the project owner can delete this project")
	}

	found, err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return notFound("project not found")
	}
	return nil
}

func (s *ProjectService) ListMembers(ctx context.Context, projectID, userID int64) ([]model.ProjectMember, error) {
	if _, err := s.requireRole(ctx, projectID, userID, model.RoleMember); err != nil {
		return nil, err
	}
	return s.repo.ListMembers(ctx, projectID)
}

func (s *ProjectService) AddMember(ctx context.Context, projectID, actorID int64, member *model.ProjectMember) error {
	actorRole, err := s.requireRole(ctx, projectID, actorID, model.RoleAdmin)
	if err != nil {
		return err
	}

	if member.Role == "" {
		member.Role = model.RoleMember
	}
	if !model.IsValidRole(member.Role) {
		return invalid("role must be one of: owner, admin, member")
	}
	if member.Role == model.RoleOwner {
		return forbidden("owner role cannot be assigned through this endpoint")
	}
	if member.Role == model.RoleAdmin && actorRole != model.RoleOwner {
		return forbidden("only the project owner can add an admin")
	}

	exists, err := s.userRepo.Exists(ctx, member.UserID)
	if err != nil {
		return err
	}
	if !exists {
		return notFound("user not found")
	}

	member.ProjectID = projectID
	if err := s.repo.AddMember(ctx, member); errors.Is(err, repository.ErrDuplicate) {
		return conflict("user is already a member of this project")
	} else if errors.Is(err, repository.ErrReference) {
		return notFound("user not found")
	} else if err != nil {
		return err
	}
	return nil
}

func (s *ProjectService) UpdateMemberRole(ctx context.Context, projectID, actorID, targetID int64, role string) error {
	actorRole, err := s.requireRole(ctx, projectID, actorID, model.RoleOwner)
	if err != nil {
		return err
	}
	if actorRole != model.RoleOwner {
		return forbidden("only the project owner can change roles")
	}
	if !model.IsValidRole(role) || role == model.RoleOwner {
		return invalid("role must be admin or member")
	}

	targetRole, err := s.memberRole(ctx, projectID, targetID)
	if err != nil {
		return err
	}
	if targetRole == model.RoleOwner {
		return forbidden("the owner role cannot be changed")
	}

	found, err := s.repo.UpdateMemberRole(ctx, projectID, targetID, role)
	if err != nil {
		return err
	}
	if !found {
		return notFound("member not found")
	}
	return nil
}

func (s *ProjectService) RemoveMember(ctx context.Context, projectID, actorID, targetID int64) error {
	actorRole, err := s.requireRole(ctx, projectID, actorID, model.RoleAdmin)
	if err != nil {
		return err
	}

	targetRole, err := s.memberRole(ctx, projectID, targetID)
	if err != nil {
		return err
	}
	if targetRole == model.RoleOwner {
		return forbidden("the project owner cannot be removed")
	}
	if actorRole == model.RoleAdmin && targetRole == model.RoleAdmin && actorID != targetID {
		return forbidden("an admin cannot remove another admin")
	}

	found, err := s.repo.RemoveMember(ctx, projectID, targetID)
	if err != nil {
		return err
	}
	if !found {
		return notFound("member not found")
	}
	return nil
}

func (s *ProjectService) IsMember(ctx context.Context, projectID, userID int64) (bool, error) {
	return s.repo.IsMember(ctx, projectID, userID)
}

func (s *ProjectService) loadProject(ctx context.Context, id int64) (*model.Project, error) {
	p, err := s.repo.GetByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notFound("project not found")
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProjectService) requireRole(ctx context.Context, projectID, userID int64, minimum string) (string, error) {
	role, err := s.repo.GetMemberRole(ctx, projectID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		if _, projectErr := s.repo.GetByID(ctx, projectID); errors.Is(projectErr, sql.ErrNoRows) {
			return "", notFound("project not found")
		}
		return "", forbidden("you are not a member of this project")
	}
	if err != nil {
		return "", err
	}

	if roleRank(role) < roleRank(minimum) {
		return "", forbidden("insufficient project role")
	}
	return role, nil
}

func (s *ProjectService) memberRole(ctx context.Context, projectID, userID int64) (string, error) {
	role, err := s.repo.GetMemberRole(ctx, projectID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", notFound("member not found")
	}
	return role, err
}

func roleRank(role string) int {
	switch role {
	case model.RoleOwner:
		return 3
	case model.RoleAdmin:
		return 2
	case model.RoleMember:
		return 1
	default:
		return 0
	}
}
