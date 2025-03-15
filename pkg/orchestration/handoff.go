package orchestration

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// HandoffRequest represents a request to hand off to another agent
type HandoffRequest struct {
	// TargetAgentID is the ID of the agent to hand off to
	TargetAgentID string

	// Reason explains why the handoff is happening
	Reason string

	// Context contains additional context for the target agent
	Context map[string]interface{}

	// Query is the query to send to the target agent
	Query string

	// PreserveMemory indicates whether to copy memory to the target agent
	PreserveMemory bool
}

// HandoffResult represents the result of a handoff
type HandoffResult struct {
	// AgentID is the ID of the agent that handled the request
	AgentID string

	// Response is the response from the agent
	Response string

	// Completed indicates whether the task was completed
	Completed bool

	// NextHandoff is the next handoff request, if any
	NextHandoff *HandoffRequest
}

// AgentRegistry maintains a registry of available agents
type AgentRegistry struct {
	agents map[string]*agent.Agent
}

// NewAgentRegistry creates a new agent registry
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*agent.Agent),
	}
}

// Register registers an agent with the registry
func (r *AgentRegistry) Register(id string, agent *agent.Agent) {
	r.agents[id] = agent
}

// Get retrieves an agent from the registry
func (r *AgentRegistry) Get(id string) (*agent.Agent, bool) {
	agent, ok := r.agents[id]
	return agent, ok
}

// List returns all registered agents
func (r *AgentRegistry) List() map[string]*agent.Agent {
	return r.agents
}

// Orchestrator orchestrates handoffs between agents
type Orchestrator struct {
	registry *AgentRegistry
	router   Router
}

// Router determines which agent should handle a request
type Router interface {
	Route(ctx context.Context, query string, context map[string]interface{}) (string, error)
}

// SimpleRouter routes requests based on a simple keyword matching
type SimpleRouter struct {
	routes map[string][]string // maps keywords to agent IDs
}

// NewSimpleRouter creates a new simple router
func NewSimpleRouter() *SimpleRouter {
	return &SimpleRouter{
		routes: make(map[string][]string),
	}
}

// AddRoute adds a route to the router
func (r *SimpleRouter) AddRoute(keyword string, agentID string) {
	r.routes[keyword] = append(r.routes[keyword], agentID)
}

// Route determines which agent should handle a request
func (r *SimpleRouter) Route(ctx context.Context, query string, context map[string]interface{}) (string, error) {
	// Simple keyword matching
	for keyword, agentIDs := range r.routes {
		if contains(query, keyword) {
			// Return the first agent ID
			if len(agentIDs) > 0 {
				return agentIDs[0], nil
			}
		}
	}

	return "", fmt.Errorf("no agent found for query: %s", query)
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// LLMRouter uses an LLM to determine which agent should handle a request
type LLMRouter struct {
	llm interfaces.LLM
}

// NewLLMRouter creates a new LLM router
func NewLLMRouter(llm interfaces.LLM) *LLMRouter {
	return &LLMRouter{
		llm: llm,
	}
}

// Route determines which agent should handle a request
func (r *LLMRouter) Route(ctx context.Context, query string, context map[string]interface{}) (string, error) {
	// Create a prompt for the LLM
	prompt := fmt.Sprintf(`You are a router that determines which specialized agent should handle a user query.
Available agents:
%s

User query: %s

Respond with only the ID of the agent that should handle this query.`, formatAgents(context["agents"].(map[string]string)), query)

	// Generate a response
	response, err := r.llm.Generate(ctx, prompt, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate response: %w", err)
	}

	// Clean up the response
	response = strings.TrimSpace(response)

	// Validate the response
	if _, ok := context["agents"].(map[string]string)[response]; !ok {
		return "", fmt.Errorf("invalid agent ID: %s", response)
	}

	return response, nil
}

// formatAgents formats a map of agent IDs to descriptions
func formatAgents(agents map[string]string) string {
	var result strings.Builder
	for id, desc := range agents {
		result.WriteString(fmt.Sprintf("- %s: %s\n", id, desc))
	}
	return result.String()
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(registry *AgentRegistry, router Router) *Orchestrator {
	return &Orchestrator{
		registry: registry,
		router:   router,
	}
}

// HandleRequest handles a request, potentially routing it through multiple agents
func (o *Orchestrator) HandleRequest(ctx context.Context, query string, initialContext map[string]interface{}) (*HandoffResult, error) {
	// Determine which agent should handle the request
	agentID, err := o.router.Route(ctx, query, initialContext)
	if err != nil {
		return nil, fmt.Errorf("failed to route request: %w", err)
	}

	// Create initial handoff request
	handoffReq := &HandoffRequest{
		TargetAgentID:  agentID,
		Query:          query,
		Context:        initialContext,
		PreserveMemory: true,
	}

	// Process handoffs until completion or max iterations
	maxIterations := 5
	for i := 0; i < maxIterations; i++ {
		// Check if context is done
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			// Continue processing
		}

		// Process handoff
		result, err := o.processHandoff(ctx, handoffReq)
		if err != nil {
			return nil, fmt.Errorf("failed to process handoff: %w", err)
		}

		// Check if completed or no next handoff
		if result.Completed || result.NextHandoff == nil {
			return result, nil
		}

		// Prepare for next handoff
		handoffReq = result.NextHandoff
	}

	return nil, fmt.Errorf("exceeded maximum number of handoffs")
}

// processHandoff processes a single handoff
func (o *Orchestrator) processHandoff(ctx context.Context, req *HandoffRequest) (*HandoffResult, error) {
	// Get the target agent
	targetAgent, ok := o.registry.Get(req.TargetAgentID)
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", req.TargetAgentID)
	}

	// Create a new context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Run the agent
	response, err := targetAgent.Run(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Check for handoff request in the response
	nextHandoff := o.parseHandoffRequest(response)

	// Create result
	result := &HandoffResult{
		AgentID:     req.TargetAgentID,
		Response:    response,
		Completed:   nextHandoff == nil,
		NextHandoff: nextHandoff,
	}

	return result, nil
}

// parseHandoffRequest parses a handoff request from an agent's response
func (o *Orchestrator) parseHandoffRequest(response string) *HandoffRequest {
	// Look for a handoff marker in the response
	// Format: [HANDOFF:agent_id:reason]
	re := regexp.MustCompile(`\[HANDOFF:([a-zA-Z0-9_-]+):([^\]]+)\]`)
	matches := re.FindStringSubmatch(response)
	if len(matches) < 3 {
		return nil
	}

	// Extract handoff information
	agentID := matches[1]
	reason := matches[2]

	// Extract the query (everything after the handoff marker)
	query := response[len(matches[0]):]
	query = strings.TrimSpace(query)

	// Create handoff request
	return &HandoffRequest{
		TargetAgentID:  agentID,
		Reason:         reason,
		Query:          query,
		PreserveMemory: true,
		Context:        make(map[string]interface{}),
	}
}
