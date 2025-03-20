// Package task provides task management functionality.
package task

import (
	"context"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
)

// TaskAdapter defines a generic interface for adapting SDK task models
// to agent-specific models and vice versa. This pattern helps separate
// the concerns of the SDK from agent-specific implementations.
//
// Implementing this interface allows agents to use their own domain models
// while still leveraging the SDK's task management capabilities.
type TaskAdapter[AgentTask any, AgentCreateRequest any, AgentApprovalRequest any, AgentTaskUpdate any] interface {
	// ToSDK conversions (Agent -> SDK)

	// ConvertCreateRequest converts an agent-specific create request to an SDK create request
	ConvertCreateRequest(req AgentCreateRequest) CreateTaskRequest

	// ConvertApproveRequest converts an agent-specific approve request to an SDK approve request
	ConvertApproveRequest(req AgentApprovalRequest) ApproveTaskPlanRequest

	// ConvertTaskUpdates converts agent-specific task updates to SDK task updates
	ConvertTaskUpdates(updates []AgentTaskUpdate) []TaskUpdate

	// FromSDK conversions (SDK -> Agent)

	// ConvertTask converts an SDK task to an agent-specific task
	ConvertTask(sdkTask *Task) AgentTask

	// ConvertTasks converts a slice of SDK tasks to a slice of agent-specific tasks
	ConvertTasks(sdkTasks []*Task) []AgentTask
}

// AdapterOptions contains optional parameters for creating a task adapter
type AdapterOptions struct {
	// Additional options can be added here as needed
	IncludeMetadata bool
	DefaultUserID   string
}

// AdapterOption is a function that configures AdapterOptions
type AdapterOption func(*AdapterOptions)

// WithMetadata configures the adapter to include SDK metadata in conversions
func WithMetadata(include bool) AdapterOption {
	return func(opts *AdapterOptions) {
		opts.IncludeMetadata = include
	}
}

// WithDefaultUserID sets a default user ID for task creation
func WithDefaultUserID(userID string) AdapterOption {
	return func(opts *AdapterOptions) {
		opts.DefaultUserID = userID
	}
}

// AdapterService provides a generic service for adapting between SDK tasks and agent-specific tasks.
// It wraps the SDK's task service and provides methods for working with agent-specific task models.
type AdapterService[AgentTask any, AgentCreateRequest any, AgentApprovalRequest any, AgentTaskUpdate any] struct {
	sdkService Service
	adapter    TaskAdapter[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]
	logger     logging.Logger
}

// NewAdapterService creates a new adapter service for adapting between SDK and agent-specific task models.
// It provides a simple way for agents to work with their own task models while leveraging the SDK's task service.
func NewAdapterService[AgentTask any, AgentCreateRequest any, AgentApprovalRequest any, AgentTaskUpdate any](
	logger logging.Logger,
	sdkService Service,
	adapter TaskAdapter[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate],
) *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate] {
	return &AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]{
		sdkService: sdkService,
		adapter:    adapter,
		logger:     logger,
	}
}

// CreateTask creates a new task using the agent-specific task model.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) CreateTask(
	ctx context.Context,
	req AgentCreateRequest,
) (AgentTask, error) {
	s.logger.Debug(ctx, "Creating new task via adapter service", nil)

	// Convert the request to SDK format
	sdkReq := s.adapter.ConvertCreateRequest(req)

	// Create task using SDK service
	sdkTask, err := s.sdkService.CreateTask(ctx, sdkReq)
	if err != nil {
		s.logger.Error(ctx, "Failed to create task via SDK", map[string]interface{}{
			"error": err.Error(),
		})
		return *new(AgentTask), err
	}

	// Convert the SDK task back to agent-specific format
	task := s.adapter.ConvertTask(sdkTask)

	return task, nil
}

// GetTask retrieves a task by ID and returns it in the agent-specific format.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) GetTask(
	ctx context.Context,
	taskID string,
) (AgentTask, error) {
	// Get task using SDK service
	sdkTask, err := s.sdkService.GetTask(ctx, taskID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get task via SDK", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
		return *new(AgentTask), err
	}

	// Convert the SDK task back to agent-specific format
	task := s.adapter.ConvertTask(sdkTask)

	return task, nil
}

