package agent

import (
	"context"
	"fmt"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
)

// Agent represents an AI agent
type Agent struct {
	llm          interfaces.LLM
	memory       interfaces.Memory
	tools        []interfaces.Tool
	orgID        string
	tracer       interfaces.Tracer
	guardrails   interfaces.Guardrails
	systemPrompt string
}

// Option represents an option for configuring an agent
type Option func(*Agent)

// WithLLM sets the LLM for the agent
func WithLLM(llm interfaces.LLM) Option {
	return func(a *Agent) {
		a.llm = llm
	}
}

// WithMemory sets the memory for the agent
func WithMemory(memory interfaces.Memory) Option {
	return func(a *Agent) {
		a.memory = memory
	}
}

// WithTools sets the tools for the agent
func WithTools(tools ...interfaces.Tool) Option {
	return func(a *Agent) {
		a.tools = tools
	}
}

// WithOrgID sets the organization ID for multi-tenancy
func WithOrgID(orgID string) Option {
	return func(a *Agent) {
		a.orgID = orgID
	}
}

// WithTracer sets the tracer for the agent
func WithTracer(tracer interfaces.Tracer) Option {
	return func(a *Agent) {
		a.tracer = tracer
	}
}

// WithGuardrails sets the guardrails for the agent
func WithGuardrails(guardrails interfaces.Guardrails) Option {
	return func(a *Agent) {
		a.guardrails = guardrails
	}
}

// WithSystemPrompt sets the system prompt for the agent
func WithSystemPrompt(prompt string) Option {
	return func(a *Agent) {
		a.systemPrompt = prompt
	}
}

// NewAgent creates a new agent with the given options
func NewAgent(options ...Option) (*Agent, error) {
	agent := &Agent{}

	for _, option := range options {
		option(agent)
	}

	// Validate required fields
	if agent.llm == nil {
		return nil, fmt.Errorf("LLM is required")
	}

	return agent, nil
}

// Run runs the agent with the given input
func (a *Agent) Run(ctx context.Context, input string) (string, error) {
	// If orgID is set on the agent, add it to the context
	if a.orgID != "" {
		ctx = multitenancy.WithOrgID(ctx, a.orgID)
	}

	// Start tracing if available
	var span interfaces.Span
	if a.tracer != nil {
		ctx, span = a.tracer.StartSpan(ctx, "agent.Run")
		defer span.End()
	}

	// Add user message to memory
	if a.memory != nil {
		if err := a.memory.AddMessage(ctx, interfaces.Message{
			Role:    "user",
			Content: input,
		}); err != nil {
			return "", fmt.Errorf("failed to add user message to memory: %w", err)
		}
	}

	// Apply guardrails to input if available
	if a.guardrails != nil {
		guardedInput, err := a.guardrails.ProcessInput(ctx, input)
		if err != nil {
			return "", fmt.Errorf("guardrails error: %w", err)
		}
		input = guardedInput
	}

	// Get conversation history if memory is available
	var prompt string
	if a.memory != nil {
		history, err := a.memory.GetMessages(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get conversation history: %w", err)
		}

		// Format history into prompt
		prompt = formatHistoryIntoPrompt(history)
	} else {
		prompt = input
	}

	// Generate response with tools if available
	var response string
	var err error
	if len(a.tools) > 0 {
		response, err = a.llm.GenerateWithTools(ctx, prompt, a.tools)
	} else {
		response, err = a.llm.Generate(ctx, prompt)
	}

	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// Apply guardrails to output if available
	if a.guardrails != nil {
		guardedResponse, err := a.guardrails.ProcessOutput(ctx, response)
		if err != nil {
			return "", fmt.Errorf("guardrails error: %w", err)
		}
		response = guardedResponse
	}

	// Add agent message to memory
	if a.memory != nil {
		if err := a.memory.AddMessage(ctx, interfaces.Message{
			Role:    "assistant",
			Content: response,
		}); err != nil {
			return "", fmt.Errorf("failed to add agent message to memory: %w", err)
		}
	}

	return response, nil
}

// formatHistoryIntoPrompt formats conversation history into a prompt
func formatHistoryIntoPrompt(history []interfaces.Message) string {
	// Implementation depends on the LLM's expected format
	var prompt string

	// Simple implementation that concatenates messages
	for _, msg := range history {
		role := msg.Role
		content := msg.Content

		prompt += role + ": " + content + "\n"
	}

	return prompt
}
