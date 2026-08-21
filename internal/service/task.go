package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"task_manager/internal/model"
	"time"

	"task_manager/internal/repository"
)

var (
	ErrTaskTitleRequired  = errors.New("task title is required")
	ErrTaskStatusRequired = errors.New("task status is required")
	ErrNotTeamMember      = errors.New("user is not a team member")
	ErrAssigneeNotMember  = errors.New("assignee is not a team member")
	ErrTaskNotFound       = errors.New("task not found")
	ErrTaskForbidden      = errors.New("task update forbidden")
	ErrVersionConflict    = errors.New("task version conflict")
)

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Status      *string
	AssigneeID  *int64
	Version     int
}

type TaskService struct {
	taskRepo  *repository.TaskRepository
	teamRepo  *repository.TeamRepository
	taskCache *repository.TaskCache
}

func NewTaskService(taskRepo *repository.TaskRepository, teamRepo *repository.TeamRepository, taskCache *repository.TaskCache) *TaskService {
	return &TaskService{
		taskRepo:  taskRepo,
		teamRepo:  teamRepo,
		taskCache: taskCache,
	}
}

func (s *TaskService) Create(ctx context.Context, teamID int64, title string, description string, status string, createdBy int64, assigneeID *int64) (int64, error) {
	title = strings.TrimSpace(title)
	status = strings.TrimSpace(status)

	if title == "" {
		return 0, ErrTaskTitleRequired
	}

	if status == "" {
		return 0, ErrTaskStatusRequired
	}

	role, err := s.teamRepo.GetMemberRole(ctx, teamID, createdBy)
	if err != nil {
		return 0, err
	}

	if role == "" {
		return 0, ErrNotTeamMember
	}

	if assigneeID != nil {
		assigneeRole, err := s.teamRepo.GetMemberRole(ctx, teamID, *assigneeID)
		if err != nil {
			return 0, err
		}

		if assigneeRole == "" {
			return 0, ErrAssigneeNotMember
		}
	}

	changes := map[string]any{
		"title": map[string]any{
			"old": nil,
			"new": title,
		},
		"description": map[string]any{
			"old": nil,
			"new": description,
		},
		"status": map[string]any{
			"old": nil,
			"new": status,
		},
		"assignee_id": map[string]any{
			"old": nil,
			"new": assigneeID,
		},
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return 0, err
	}

	taskID, err := s.taskRepo.Create(ctx, teamID, title, description, status, createdBy, assigneeID, changesJSON)
	if err != nil {
		return 0, err
	}

	_ = s.taskCache.InvalidateTeam(ctx, teamID)

	return taskID, nil
}

func (s *TaskService) GetAll(ctx context.Context, userID int64, filter model.TaskFilter) ([]model.Task, error) {
	role, err := s.teamRepo.GetMemberRole(ctx, filter.TeamID, userID)
	if err != nil {
		return nil, err
	}

	if role == "" {
		return nil, ErrNotTeamMember
	}

	cached, found, err := s.taskCache.Get(ctx, filter)
	if err == nil && found {
		return cached, nil
	}

	tasks, err := s.taskRepo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	if err := s.taskCache.Set(ctx, filter, tasks); err != nil {
		// кеш не должен ломать основной запрос
	}

	return tasks, nil
}

func (s *TaskService) Update(ctx context.Context, taskID int64, userID int64, input UpdateTaskInput) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return ErrTaskNotFound
	}

	role, err := s.teamRepo.GetMemberRole(ctx, task.TeamID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return ErrTaskForbidden
	}

	isOwnerOrAdmin := role == "owner" || role == "admin"
	isCreator := task.CreatedBy == userID
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == userID

	if !isOwnerOrAdmin && !isCreator && !isAssignee {
		return ErrTaskForbidden
	}
	if isAssignee && !isOwnerOrAdmin && !isCreator {
		if input.Title != nil || input.Description != nil || input.AssigneeID != nil {
			return ErrTaskForbidden
		}
	}
	newTitle := task.Title
	newDescription := task.Description
	newStatus := task.Status
	newAssigneeID := task.AssigneeID

	if input.Title != nil {
		newTitle = strings.TrimSpace(*input.Title)
		if newTitle == "" {
			return ErrTaskTitleRequired
		}
	}

	if input.Description != nil {
		newDescription = *input.Description
	}

	if input.Status != nil {
		newStatus = strings.TrimSpace(*input.Status)
		if newStatus == "" {
			return ErrTaskStatusRequired
		}
	}

	newClosedAt := task.ClosedAt

	if newStatus != task.Status {
		if newStatus == "done" {
			now := time.Now()
			newClosedAt = &now
		} else if task.Status == "done" {
			newClosedAt = nil
		}
	}

	if input.AssigneeID != nil {
		assigneeRole, err := s.teamRepo.GetMemberRole(ctx, task.TeamID, *input.AssigneeID)
		if err != nil {
			return err
		}
		if assigneeRole == "" {
			return ErrAssigneeNotMember
		}
		newAssigneeID = input.AssigneeID
	}

	changes := make(map[string]any)

	if newTitle != task.Title {
		changes["title"] = map[string]any{
			"old": task.Title,
			"new": newTitle,
		}
	}

	if newDescription != task.Description {
		changes["description"] = map[string]any{
			"old": task.Description,
			"new": newDescription,
		}
	}

	if newStatus != task.Status {
		changes["status"] = map[string]any{
			"old": task.Status,
			"new": newStatus,
		}
	}

	if !sameInt64Ptr(newAssigneeID, task.AssigneeID) {
		changes["assignee_id"] = map[string]any{
			"old": task.AssigneeID,
			"new": newAssigneeID,
		}
	}

	if len(changes) == 0 {
		return nil
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}

	err = s.taskRepo.Update(ctx, task.ID, input.Version, newTitle, newDescription, newStatus, newAssigneeID, newClosedAt, userID, changesJSON)
	if err != nil {
		if errors.Is(err, repository.ErrVersionConflict) {
			return ErrVersionConflict
		}
		return err
	}
	_ = s.taskCache.InvalidateTeam(ctx, task.TeamID)
	return nil
}

func sameInt64Ptr(a, b *int64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (s *TaskService) GetHistory(ctx context.Context, taskID int64, userID int64) ([]model.TaskHistory, error) {
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

	return s.taskRepo.GetHistory(ctx, taskID)
}
