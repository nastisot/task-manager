package service

import (
	"context"
	"errors"
	"strings"
	"task_manager/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailRequired      = errors.New("email is required")
	ErrNameRequired       = errors.New("name is required")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *JWTManager
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

func (s *AuthService) Register(ctx context.Context, email string, password string, name string) (int64, error) {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if email == "" {
		return 0, ErrEmailRequired
	}
	if name == "" {
		return 0, ErrNameRequired
	}
	if len(password) < 8 {
		return 0, ErrPasswordTooShort
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	return s.userRepo.Create(ctx, email, string(passwordHash), name)
}

func (s *AuthService) Login(ctx context.Context, email string, password string) (string, error) {
	email = strings.TrimSpace(email)

	if email == "" || password == "" {
		return "", ErrInvalidCredentials
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if user == nil {
		return "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, err := s.jwtManager.Generate(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
