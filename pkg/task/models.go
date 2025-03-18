package task

import (
	"time"
)

// Status represents the current status of a task or step
type Status string

// Task statuses
const (
	StatusPending   Status = "pending"
	StatusPlanning  Status = "planning"
	StatusApproval  Status = "awaiting_approval"
	StatusExecuting Status = "executing"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// Step statuses
const (
	StepStatusPending   Status = "pending"
	StepStatusExecuting Status = "executing"
	StepStatusCompleted Status = "completed"
	StepStatusFailed    Status = "failed"
)

// Task represents an infrastructure task to be executed
type Task struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	Status      Status                 `json:"status"`
	Plan        *Plan                  `json:"plan,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	UserID      string                 `json:"user_id"`
	Logs        []LogEntry             `json:"logs,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // For extensibility
}

// Plan represents the execution plan for a task
type Plan struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"task_id"`
	Steps      []Step     `json:"steps"`
	CreatedAt  time.Time  `json:"created_at"`
	ApprovedAt *time.Time `json:"approved_at,omitempty"`
	IsApproved bool       `json:"is_approved"`
}

// Step represents a single step in an execution plan
type Step struct {
	ID          string                 `json:"id"`
	PlanID      string                 `json:"plan_id"`
	Description string                 `json:"description"`
	Status      Status                 `json:"status"`
	Order       int                    `json:"order"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// LogEntry represents a log entry for a task
type LogEntry struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	Message   string    `json:"message"`
	Level     string    `json:"level"` // info, warning, error
	Timestamp time.Time `json:"timestamp"`
}

// CreateTaskRequest represents the request to create a new task
type CreateTaskRequest struct {
	Description string                 `json:"description"`
	UserID      string                 `json:"user_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ApproveTaskPlanRequest represents the request to approve a task plan
type ApproveTaskPlanRequest struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback,omitempty"`
}

// TaskUpdate represents an update to a task
type TaskUpdate struct {
	Type        string `json:"type"` // add_step, modify_step, remove_step
	StepID      string `json:"step_id,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// TaskFilter represents filters for querying tasks
type TaskFilter struct {
	UserID        string     `json:"user_id,omitempty"`
	Status        []Status   `json:"status,omitempty"`
	CreatedAfter  *time.Time `json:"created_after,omitempty"`
	CreatedBefore *time.Time `json:"created_before,omitempty"`
}
