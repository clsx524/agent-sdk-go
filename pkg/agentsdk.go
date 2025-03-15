package agentsdk

import (
	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
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
