package service

import (
	"context"
	"task_manager/internal/model"
	"task_manager/internal/repository"
)

type StatsService struct {
	statsRepo *repository.StatsRepository
	teamRepo  *repository.TeamRepository
}

func NewStatsService(statsRepo *repository.StatsRepository, teamRepo *repository.TeamRepository) *StatsService {
	return &StatsService{
		statsRepo: statsRepo,
		teamRepo:  teamRepo,
	}
}

func (s *StatsService) GetTeamStats(ctx context.Context, teamID int64, userID int64) (*model.TeamStats, error) {
	role, err := s.teamRepo.GetMemberRole(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}

	if role != "owner" && role != "admin" {
		return nil, ErrForbidden
	}
	return s.statsRepo.GetTeamStats(ctx, teamID)
}