// ListTasks returns all tasks for a user in the agent-specific format.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) ListTasks(
	ctx context.Context,
	userID string,
) ([]AgentTask, error) {
	// Create filter for SDK service
	filter := TaskFilter{
		UserID: userID,
	}

	// List tasks using SDK service
	sdkTasks, err := s.sdkService.ListTasks(ctx, filter)
	if err != nil {
		s.logger.Error(ctx, "Failed to list tasks via SDK", map[string]interface{}{
			"error":   err.Error(),
			"user_id": userID,
		})
		return nil, err
	}

	// Convert the SDK tasks back to agent-specific format
	tasks := s.adapter.ConvertTasks(sdkTasks)

	return tasks, nil
}

// ApproveTaskPlan approves or rejects a task plan using the agent-specific approval model.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) ApproveTaskPlan(
	ctx context.Context,
	taskID string,
	req AgentApprovalRequest,
) (AgentTask, error) {
	// Convert the request to SDK format
	sdkReq := s.adapter.ConvertApproveRequest(req)

	// Approve task plan using SDK service
	sdkTask, err := s.sdkService.ApproveTaskPlan(ctx, taskID, sdkReq)
	if err != nil {
		s.logger.Error(ctx, "Failed to approve task plan via SDK", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
		return *new(AgentTask), err
	}

	// Convert the SDK task back to agent-specific format
	task := s.adapter.ConvertTask(sdkTask)

	return task, nil
}

// UpdateTask updates an existing task using agent-specific task updates.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) UpdateTask(
	ctx context.Context,
	taskID string,
	conversationID string,
	updates []AgentTaskUpdate,
) (AgentTask, error) {
	// Get the existing task to update its conversationID if needed
	sdkTask, err := s.sdkService.GetTask(ctx, taskID)
	if err != nil {
		s.logger.Error(ctx, "Failed to get task for update via SDK", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
		return *new(AgentTask), err
	}

	// Update the conversationID in the task if it doesn't match
	if sdkTask.ConversationID != conversationID && conversationID != "" {
		sdkTask.ConversationID = conversationID
	}

	// Convert the updates to SDK format
	sdkUpdates := s.adapter.ConvertTaskUpdates(updates)

	// Update task using SDK service
	sdkTask, err = s.sdkService.UpdateTask(ctx, taskID, sdkUpdates)
	if err != nil {
		s.logger.Error(ctx, "Failed to update task via SDK", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
		return *new(AgentTask), err
	}

	// Convert the SDK task back to agent-specific format
	task := s.adapter.ConvertTask(sdkTask)

	return task, nil
}

// AddTaskLog adds a log entry to a task.
func (s *AdapterService[AgentTask, AgentCreateRequest, AgentApprovalRequest, AgentTaskUpdate]) AddTaskLog(
	ctx context.Context,
	taskID string,
	message string,
	level string,
) error {
	// Add log using SDK service
	err := s.sdkService.AddTaskLog(ctx, taskID, message, level)
	if err != nil {
		s.logger.Error(ctx, "Failed to add task log via SDK", map[string]interface{}{
			"error":   err.Error(),
			"task_id": taskID,
		})
		return err
	}

	return nil
}

//
// DEFAULT ADAPTER IMPLEMENTATION
//

// DefaultTask provides a standard agent task model that can be embedded or used directly
type DefaultTask struct {
	ID             string                 `json:"id"`
	Description    string                 `json:"description"`
	Status         string                 `json:"status"`
	Title          string                 `json:"title,omitempty"`
	TaskKind       string                 `json:"task_kind,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	UserID         string                 `json:"user_id"`
	Steps          []DefaultStep          `json:"steps,omitempty"`
	Requirements   interface{}            `json:"requirements,omitempty"`
	Feedback       string                 `json:"feedback,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// DefaultStep represents a task step in the default model
type DefaultStep struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Error       string     `json:"error,omitempty"`
	Output      string     `json:"output,omitempty"`
}

// DefaultCreateRequest provides a standard create request model
type DefaultCreateRequest struct {
	Description string `json:"description"`
	UserID      string `json:"user_id"`
	Title       string `json:"title,omitempty"`
	TaskKind    string `json:"task_kind,omitempty"`
}

// DefaultApproveRequest provides a standard approve request model
type DefaultApproveRequest struct {
	Approved bool   `json:"approved"`
	Feedback string `json:"feedback,omitempty"`
}

// DefaultTaskUpdate provides a standard task update model
type DefaultTaskUpdate struct {
	Type        string `json:"type"`
	StepID      string `json:"step_id,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// DefaultTaskAdapter provides a standard implementation of TaskAdapter
// Agents can use this directly or embed it in their own adapter
type DefaultTaskAdapter struct {
	logger logging.Logger
}

// NewDefaultTaskAdapter creates a new default task adapter
func NewDefaultTaskAdapter(logger logging.Logger) TaskAdapter[*DefaultTask, DefaultCreateRequest, DefaultApproveRequest, DefaultTaskUpdate] {
	return &DefaultTaskAdapter{
		logger: logger,
	}
}

// ConvertCreateRequest converts a default create request to an SDK create request
func (a *DefaultTaskAdapter) ConvertCreateRequest(req DefaultCreateRequest) CreateTaskRequest {
	return CreateTaskRequest{
		Description: req.Description,
		UserID:      req.UserID,
		Title:       req.Title,
		TaskKind:    req.TaskKind,
		Metadata:    make(map[string]interface{}),
	}
}

// ConvertApproveRequest converts a default approve request to an SDK approve request
func (a *DefaultTaskAdapter) ConvertApproveRequest(req DefaultApproveRequest) ApproveTaskPlanRequest {
	return ApproveTaskPlanRequest(req)
}

// ConvertTaskUpdates converts default task updates to SDK task updates
func (a *DefaultTaskAdapter) ConvertTaskUpdates(updates []DefaultTaskUpdate) []TaskUpdate {
	sdkUpdates := make([]TaskUpdate, len(updates))
	for i, update := range updates {
		sdkUpdates[i] = TaskUpdate(update)
	}
	return sdkUpdates
}

// convertStepsToDefaultSteps converts SDK steps to default steps
func (a *DefaultTaskAdapter) convertStepsToDefaultSteps(sdkSteps []Step) []DefaultStep {
	defaultSteps := make([]DefaultStep, len(sdkSteps))
	for i, sdkStep := range sdkSteps {
		defaultSteps[i] = DefaultStep{
			ID:          sdkStep.ID,
			Description: sdkStep.Description,
			Status:      string(sdkStep.Status),
			StartTime:   sdkStep.StartedAt,
			EndTime:     sdkStep.CompletedAt,
			Error:       sdkStep.Error,
			Output:      sdkStep.Output,
		}
	}
	return defaultSteps
}

// ConvertTask converts an SDK task to a default task
func (a *DefaultTaskAdapter) ConvertTask(sdkTask *Task) *DefaultTask {
	if sdkTask == nil {
		return nil
	}

	task := &DefaultTask{
		ID:             sdkTask.ID,
		Description:    sdkTask.Description,
		Status:         string(sdkTask.Status),
		CreatedAt:      sdkTask.CreatedAt,
		UpdatedAt:      sdkTask.UpdatedAt,
		CompletedAt:    sdkTask.CompletedAt,
		UserID:         sdkTask.UserID,
		Title:          sdkTask.Title,
		TaskKind:       sdkTask.TaskKind,
		ConversationID: sdkTask.ConversationID,
		StartedAt:      sdkTask.StartedAt,
		Requirements:   sdkTask.Requirements,
		Feedback:       sdkTask.Feedback,
		Metadata:       sdkTask.Metadata,
	}

	// Convert steps using a unified approach
	if len(sdkTask.Steps) > 0 {
		task.Steps = a.convertStepsToDefaultSteps(sdkTask.Steps)
	} else if sdkTask.Plan != nil && len(sdkTask.Plan.Steps) > 0 {
		task.Steps = a.convertStepsToDefaultSteps(sdkTask.Plan.Steps)
	}

	return task
}

// ConvertTasks converts a slice of SDK tasks to a slice of default tasks
func (a *DefaultTaskAdapter) ConvertTasks(sdkTasks []*Task) []*DefaultTask {
	if sdkTasks == nil {
		return nil
	}

	tasks := make([]*DefaultTask, len(sdkTasks))
	for i, sdkTask := range sdkTasks {
		tasks[i] = a.ConvertTask(sdkTask)
	}
	return tasks
}
