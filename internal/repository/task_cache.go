package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"task_manager/internal/model"
	"time"

	"github.com/redis/go-redis/v9"
)

type TaskCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewTaskCache(client *redis.Client) *TaskCache {
	return &TaskCache{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (c *TaskCache) key(filter model.TaskFilter) string {
	status := "all"
	if filter.Status != nil {
		status = *filter.Status
	}

	assignee := "all"
	if filter.AssigneeID != nil {
		assignee = fmt.Sprintf("%d", *filter.AssigneeID)
	}

	return fmt.Sprintf(
		"tasks:team:%d:status:%s:assignee:%s:limit:%d:offset:%d",
		filter.TeamID,
		status,
		assignee,
		filter.Limit,
		filter.Offset,
	)
}

func (c *TaskCache) Get(ctx context.Context, filter model.TaskFilter) ([]model.Task, bool, error) {
	data, err := c.client.Get(ctx, c.key(filter)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}

		return nil, false, err
	}

	var tasks []model.Task

	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, false, err
	}

	return tasks, true, nil
}

func (c *TaskCache) Set(ctx context.Context, filter model.TaskFilter, tasks []model.Task) error {
	data, err := json.Marshal(tasks)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, c.key(filter), data, c.ttl).Err()
}

func (c *TaskCache) InvalidateTeam(ctx context.Context, teamID int64) error {
	pattern := fmt.Sprintf("tasks:team:%d:*", teamID)

	var cursor uint64

	for {
		keys, nextCursor, err := c.client.Scan(
			ctx,
			cursor,
			pattern,
			100,
		).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
