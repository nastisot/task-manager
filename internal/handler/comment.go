package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"task_manager/internal/middleware"
	"task_manager/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{
		commentService: commentService,
	}
}

type createCommentRequest struct {
	Content string `json:"content"`
}

type createCommentResponse struct {
	ID int64 `json:"id"`
}

func (h *CommentHandler) Create(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || taskID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_task_id", "invalid task id")
		return
	}

	var req createCommentRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	id, err := h.commentService.Create(r.Context(), taskID, userID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCommentContentRequired):
			writeJSONError(w, http.StatusBadRequest, "validation_error", err.Error())

		case errors.Is(err, service.ErrTaskNotFound):
			writeJSONError(w, http.StatusNotFound, "task_not_found", "task not found")

		case errors.Is(err, service.ErrTaskForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have access to this task")

		default:
			log.Printf("create comment: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createCommentResponse{ID: id}); err != nil {
		log.Printf("encode create comment response: %v", err)
	}
}

func (h *CommentHandler) GetAll(w http.ResponseWriter, r *http.Request) {
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

	comments, err := h.commentService.GetAll(r.Context(), taskID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTaskNotFound):
			writeJSONError(w, http.StatusNotFound, "task_not_found", "task not found")

		case errors.Is(err, service.ErrTaskForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have access to this task")

		default:
			log.Printf("get comments: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(comments); err != nil {
		log.Printf("encode comments response: %v", err)
	}
}
