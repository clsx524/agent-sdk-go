package agentsdk

import (
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/task"
)

// NewAgent creates a new agent with the given options
func NewAgent(options ...agent.Option) (*agent.Agent, error) {
	return agent.NewAgent(options...)
}

// WithLLM sets the LLM for the agent
func WithLLM(llm interfaces.LLM) agent.Option {
	return agent.WithLLM(llm)
}

// WithMemory sets the memory for the agent
func WithMemory(memory interfaces.Memory) agent.Option {
	return agent.WithMemory(memory)
}

// WithTools sets the tools for the agent
func WithTools(tools ...interfaces.Tool) agent.Option {
	return agent.WithTools(tools...)
}

// WithOrgID sets the organization ID for multi-tenancy
func WithOrgID(orgID string) agent.Option {
	return agent.WithOrgID(orgID)
}

// WithTracer sets the tracer for the agent
func WithTracer(tracer interfaces.Tracer) agent.Option {
	return agent.WithTracer(tracer)
}

// WithGuardrails sets the guardrails for the agent
func WithGuardrails(guardrails interfaces.Guardrails) agent.Option {
	return agent.WithGuardrails(guardrails)
}

// Task Execution

// NewTaskExecutor creates a new task executor
func NewTaskExecutor() *task.Executor {
	return task.NewExecutor()
}

// NewAPIClient creates a new API client for making API calls
func NewAPIClient(baseURL string, timeout time.Duration) *task.APIClient {
	return task.NewAPIClient(baseURL, timeout)
}

// NewTemporalClient creates a new Temporal client for executing workflows
func NewTemporalClient(config task.TemporalConfig) *task.TemporalClient {
	return task.NewTemporalClient(config)
}

// APITask creates a task function for making an API request
func APITask(client *task.APIClient, req task.APIRequest) task.TaskFunc {
	return task.APITask(client, req)
}

// TemporalWorkflowTask creates a task function for executing a Temporal workflow
func TemporalWorkflowTask(client *task.TemporalClient, workflowName string) task.TaskFunc {
	return task.TemporalWorkflowTask(client, workflowName)
}

// TemporalWorkflowAsyncTask creates a task function for executing a Temporal workflow asynchronously
func TemporalWorkflowAsyncTask(client *task.TemporalClient, workflowName string) task.TaskFunc {
	return task.TemporalWorkflowAsyncTask(client, workflowName)
}
