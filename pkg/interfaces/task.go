package interfaces

import (
	"context"
	"time"
)

// TaskResult represents the result of a task execution
type TaskResult struct {
	// Data contains the result data
	Data interface{}
	// Error contains any error that occurred during task execution
	Error error
	// Metadata contains additional information about the task execution
	Metadata map[string]interface{}
}

// TaskOptions represents options for task execution
type TaskOptions struct {
	// Timeout specifies the maximum duration for task execution
	Timeout *time.Duration
	// RetryPolicy specifies the retry policy for the task
	RetryPolicy *RetryPolicy
	// Metadata contains additional information for the task execution
	Metadata map[string]interface{}
}

// RetryPolicy defines how tasks should be retried
type RetryPolicy struct {
	// MaxRetries is the maximum number of retries
	MaxRetries int
	// InitialBackoff is the initial backoff duration
	InitialBackoff time.Duration
	// MaxBackoff is the maximum backoff duration
	MaxBackoff time.Duration
	// BackoffMultiplier is the multiplier for backoff duration after each retry
	BackoffMultiplier float64
}

// TaskExecutor is the interface for executing tasks
type TaskExecutor interface {
	// ExecuteSync executes a task synchronously
	ExecuteSync(ctx context.Context, taskName string, params interface{}, opts *TaskOptions) (*TaskResult, error)

	// ExecuteAsync executes a task asynchronously and returns a channel for the result
	ExecuteAsync(ctx context.Context, taskName string, params interface{}, opts *TaskOptions) (<-chan *TaskResult, error)

	// ExecuteWorkflow initiates a temporal workflow
	ExecuteWorkflow(ctx context.Context, workflowName string, params interface{}, opts *TaskOptions) (*TaskResult, error)

	// ExecuteWorkflowAsync initiates a temporal workflow asynchronously
	ExecuteWorkflowAsync(ctx context.Context, workflowName string, params interface{}, opts *TaskOptions) (<-chan *TaskResult, error)

	// CancelTask cancels a running task
	CancelTask(ctx context.Context, taskID string) error

	// GetTaskStatus gets the status of a task
	GetTaskStatus(ctx context.Context, taskID string) (string, error)
}
