package service

import (
	"context"
	"errors"
	"strings"
	"task_manager/internal/model"

	"task_manager/internal/repository"
)

var (
	ErrTeamNameRequired  = errors.New("team name is required")
	ErrForbidden         = errors.New("forbidden")
	ErrInvalidRole       = errors.New("invalid role")
	ErrCannotAssignOwner = errors.New("owner role cannot be assigned")
	ErrAlreadyMember     = errors.New("user is already a team member")
	ErrCannotChangeOwner = errors.New("cannot change owner role")
)

type TeamService struct {
	teamRepo *repository.TeamRepository
}

func NewTeamService(teamRepo *repository.TeamRepository) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
	}
}

func (s *TeamService) Create(ctx context.Context, name string, createdBy int64) (int64, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return 0, ErrTeamNameRequired
	}

	return s.teamRepo.Create(ctx, name, createdBy)
}

func (s *TeamService) GetByUserID(ctx context.Context, userID int64) ([]model.Team, error) {
	return s.teamRepo.GetByUserID(ctx, userID)
}

func (s *TeamService) Invite(ctx context.Context, teamID int64, currentUserID int64, userID int64, role string) error {
	currentRole, err := s.teamRepo.GetMemberRole(ctx, teamID, currentUserID)
	if err != nil {
		return err
	}

	if currentRole != "owner" && currentRole != "admin" {
		return ErrForbidden
	}

	if role == "owner" {
		return ErrCannotAssignOwner
	}

	if role != "admin" && role != "member" {
		return ErrInvalidRole
	}

	existingRole, err := s.teamRepo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		return err
	}

	if existingRole != "" {
		return ErrAlreadyMember
	}

	return s.teamRepo.AddMember(ctx, teamID, userID, role)
}

func (s *TeamService) UpdateMemberRole(ctx context.Context, teamID int64, currentUserID int64, targetUserID int64, newRole string) error {
	currentRole, err := s.teamRepo.GetMemberRole(ctx, teamID, currentUserID)
	if err != nil {
		return err
	}

	if currentRole != "owner" {
		return ErrForbidden
	}

	if newRole != "admin" && newRole != "member" {
		return ErrInvalidRole
	}

	targetRole, err := s.teamRepo.GetMemberRole(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}

	if targetRole == "" {
		return repository.ErrMemberNotFound
	}

	if targetRole == "owner" {
		return ErrCannotChangeOwner
	}

	return s.teamRepo.UpdateMemberRole(ctx, teamID, targetUserID, newRole)
}
