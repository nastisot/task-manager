package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"task_manager/internal/repository"
	"task_manager/internal/service"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type registerResponse struct {
	ID int64 `json:"id"`
}

func writeJSONError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(errorResponse{
		Error:   code,
		Message: message,
	})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not allowed", "method not allowed")
		return
	}

	var req registerRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	id, err := h.authService.Register(r.Context(), req.Email, req.Password, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailRequired), errors.Is(err, service.ErrNameRequired), errors.Is(err, service.ErrPasswordTooShort):
			writeJSONError(w, http.StatusBadRequest, "validation_error", err.Error())
		case errors.Is(err, repository.ErrEmailAlreadyExists):
			writeJSONError(w, http.StatusConflict, "email_already_exists", "user with email already exists")
		default:
			log.Printf("register user: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(registerResponse{ID: id}); err != nil {
		log.Printf("encode register response: %v", err)
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req loginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			writeJSONError(w, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		default:
			log.Printf("login user: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(loginResponse{Token: token}); err != nil {
		log.Printf("encode login response: %v", err)
	}
}
