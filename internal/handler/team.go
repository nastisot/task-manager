package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"task_manager/internal/repository"

	"task_manager/internal/middleware"
	"task_manager/internal/service"
)

type TeamHandler struct {
	teamService *service.TeamService
}

func NewTeamHandler(teamService *service.TeamService) *TeamHandler {
	return &TeamHandler{
		teamService: teamService,
	}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

type createTeamResponse struct {
	ID int64 `json:"id"`
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTeamRequest

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

	teamID, err := h.teamService.Create(r.Context(), req.Name, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTeamNameRequired):
			writeJSONError(w, http.StatusBadRequest, "validation_error", err.Error())
		default:
			log.Printf("create team: %v", err)

			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createTeamResponse{ID: teamID}); err != nil {
		log.Printf("encode create team response: %v", err)
	}
}

func (h *TeamHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	teams, err := h.teamService.GetByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("get teams: %v", err)
		writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(teams); err != nil {
		log.Printf("encode teams response: %v", err)
	}
}

type inviteMemberRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	teamIDStr := r.PathValue("id")

	teamID, err := strconv.ParseInt(teamIDStr, 10, 64)
	if err != nil || teamID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_team_id", "invalid team id")
		return
	}

	var req inviteMemberRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	err = h.teamService.Invite(r.Context(), teamID, currentUserID, req.UserID, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have permission to invite users")

		case errors.Is(err, service.ErrCannotAssignOwner),
			errors.Is(err, service.ErrInvalidRole):

			writeJSONError(w, http.StatusBadRequest, "invalid_role", err.Error())

		case errors.Is(err, service.ErrAlreadyMember):
			writeJSONError(w, http.StatusConflict, "already_member", "user is already a team member")

		default:
			log.Printf("invite team member: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

func (h *TeamHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || teamID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_team_id", "invalid team id")
		return
	}

	targetUserID, err := strconv.ParseInt(r.PathValue("user_id"), 10, 64)
	if err != nil || targetUserID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}

	var req updateRoleRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	currentUserID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	err = h.teamService.UpdateMemberRole(r.Context(), teamID, currentUserID, targetUserID, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you do not have permission to change roles")

		case errors.Is(err, service.ErrInvalidRole),
			errors.Is(err, service.ErrCannotChangeOwner):

			writeJSONError(w, http.StatusBadRequest, "invalid_role", err.Error())

		case errors.Is(err, repository.ErrMemberNotFound):
			writeJSONError(w, http.StatusNotFound, "member_not_found", "team member not found")

		default:
			log.Printf("update member role: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
