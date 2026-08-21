package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"task_manager/internal/model"

	"task_manager/internal/middleware"
	"task_manager/internal/service"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

type createTaskRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	AssigneeID  *int64 `json:"assignee_id"`
}

type createTaskResponse struct {
	ID int64 `json:"id"`
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if req.TeamID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_team_id", "invalid team id")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	taskID, err := h.taskService.Create(r.Context(), req.TeamID, req.Title, req.Description, req.Status, userID, req.AssigneeID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskTitleRequired),
			errors.Is(err, service.ErrTaskStatusRequired):

			writeJSONError(w, http.StatusBadRequest, "validation_error", err.Error())

		case errors.Is(err, service.ErrNotTeamMember):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you are not a member of this team")

		case errors.Is(err, service.ErrAssigneeNotMember):
			writeJSONError(w, http.StatusBadRequest, "invalid_assignee", "assignee must be a member of the team")

		default:
			log.Printf("create task: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createTaskResponse{ID: taskID}); err != nil {
		log.Printf("encode create task response: %v", err)
	}
}

func (h *TaskHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseInt(r.URL.Query().Get("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_team_id", "team_id is required and must be positive")
		return
	}

	limit := 20
	offset := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid_limit", "limit must be positive")
			return
		}

		limit = parsed
	}

	if limit > 100 {
		limit = 100
	}

	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid_offset", "offset must not be negative")
			return
		}

		offset = parsed
	}

	var status *string

	if value := r.URL.Query().Get("status"); value != "" {
		status = &value
	}

	var assigneeID *int64

	if value := r.URL.Query().Get("assignee_id"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed <= 0 {
			writeJSONError(w, http.StatusBadRequest, "invalid_assignee_id", "assignee_id must be positive")
			return
		}

		assigneeID = &parsed
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	tasks, err := h.taskService.GetAll(
		r.Context(),
		userID,
		model.TaskFilter{
			TeamID:     teamID,
			Status:     status,
			AssigneeID: assigneeID,
			Limit:      limit,
			Offset:     offset,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotTeamMember):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you are not a member of this team")
		default:
			log.Printf("get tasks: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		log.Printf("encode tasks response: %v", err)
	}
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	AssigneeID  *int64  `json:"assignee_id"`
	Version     int     `json:"version"`
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_task_id", "invalid task id")
		return
	}

	var req updateTaskRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	if req.Version <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_version", "version must be positive")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	err = h.taskService.Update(
		r.Context(),
		taskID,
		userID,
		service.UpdateTaskInput{
			Title:       req.Title,
			Description: req.Description,
			Status:      req.Status,
			AssigneeID:  req.AssigneeID,
			Version:     req.Version,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			writeJSONError(w, http.StatusNotFound, "task_not_found", "task not found")

		case errors.Is(err, service.ErrTaskForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have permission to update this task")

		case errors.Is(err, service.ErrTaskTitleRequired),
			errors.Is(err, service.ErrTaskStatusRequired):
			writeJSONError(w, http.StatusBadRequest, "validation_error", err.Error())

		case errors.Is(err, service.ErrAssigneeNotMember):
			writeJSONError(w, http.StatusBadRequest, "invalid_assignee", "assignee must be a member of the team")

		case errors.Is(err, service.ErrVersionConflict):
			writeJSONError(w, http.StatusConflict, "version_conflict", "task was updated by another request")

		default:
			log.Printf("update task: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_task_id", "invalid task id")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	history, err := h.taskService.GetHistory(r.Context(), taskID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			writeJSONError(w, http.StatusNotFound, "task_not_found", "task not found")

		case errors.Is(err, service.ErrTaskForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have access to this task")

		default:
			log.Printf("get task history: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(history); err != nil {
		log.Printf("encode task history: %v", err)
	}
}
