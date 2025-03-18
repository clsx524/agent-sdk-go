package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/google/uuid"
)

// ExecutionPlanStatus represents the status of an execution plan
type ExecutionPlanStatus string

const (
	// StatusDraft indicates the plan is in draft state
	StatusDraft ExecutionPlanStatus = "draft"
	// StatusPendingApproval indicates the plan is waiting for user approval
	StatusPendingApproval ExecutionPlanStatus = "pending_approval"
	// StatusApproved indicates the plan has been approved
	StatusApproved ExecutionPlanStatus = "approved"
	// StatusExecuting indicates the plan is currently executing
	StatusExecuting ExecutionPlanStatus = "executing"
	// StatusCompleted indicates the plan has completed execution
	StatusCompleted ExecutionPlanStatus = "completed"
	// StatusFailed indicates the plan execution failed
	StatusFailed ExecutionPlanStatus = "failed"
	// StatusCancelled indicates the plan was cancelled
	StatusCancelled ExecutionPlanStatus = "cancelled"
)

// ExecutionPlan represents a plan of tool executions that the agent intends to perform
type ExecutionPlan struct {
	// Steps is a list of planned tool executions
	Steps []ExecutionStep
	// Description is a high-level description of what the plan will accomplish
	Description string
	// UserApproved indicates whether the user has approved the plan
	UserApproved bool
	// TaskID is a unique identifier for the task associated with this plan
	TaskID string
	// Status represents the current status of the execution plan
	Status ExecutionPlanStatus
	// CreatedAt is the time when the plan was created
	CreatedAt time.Time
	// UpdatedAt is the time when the plan was last updated
	UpdatedAt time.Time
}

// ExecutionStep represents a single step in an execution plan
type ExecutionStep struct {
	// ToolName is the name of the tool to execute
	ToolName string
	// Input is the input to provide to the tool
	Input string
	// Description is a description of what this step will accomplish
	Description string
	// Parameters contains the parameters for the tool execution
	Parameters map[string]interface{}
}

// NewExecutionPlan creates a new execution plan
func NewExecutionPlan(description string, steps []ExecutionStep) *ExecutionPlan {
	now := time.Now()
	return &ExecutionPlan{
		Description:  description,
		Steps:        steps,
		UserApproved: false,
		TaskID:       uuid.New().String(),
		Status:       StatusDraft,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// GenerateExecutionPlan generates an execution plan based on the user input
func (a *Agent) GenerateExecutionPlan(ctx context.Context, input string) (*ExecutionPlan, error) {
	// If no tools are available, return an error
	if len(a.tools) == 0 {
		return nil, fmt.Errorf("no tools available for execution planning")
	}

	// Create a prompt for the LLM to generate an execution plan
	prompt := a.createExecutionPlanPrompt(input)

	// Add system prompt as a generate option
	generateOptions := []interfaces.GenerateOption{}
	if a.systemPrompt != "" {
		generateOptions = append(generateOptions, openai.WithSystemMessage(a.systemPrompt))
	}

	// Generate the execution plan using the LLM
	response, err := a.llm.Generate(ctx, prompt, generateOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate execution plan: %w", err)
	}

	// Parse the execution plan from the LLM response
	plan, err := parseExecutionPlanFromResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse execution plan: %w", err)
	}

	// Update the status to pending approval
	plan.Status = StatusPendingApproval

	// Store the plan
	a.plansMutex.Lock()
	a.plans[plan.TaskID] = plan
	a.plansMutex.Unlock()

	return plan, nil
}

// createExecutionPlanPrompt creates a prompt for the LLM to generate an execution plan
func (a *Agent) createExecutionPlanPrompt(input string) string {
	// Build a list of available tools
	var toolDescriptions strings.Builder
	for _, tool := range a.tools {
		toolDescriptions.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))

		// Add parameter descriptions
		params := tool.Parameters()
		if len(params) > 0 {
			toolDescriptions.WriteString("  Parameters:\n")
			for name, spec := range params {
				required := ""
				if spec.Required {
					required = " (required)"
				}
				toolDescriptions.WriteString(fmt.Sprintf("  - %s: %s%s\n", name, spec.Description, required))
			}
		}
	}

	// Create the prompt
	prompt := fmt.Sprintf(`
You are an AI assistant that creates execution plans for tasks. Based on the user's request, create a detailed execution plan using the available tools.

Available tools:
%s

User request: %s

Create an execution plan in the following JSON format:
{
  "description": "High-level description of what the plan will accomplish",
  "steps": [
    {
      "toolName": "Name of the tool to use",
      "description": "Description of what this step will accomplish",
      "input": "Input to provide to the tool",
      "parameters": {
        "param1": "value1",
        "param2": "value2"
      }
    }
  ]
}

Ensure that:
1. Each step uses a valid tool from the list of available tools
2. All required parameters for each tool are provided
3. The plan is comprehensive and addresses all aspects of the user's request
4. The plan is presented in valid JSON format

Execution Plan:
`, toolDescriptions.String(), input)

	return prompt
}

