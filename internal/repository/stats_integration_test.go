//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestStatsRepository_GetTeamStats(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Fatal("TEST_MYSQL_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("ping db: %v", err)
	}

	ctx := context.Background()

	user1ID := insertTestUser(t, ctx, db, "stats1@example.com", "User One")
	user2ID := insertTestUser(t, ctx, db, "stats2@example.com", "User Two")

	teamID := insertTestTeam(t, ctx, db, user1ID, user2ID)

	task1ID := insertTestTask(t, ctx, db, teamID, user1ID, user1ID, "done", time.Now().Add(-2*time.Hour), time.Now().Add(-1*time.Hour))

	task2ID := insertTestTask(t, ctx, db, teamID, user1ID, user2ID, "done", time.Now().Add(-4*time.Hour), time.Now().Add(-2*time.Hour))

	_ = insertTestTask(t, ctx, db, teamID, user1ID, user2ID, "todo", time.Now(), time.Time{})

	insertTestComment(t, ctx, db, task1ID, user1ID, "first")
	insertTestComment(t, ctx, db, task2ID, user2ID, "second")

	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM task_comments WHERE task_id IN (?, ?)`, task1ID, task2ID)
		_, _ = db.ExecContext(ctx, `DELETE FROM tasks WHERE team_id = ?`, teamID)
		_, _ = db.ExecContext(ctx, `DELETE FROM team_members WHERE team_id = ?`, teamID)
		_, _ = db.ExecContext(ctx, `DELETE FROM teams WHERE id = ?`, teamID)
		_, _ = db.ExecContext(ctx, `DELETE FROM users WHERE id IN (?, ?)`, user1ID, user2ID)
	})

	repo := NewStatsRepository(db)

	stats, err := repo.GetTeamStats(ctx, teamID)
	if err != nil {
		t.Fatalf("get team stats: %v", err)
	}

	if stats.TasksByStatus["done"] != 2 {
		t.Fatalf("expected 2 done tasks, got %d", stats.TasksByStatus["done"])
	}

	if stats.TasksByStatus["todo"] != 1 {
		t.Fatalf("expected 1 todo task, got %d", stats.TasksByStatus["todo"])
	}

	if stats.CommentsCount != 2 {
		t.Fatalf("expected 2 comments, got %d", stats.CommentsCount)
	}

	if len(stats.TopAssignees) != 2 {
		t.Fatalf("expected 2 top assignees, got %d", len(stats.TopAssignees))
	}

	if stats.AvgCloseTimeSeconds == nil {
		t.Fatal("expected avg close time, got nil")
	}

	expectedAvg := float64(5400)

	if *stats.AvgCloseTimeSeconds < expectedAvg-5 ||
		*stats.AvgCloseTimeSeconds > expectedAvg+5 {
		t.Fatalf(
			"expected avg close time around %.0f seconds, got %.0f",
			expectedAvg,
			*stats.AvgCloseTimeSeconds,
		)
	}
}

func insertTestUser(t *testing.T, ctx context.Context, db *sql.DB, email string, name string) int64 {
	t.Helper()

	result, err := db.ExecContext(
		ctx,
		`
		INSERT INTO users (email, password_hash, name)
		VALUES (?, ?, ?)
		`,
		email,
		"test_hash",
		name,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("user last insert id: %v", err)
	}

	return id
}

func insertTestTeam(t *testing.T, ctx context.Context, db *sql.DB, ownerID int64, memberID int64) int64 {
	t.Helper()

	result, err := db.ExecContext(
		ctx,
		`
		INSERT INTO teams (name, created_by)
		VALUES (?, ?)
		`,
		"Stats Test Team",
		ownerID,
	)
	if err != nil {
		t.Fatalf("insert team: %v", err)
	}

	teamID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("team last insert id: %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, 'owner')
		`,
		teamID,
		ownerID,
	)
	if err != nil {
		t.Fatalf("insert owner: %v", err)
	}

	_, err = db.ExecContext(
		ctx,
		`
		INSERT INTO team_members (team_id, user_id, role)
		VALUES (?, ?, 'member')
		`,
		teamID,
		memberID,
	)
	if err != nil {
		t.Fatalf("insert member: %v", err)
	}
	return teamID
}

func insertTestTask(t *testing.T, ctx context.Context, db *sql.DB, teamID int64, createdBy int64, assigneeID int64, status string, createdAt time.Time, closedAt time.Time) int64 {
	t.Helper()

	var closed any
	if closedAt.IsZero() {
		closed = nil
	} else {
		closed = closedAt
	}

	result, err := db.ExecContext(
		ctx,
		`
		INSERT INTO tasks (
			team_id,
			title,
			description,
			status,
			created_by,
			assignee_id,
			created_at,
			updated_at,
			closed_at,
			version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 1)
		`,
		teamID,
		"Stats test task",
		"test",
		status,
		createdBy,
		assigneeID,
		createdAt,
		createdAt,
		closed,
	)
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("task last insert id: %v", err)
	}
	return id
}

func insertTestComment(t *testing.T, ctx context.Context, db *sql.DB, taskID int64, userID int64, content string) {
	t.Helper()

	_, err := db.ExecContext(
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
		t.Fatalf("insert comment: %v", err)
	}
}
