package service

import (
	"context"
	"errors"
	"strings"
	"task_manager/internal/model"
	"task_manager/internal/repository"
)

var ErrCommentContentRequired = errors.New("comment content is required")

type CommentService struct {
	commentRepo *repository.CommentRepository
	taskRepo    *repository.TaskRepository
	teamRepo    *repository.TeamRepository
}

func NewCommentService(
	commentRepo *repository.CommentRepository,
	taskRepo *repository.TaskRepository,
	teamRepo *repository.TeamRepository,
) *CommentService {
	return &CommentService{
		commentRepo: commentRepo,
		taskRepo:    taskRepo,
		teamRepo:    teamRepo,
	}
}

func (s *CommentService) Create(ctx context.Context, taskID int64, userID int64, content string) (int64, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, ErrCommentContentRequired
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return 0, err
	}
	if task == nil {
		return 0, ErrTaskNotFound
	}

	role, err := s.teamRepo.GetMemberRole(ctx, task.TeamID, userID)
	if err != nil {
		return 0, err
	}
	if role == "" {
		return 0, ErrTaskForbidden
	}

	return s.commentRepo.Create(ctx, taskID, userID, content)
}

func (s *CommentService) GetAll(ctx context.Context, taskID int64, userID int64) ([]model.TaskComment, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, ErrTaskNotFound
	}

	role, err := s.teamRepo.GetMemberRole(ctx, task.TeamID, userID)
	if err != nil {
		return nil, err
	}
	if role == "" {
		return nil, ErrTaskForbidden
	}
	return s.commentRepo.GetByTaskID(ctx, taskID)
}
