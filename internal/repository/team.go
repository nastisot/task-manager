package repository

import (
	"context"
	"database/sql"
	"errors"
	"task_manager/internal/model"
)

var ErrMemberNotFound = errors.New("team member not found")

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, name string, createdBy int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(
		ctx,
		`
		INSERT INTO teams (name, created_by)
		VALUES (?, ?)
		`,
		name,
		createdBy,
	)
	if err != nil {
		return 0, err
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, 'owner')
		`,
		teamID,
		createdBy,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return teamID, nil
}

func (r *TeamRepository) GetByUserID(ctx context.Context, userID int64) ([]model.Team, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT t.id, t.name, t.created_by, t.created_at
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		WHERE tm.user_id = ?
		ORDER BY t.id
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := make([]model.Team, 0)

	for rows.Next() {
		var team model.Team

		if err := rows.Scan(
			&team.ID,
			&team.Name,
			&team.CreatedBy,
			&team.CreatedAt,
		); err != nil {
			return nil, err
		}

		teams = append(teams, team)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return teams, nil
}

func (r *TeamRepository) GetMemberRole(ctx context.Context, teamID int64, userID int64) (string, error) {
	var role string

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT role
		FROM team_members
		WHERE team_id = ? AND user_id = ?
		`,
		teamID,
		userID,
	).Scan(&role)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID int64, userID int64, role string) error {
	_, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, ?)
		`,
		teamID,
		userID,
		role,
	)
	return err
}

func (r *TeamRepository) UpdateMemberRole(ctx context.Context, teamID int64, userID int64, role string) error {
	result, err := r.db.ExecContext(
		ctx,
		`
		UPDATE team_members
		SET role = ?
		WHERE team_id = ? AND user_id = ?
		`,
		role,
		teamID,
		userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrMemberNotFound
	}

	return nil
}
