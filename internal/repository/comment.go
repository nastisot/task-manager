package repository

import (
	"context"
	"database/sql"
	"task_manager/internal/model"
)

type CommentRepository struct {
	db *sql.DB
}

func NewCommentRepository(db *sql.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(ctx context.Context, taskID int64, userID int64, content string) (int64, error) {
	result, err := r.db.ExecContext(
		ctx,
		`
		INSERT INTO task_comments (task_id, user_id, content)
		VALUES (?, ?, ?)
		`,
		taskID,
		userID,
		content,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (r *CommentRepository) GetByTaskID(ctx context.Context, taskID int64) ([]model.TaskComment, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`
		SELECT id, task_id, user_id, content, created_at
		FROM task_comments
		WHERE task_id = ?
		ORDER BY id ASC
		`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := make([]model.TaskComment, 0)

	for rows.Next() {
		var comment model.TaskComment

		if err := rows.Scan(
			&comment.ID,
			&comment.TaskID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
		); err != nil {
			return nil, err
		}

		comments = append(comments, comment)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}
