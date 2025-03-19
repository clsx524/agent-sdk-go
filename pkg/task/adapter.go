// Package task provides task management functionality.
package task

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
