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

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

func (h *StatsHandler) GetTeamStats(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseInt(r.PathValue("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "invalid_team_id", "invalid team id")
		return
	}

	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	stats, err := h.statsService.GetTeamStats(r.Context(), teamID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeJSONError(w, http.StatusForbidden, "forbidden", "you don't have permission to view team stats")
		default:
			log.Printf("get team stats: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("encode team stats: %v", err)
	}
}
