package model

type AssigneeStats struct {
	UserID      int64  `json:"user_id"`
	Name        string `json:"name"`
	ClosedTasks int64  `json:"closed_tasks"`
}

type TeamStats struct {
	TasksByStatus       map[string]int64 `json:"tasks_by_status"`
	TopAssignees        []AssigneeStats  `json:"top_assignees"`
	AvgCloseTimeSeconds *float64         `json:"avg_close_time_seconds"`
	CommentsCount       int64            `json:"comments_count"`
}
