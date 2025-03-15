package task

import (
	"context"
	"fmt"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Executor implements the TaskExecutor interface
type Executor struct {
	// Add fields as needed for configuration
	taskRegistry map[string]TaskFunc
	// Add more fields as needed
}

// TaskFunc is a function that executes a task
type TaskFunc func(ctx context.Context, params interface{}) (interface{}, error)

// NewExecutor creates a new task executor
func NewExecutor() *Executor {
	return &Executor{
		taskRegistry: make(map[string]TaskFunc),
	}
}

// RegisterTask registers a task function with the executor
func (e *Executor) RegisterTask(name string, taskFunc TaskFunc) {
	e.taskRegistry[name] = taskFunc
}

// ExecuteSync executes a task synchronously
func (e *Executor) ExecuteSync(ctx context.Context, taskName string, params interface{}, opts *interfaces.TaskOptions) (*interfaces.TaskResult, error) {
	taskFunc, exists := e.taskRegistry[taskName]
	if !exists {
		return nil, fmt.Errorf("task %s not registered", taskName)
	}

	// Apply timeout if specified
	if opts != nil && opts.Timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *opts.Timeout)
		defer cancel()
	}

	// Execute the task with retry if specified
	result, err := e.executeWithRetry(ctx, taskFunc, params, opts)

	taskResult := &interfaces.TaskResult{
		Data:     result,
		Error:    err,
		Metadata: make(map[string]interface{}),
	}

	// Add metadata
	if opts != nil && opts.Metadata != nil {
		for k, v := range opts.Metadata {
			taskResult.Metadata[k] = v
		}
	}

	taskResult.Metadata["executionTime"] = time.Now().UTC()

	return taskResult, nil
}

// ExecuteAsync executes a task asynchronously
func (e *Executor) ExecuteAsync(ctx context.Context, taskName string, params interface{}, opts *interfaces.TaskOptions) (<-chan *interfaces.TaskResult, error) {
	taskFunc, exists := e.taskRegistry[taskName]
	if !exists {
		return nil, fmt.Errorf("task %s not registered", taskName)
	}

	resultChan := make(chan *interfaces.TaskResult, 1)

	go func() {
		defer close(resultChan)

		// Create a new context for the async task
		asyncCtx := ctx
		if opts != nil && opts.Timeout != nil {
			var cancel context.CancelFunc
			asyncCtx, cancel = context.WithTimeout(ctx, *opts.Timeout)
			defer cancel()
		}

		// Execute the task with retry if specified
		result, err := e.executeWithRetry(asyncCtx, taskFunc, params, opts)

		taskResult := &interfaces.TaskResult{
			Data:     result,
			Error:    err,
			Metadata: make(map[string]interface{}),
		}

		// Add metadata
		if opts != nil && opts.Metadata != nil {
			for k, v := range opts.Metadata {
				taskResult.Metadata[k] = v
			}
		}

		taskResult.Metadata["executionTime"] = time.Now().UTC()
		taskResult.Metadata["taskID"] = uuid.New().String()

		select {
		case resultChan <- taskResult:
			// Result sent successfully
		case <-asyncCtx.Done():
			// Context was canceled or timed out
			log.Warn().Str("taskName", taskName).Msg("Task execution was canceled or timed out")
		}
	}()

	return resultChan, nil
}

// ExecuteWorkflow initiates a temporal workflow
func (e *Executor) ExecuteWorkflow(ctx context.Context, workflowName string, params interface{}, opts *interfaces.TaskOptions) (*interfaces.TaskResult, error) {
	// This is a placeholder for actual temporal workflow implementation
	// In a real implementation, this would use the Temporal SDK to start a workflow

	return &interfaces.TaskResult{
		Data:  nil,
		Error: fmt.Errorf("temporal workflow execution not implemented"),
		Metadata: map[string]interface{}{
			"workflowName":  workflowName,
			"executionTime": time.Now().UTC(),
		},
	}, nil
}

// ExecuteWorkflowAsync initiates a temporal workflow asynchronously
func (e *Executor) ExecuteWorkflowAsync(ctx context.Context, workflowName string, params interface{}, opts *interfaces.TaskOptions) (<-chan *interfaces.TaskResult, error) {
	// This is a placeholder for actual temporal workflow implementation
	// In a real implementation, this would use the Temporal SDK to start a workflow asynchronously

	resultChan := make(chan *interfaces.TaskResult, 1)

	go func() {
		defer close(resultChan)

		// Simulate workflow execution
		time.Sleep(100 * time.Millisecond)

		resultChan <- &interfaces.TaskResult{
			Data:  nil,
			Error: fmt.Errorf("temporal workflow execution not implemented"),
			Metadata: map[string]interface{}{
				"workflowName":  workflowName,
				"executionTime": time.Now().UTC(),
				"taskID":        uuid.New().String(),
			},
		}
	}()

	return resultChan, nil
}

// CancelTask cancels a running task
func (e *Executor) CancelTask(ctx context.Context, taskID string) error {
	// This is a placeholder for actual task cancellation
	// In a real implementation, this would track running tasks and cancel them

	return fmt.Errorf("task cancellation not implemented")
}

// GetTaskStatus gets the status of a task
func (e *Executor) GetTaskStatus(ctx context.Context, taskID string) (string, error) {
	// This is a placeholder for actual task status retrieval
	// In a real implementation, this would track task status and return it

	return "unknown", fmt.Errorf("task status retrieval not implemented")
}

// executeWithRetry executes a task with retry if specified in the options
func (e *Executor) executeWithRetry(ctx context.Context, taskFunc TaskFunc, params interface{}, opts *interfaces.TaskOptions) (interface{}, error) {
	if opts == nil || opts.RetryPolicy == nil {
		// No retry policy, execute once
		return taskFunc(ctx, params)
	}

	retryPolicy := opts.RetryPolicy
	var lastErr error
	backoff := retryPolicy.InitialBackoff

	for attempt := 0; attempt <= retryPolicy.MaxRetries; attempt++ {
		// Execute the task
		result, err := taskFunc(ctx, params)
		if err == nil {
			// Success, return the result
			return result, nil
		}

		lastErr = err

		// Check if we should retry
		if attempt >= retryPolicy.MaxRetries {
			break
		}

		// Wait for backoff duration
		select {
		case <-ctx.Done():
			// Context was canceled or timed out
			return nil, ctx.Err()
		case <-time.After(backoff):
			// Backoff completed, continue with next attempt
		}

		// Increase backoff for next attempt
		backoff = time.Duration(float64(backoff) * retryPolicy.BackoffMultiplier)
		if backoff > retryPolicy.MaxBackoff {
			backoff = retryPolicy.MaxBackoff
		}
	}

	return nil, fmt.Errorf("task failed after %d attempts: %w", retryPolicy.MaxRetries+1, lastErr)
}
