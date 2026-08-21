package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"task_manager/internal/model"
)

type StatsRepository struct {
	db *sql.DB
}

func NewStatsRepository(db *sql.DB) *StatsRepository {
	return &StatsRepository{db: db}
}

func (r *StatsRepository) GetTeamStats(
	ctx context.Context,
	teamID int64,
) (*model.TeamStats, error) {
	const query = `
WITH
status_stats AS (
	SELECT JSON_OBJECTAGG(status, cnt) AS tasks_by_status
	FROM (
		SELECT status, COUNT(*) AS cnt
		FROM tasks
		WHERE team_id = ?
		GROUP BY status
	) s
),
top_assignees AS (
	SELECT JSON_ARRAYAGG(JSON_OBJECT('user_id', user_id, 'name', name, 'closed_tasks', closed_tasks)) AS top_assignees
	FROM (
		SELECT t.assignee_id AS user_id, u.name AS name, COUNT(*) AS closed_tasks
		FROM tasks t
		JOIN users u ON u.id = t.assignee_id
		WHERE t.team_id = ? AND t.closed_at IS NOT NULL AND t.closed_at >= CURRENT_TIMESTAMP - INTERVAL 30 DAY
		GROUP BY t.assignee_id, u.name
		ORDER BY closed_tasks DESC, t.assignee_id
		LIMIT 3
	) a
),
avg_close AS (
	SELECT AVG(TIMESTAMPDIFF(SECOND, created_at, closed_at)) AS avg_close_time_seconds
	FROM tasks
	WHERE team_id = ? AND closed_at IS NOT NULL
),
comment_stats AS (
	SELECT COUNT(tc.id) AS comments_count
	FROM tasks t
	LEFT JOIN task_comments tc ON tc.task_id = t.id
	WHERE t.team_id = ?
)
SELECT COALESCE(status_stats.tasks_by_status, JSON_OBJECT()), COALESCE(top_assignees.top_assignees, JSON_ARRAY()), avg_close.avg_close_time_seconds, comment_stats.comments_count
FROM status_stats
CROSS JOIN top_assignees
CROSS JOIN avg_close
CROSS JOIN comment_stats;
`

	var (
		statusJSON       []byte
		topAssigneesJSON []byte
		avgClose         sql.NullFloat64
		commentsCount    int64
	)

	err := r.db.QueryRowContext(
		ctx,
		query,
		teamID,
		teamID,
		teamID,
		teamID,
	).Scan(
		&statusJSON,
		&topAssigneesJSON,
		&avgClose,
		&commentsCount,
	)
	if err != nil {
		return nil, err
	}

	stats := &model.TeamStats{
		TasksByStatus: make(map[string]int64),
		TopAssignees:  make([]model.AssigneeStats, 0),
		CommentsCount: commentsCount,
	}

	if err := json.Unmarshal(statusJSON, &stats.TasksByStatus); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(topAssigneesJSON, &stats.TopAssignees); err != nil {
		return nil, err
	}

	if avgClose.Valid {
		stats.AvgCloseTimeSeconds = &avgClose.Float64
	}

	return stats, nil
}
