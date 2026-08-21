package repository

import (
	"context"
	"database/sql"
	"errors"
	"task_manager/internal/model"
	"time"
)

var ErrVersionConflict = errors.New("task version conflict")

type TaskRepository struct {
	db *sql.DB
}

func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, teamID int64, title string, description string, status string, createdBy int64, assigneeID *int64, changes []byte) (int64, error) {
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
		INSERT INTO tasks (team_id, title, description, status, created_by, assignee_id)
		VALUES (?, ?, ?, ?, ?, ?)
		`,
		teamID,
		title,
		description,
		status,
		createdBy,
		assigneeID,
	)
	if err != nil {
		return 0, err
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO task_history (task_id, changed_by, changes)
		VALUES (?, ?, ?)
		`,
		taskID,
		createdBy,
		changes,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return taskID, nil
}

func (r *TaskRepository) GetAll(ctx context.Context, filter model.TaskFilter) ([]model.Task, error) {
	query := `
		SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		FROM tasks
		WHERE team_id = ?
	`

	args := []any{filter.TeamID}

	if filter.Status != nil {
		query += ` AND status = ?`
		args = append(args, *filter.Status)
	}

	if filter.AssigneeID != nil {
		query += ` AND assignee_id = ?`
		args = append(args, *filter.AssigneeID)
	}

	query += ` ORDER BY id DESC LIMIT ? OFFSET ?`

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]model.Task, 0)

	for rows.Next() {
		var task model.Task

		if err := rows.Scan(
			&task.ID,
			&task.TeamID,
			&task.Title,
			&task.Description,
			&task.Status,
			&task.CreatedBy,
			&task.AssigneeID,
			&task.CreatedAt,
			&task.UpdatedAt,
			&task.ClosedAt,
			&task.Version,
		); err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, taskID int64) (*model.Task, error) {
	var task model.Task

	err := r.db.QueryRowContext(
		ctx,
		`
		SELECT id, team_id, title, description, status, created_by, assignee_id, created_at, updated_at, closed_at, version
		FROM tasks
		WHERE id = ?
		`,
		taskID,
	).Scan(
		&task.ID,
		&task.TeamID,
		&task.Title,
		&task.Description,
		&task.Status,
		&task.CreatedBy,
		&task.AssigneeID,
		&task.CreatedAt,
		&task.UpdatedAt,
		&task.ClosedAt,
		&task.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &task, nil
}

func (r *TaskRepository) Update(ctx context.Context, taskID int64, expectedVersion int, title string, description string, status string, assigneeID *int64, closedAt *time.Time, changedBy int64, changes []byte) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(
		ctx,
		`
		UPDATE tasks
		SET title = ?, description = ?, status = ?, assignee_id = ?, closed_at = ?, updated_at = CURRENT_TIMESTAMP, version = version + 1
		WHERE id = ? AND version = ?
		`,
		title,
		description,
		status,
		assigneeID,
		closedAt,
		taskID,
		expectedVersion,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrVersionConflict
	}

	_, err = tx.ExecContext(
		ctx,
		`
		INSERT INTO task_history (task_id, changed_by, changes)
		VALUES (?, ?, ?)
		`,
		taskID,
		changedBy,
		changes,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *TaskRepository) GetHistory(ctx context.Context, taskID int64) ([]model.TaskHistory, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT id, task_id, changed_by, changes, created_at
		FROM task_history
		WHERE task_id = ?
		ORDER BY id ASC
		`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := make([]model.TaskHistory, 0)

	for rows.Next() {
		var item model.TaskHistory
		var changes []byte

		if err := rows.Scan(
			&item.ID,
			&item.TaskID,
			&item.ChangedBy,
			&changes,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		item.Changes = changes
		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return history, nil
}
