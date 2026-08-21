package repository

import (
	"context"
	"database/sql"
	"errors"
	"task_manager/internal/model"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

var ErrEmailAlreadyExists = errors.New("email already exists")

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, email string, passwordHash string, name string) (int64, error) {
	result, err := r.db.ExecContext(
		ctx,
		`INSERT INTO users (email, password_hash, name)
		 VALUES (?, ?, ?)`,
		email,
		passwordHash,
		name,
	)
	if err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return 0, ErrEmailAlreadyExists
		}
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id, email, password_hash, name, created_at
		FROM users
		WHERE email = ?
		`,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Name,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}