// parseExecutionPlanFromResponse parses an execution plan from the LLM response
func parseExecutionPlanFromResponse(response string) (*ExecutionPlan, error) {
	// Extract the JSON part of the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	if jsonStart == -1 || jsonEnd == -1 || jsonEnd < jsonStart {
		return nil, fmt.Errorf("could not find valid JSON in response")
	}

	jsonStr := response[jsonStart : jsonEnd+1]

	// Parse the JSON
	var planData struct {
		Description string `json:"description"`
		Steps       []struct {
			ToolName    string                 `json:"toolName"`
			Description string                 `json:"description"`
			Input       string                 `json:"input"`
			Parameters  map[string]interface{} `json:"parameters"`
		} `json:"steps"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &planData); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan JSON: %w", err)
	}

	// Convert to ExecutionPlan
	steps := make([]ExecutionStep, len(planData.Steps))
	for i, step := range planData.Steps {
		steps[i] = ExecutionStep{
			ToolName:    step.ToolName,
			Description: step.Description,
			Input:       step.Input,
			Parameters:  step.Parameters,
		}
	}

	return NewExecutionPlan(planData.Description, steps), nil
}

// FormatExecutionPlan formats an execution plan for display to the user
func FormatExecutionPlan(plan *ExecutionPlan) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Execution Plan: %s\n\n", plan.Description))
	sb.WriteString(fmt.Sprintf("Task ID: %s\n", plan.TaskID))
	sb.WriteString(fmt.Sprintf("Status: %s\n\n", plan.Status))

	for i, step := range plan.Steps {
		sb.WriteString(fmt.Sprintf("## Step %d: %s\n", i+1, step.Description))
		sb.WriteString(fmt.Sprintf("Tool: %s\n", step.ToolName))
		sb.WriteString(fmt.Sprintf("Input: %s\n", step.Input))

		if len(step.Parameters) > 0 {
			sb.WriteString("Parameters:\n")
			for name, value := range step.Parameters {
				sb.WriteString(fmt.Sprintf("- %s: %v\n", name, value))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// ExecutePlan executes an approved execution plan
func (a *Agent) ExecutePlan(ctx context.Context, plan *ExecutionPlan) (string, error) {
	if !plan.UserApproved {
		return "", fmt.Errorf("execution plan has not been approved by the user")
	}

	// Update status to executing
	plan.Status = StatusExecuting
	plan.UpdatedAt = time.Now()

	var results strings.Builder
	results.WriteString(fmt.Sprintf("Executing plan: %s\n\n", plan.Description))

	success := true
	for i, step := range plan.Steps {
		results.WriteString(fmt.Sprintf("Step %d: %s\n", i+1, step.Description))

		// Find the tool
		var tool interfaces.Tool
		for _, t := range a.tools {
			if t.Name() == step.ToolName {
				tool = t
				break
			}
		}

		if tool == nil {
			results.WriteString(fmt.Sprintf("Error: Tool '%s' not found\n\n", step.ToolName))
			success = false
			continue
		}

		// Execute the tool
		result, err := tool.Execute(ctx, step.Input)
		if err != nil {
			results.WriteString(fmt.Sprintf("Error: %v\n\n", err))
			success = false
			continue
		}

		results.WriteString(fmt.Sprintf("Result: %s\n\n", result))
	}

	// Update status based on execution result
	if success {
		plan.Status = StatusCompleted
	} else {
		plan.Status = StatusFailed
	}
	plan.UpdatedAt = time.Now()

	return results.String(), nil
}

// CancelPlan cancels an execution plan
func (a *Agent) CancelPlan(plan *ExecutionPlan) {
	plan.Status = StatusCancelled
	plan.UpdatedAt = time.Now()
}

// GetPlanStatus returns the status of an execution plan
func (a *Agent) GetPlanStatus(plan *ExecutionPlan) ExecutionPlanStatus {
	return plan.Status
}

// GetPlanByTaskID returns an execution plan by its task ID
func (a *Agent) GetPlanByTaskID(taskID string) (*ExecutionPlan, bool) {
	a.plansMutex.RLock()
	defer a.plansMutex.RUnlock()

	plan, exists := a.plans[taskID]
	return plan, exists
}
