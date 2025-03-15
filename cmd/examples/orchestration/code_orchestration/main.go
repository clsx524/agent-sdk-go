package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/orchestration"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/calculator"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/websearch"
)

// Define custom context key types to avoid using string literals
type userIDKey struct{}

// CustomCodeOrchestrator extends the default CodeOrchestrator to fix issues with task dependencies
type CustomCodeOrchestrator struct {
	registry *orchestration.AgentRegistry
}

// NewCustomCodeOrchestrator creates a new custom code orchestrator
func NewCustomCodeOrchestrator(registry *orchestration.AgentRegistry) *CustomCodeOrchestrator {
	return &CustomCodeOrchestrator{
		registry: registry,
	}
}

// ExecuteWorkflow executes a workflow with enhanced dependency handling
func (o *CustomCodeOrchestrator) ExecuteWorkflow(ctx context.Context, workflow *orchestration.Workflow) (string, error) {
	fmt.Println("DEBUG - Using custom workflow executor with enhanced dependency handling")

	// Execute tasks in order (no parallelism for simplicity)
	for _, task := range workflow.Tasks {
		fmt.Printf("DEBUG - Executing task: %s (Agent: %s)\n", task.ID, task.AgentID)

		// Get the agent
		agent, ok := o.registry.Get(task.AgentID)
		if !ok {
			err := fmt.Errorf("agent not found: %s", task.AgentID)
			workflow.Errors[task.ID] = err
			fmt.Printf("DEBUG - Error: %v\n", err)
			continue
		}

		// Prepare input with results from dependencies
		input := task.Input
		for _, depID := range task.Dependencies {
			if result, ok := workflow.Results[depID]; ok {
				// Format the dependency result clearly
				input = fmt.Sprintf("%s\n\n===== Result from %s =====\n%s\n=====\n",
					input, depID, result)
				fmt.Printf("DEBUG - Added dependency result from %s (length: %d)\n",
					depID, len(result))
			}
		}

		// Execute the agent
		fmt.Printf("DEBUG - Running agent with input (length: %d)\n", len(input))
		result, err := agent.Run(ctx, input)
		if err != nil {
			workflow.Errors[task.ID] = err
			fmt.Printf("DEBUG - Agent execution failed: %v\n", err)
			continue
		}

		// Store the result
		workflow.Results[task.ID] = result
		fmt.Printf("DEBUG - Task completed with result (length: %d)\n", len(result))
	}

	// Return the final result
	if workflow.FinalTaskID != "" {
		if result, ok := workflow.Results[workflow.FinalTaskID]; ok && result != "" {
			return result, nil
		}
		return "", fmt.Errorf("final task result not found")
	}

	return "", nil
}

func main() {
	// Check for required API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" || apiKey == "your_openai_api_key" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set or is invalid.")
		fmt.Println("Please set a valid OpenAI API key to continue.")
		os.Exit(1)
	}

	// Create LLM client with explicit API key
	openaiClient := openai.NewClient(apiKey)

	// Test the API key with a simple query
	fmt.Println("Testing OpenAI API key...")
	_, err := openaiClient.Generate(context.Background(), "Hello")
	if err != nil {
		fmt.Printf("Error: Failed to validate OpenAI API key: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("API key is valid!")

	// Create agent registry
	registry := orchestration.NewAgentRegistry()

	// Create specialized agents
	createAndRegisterAgents(registry, openaiClient)

	// Debug: Print registered agents
	fmt.Println("Registered agents:")
	for id := range registry.List() {
		fmt.Printf("- %s\n", id)
	}

	// Create code orchestrator for workflow execution
	codeOrchestrator := NewCustomCodeOrchestrator(registry)

	// Create context with a long timeout for the entire program
	baseCtx, baseCancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer baseCancel()

	// Add required IDs to the base context
	baseCtx = multitenancy.WithOrgID(baseCtx, "default-org")
	baseCtx = context.WithValue(baseCtx, memory.ConversationIDKey, "default-conversation")
	baseCtx = context.WithValue(baseCtx, userIDKey{}, "default-user")

	// Handle user queries
	for {
		// Get user input
		fmt.Print("\nEnter your query (or 'exit' to quit): ")
		var query string
		reader := bufio.NewReader(os.Stdin)
		query, _ = reader.ReadString('\n')
		query = strings.TrimSpace(query)

		if query == "exit" {
			break
		}

		// Create a context for this specific query
		queryCtx, queryCancel := context.WithTimeout(baseCtx, 5*time.Minute)
		defer queryCancel()

		// Use Workflow Orchestration by default
		fmt.Println("\nUsing Workflow Orchestration...")

		// Create a workflow for this query
		workflow := createWorkflow(query)

		// Print workflow details
		fmt.Println("Workflow details:")
		fmt.Printf("Final task ID: %s\n", workflow.FinalTaskID)
		fmt.Println("Tasks:")
		for _, task := range workflow.Tasks {
			fmt.Printf("- ID: %s, Agent: %s, Dependencies: %v\n",
				task.ID, task.AgentID, task.Dependencies)
		}

		fmt.Println("\nProcessing your request...")
		fmt.Println("Starting workflow execution...")

		// Execute workflow in a goroutine with timeout
		resultChan := make(chan struct {
			response string
			err      error
		})

		go func() {
			fmt.Println("DEBUG - Starting workflow execution in goroutine...")
			response, err := codeOrchestrator.ExecuteWorkflow(queryCtx, workflow)
			fmt.Printf("DEBUG - Workflow execution completed with response length: %d, error: %v\n",
				len(response), err)
			resultChan <- struct {
				response string
				err      error
			}{response, err}
		}()

		// Wait for result or timeout
		var result struct {
			response string
			err      error
		}

		select {
		case result = <-resultChan:
			// Result received
		case <-time.After(4 * time.Minute):
			result.err = fmt.Errorf("workflow execution timed out")
		}

		fmt.Println("\nWorkflow execution completed.")

		// Print results
		fmt.Println("Results:")
		hasResults := false
		for taskID, taskResult := range workflow.Results {
			if taskResult != "" {
				hasResults = true
				fmt.Printf("- Task %s completed with result (%d chars)\n  Preview: %s\n",
					taskID, len(taskResult), truncateString(taskResult, 70))
			}
		}

		// Add detailed debugging information
		debugWorkflowExecution(workflow)

		if result.err != nil {
			fmt.Printf("Error: %v\n", result.err)
			printWorkflowErrors(workflow)

			// If we have a "final task result not found" error but the final task's dependency has a result,
			// try to use that result as the final response
			if strings.Contains(result.err.Error(), "final task result not found") && hasResults {
				fmt.Println("\nAttempting to recover result from dependencies...")

				// Find the final task
				var finalTask *orchestration.Task
				for _, task := range workflow.Tasks {
					if task.ID == workflow.FinalTaskID {
						finalTask = task
						break
					}
				}

				// If the final task has dependencies with results, use the first one
				if finalTask != nil && len(finalTask.Dependencies) > 0 {
					depID := finalTask.Dependencies[0]
					if depResult, ok := workflow.Results[depID]; ok && depResult != "" {
						fmt.Printf("\nUsing result from dependency '%s' as final response:\n%s\n",
							depID, depResult)
						continue
					}
				}
			}

			continue
		}

		fmt.Printf("\nResponse (took %.2f seconds):\n%s\n",
			time.Since(time.Now().Add(-5*time.Minute)).Seconds(), result.response)
	}
}

func createAndRegisterAgents(registry *orchestration.AgentRegistry, llm interfaces.LLM) {
	// Create memory with proper initialization
	researchMem := memory.NewConversationBuffer()
	mathMem := memory.NewConversationBuffer()
	creativeMem := memory.NewConversationBuffer()
	summaryMem := memory.NewConversationBuffer()

	// Create research agent with handoff capability
	researchAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(researchMem),
		agent.WithTools(createResearchTools().List()...),
		agent.WithSystemPrompt(`You are a research agent specialized in finding and summarizing information.
You excel at answering factual questions and providing up-to-date information.

When you've completed your research, you should hand off to the summary agent.
To hand off to the summary agent, respond with:
[HANDOFF:summary:Here are my research findings: {your detailed research}]

Make sure your research is thorough and includes at least 5 key points before handing off.`),
		agent.WithOrgID("default-org"),
	)
	if err != nil {
		fmt.Printf("Warning: Error creating research agent: %v\n", err)
	}
	registry.Register("research", researchAgent)

	// Create math agent with error handling and explicit org ID
	mathAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(mathMem),
		agent.WithTools(createMathTools().List()...),
		agent.WithSystemPrompt("You are a math agent specialized in solving mathematical problems. You excel at calculations, equations, and numerical analysis."),
		agent.WithOrgID("default-org"),
	)
	if err != nil {
		fmt.Printf("Warning: Error creating math agent: %v\n", err)
	}
	registry.Register("math", mathAgent)

	// Create creative agent with error handling and explicit org ID
	creativeAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(creativeMem),
		agent.WithSystemPrompt("You are a creative agent specialized in generating creative content. You excel at writing, storytelling, and creative problem-solving."),
		agent.WithOrgID("default-org"),
	)
	if err != nil {
		fmt.Printf("Warning: Error creating creative agent: %v\n", err)
	}
	registry.Register("creative", creativeAgent)

	// Create summary agent that receives handoffs
	summaryAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(summaryMem),
		agent.WithSystemPrompt(`You are a summarization agent specialized in creating concise summaries.
You will receive input that includes research findings in the format: "Result from research: [research content]"

Your task is to extract the research content and create a well-structured summary.
Your summary should include:
1) A brief explanation of the topic
2) Key points (3-5 bullet points)
3) A conclusion

Format your response as a complete summary suitable for someone with basic technical knowledge.
Do NOT mention that you're a summary agent or that you received input from another source.

IMPORTANT: You MUST always produce a summary response, even if the input is complex or lengthy.`),
		agent.WithOrgID("default-org"),
	)
	if err != nil {
		fmt.Printf("Warning: Error creating summary agent: %v\n", err)
	}
	registry.Register("summary", summaryAgent)
}

func createResearchTools() *tools.Registry {
	toolRegistry := tools.NewRegistry()

	// Add web search tool
	searchTool := websearch.New(
		os.Getenv("GOOGLE_API_KEY"),
		os.Getenv("GOOGLE_SEARCH_ENGINE_ID"),
	)
	toolRegistry.Register(searchTool)

	return toolRegistry
}

func createMathTools() *tools.Registry {
	toolRegistry := tools.NewRegistry()

	// Add calculator tool
	calcTool := calculator.NewCalculator()
	toolRegistry.Register(calcTool)

	return toolRegistry
}

func createWorkflow(query string) *orchestration.Workflow {
	workflow := orchestration.NewWorkflow()

	// Extract math expression if present
	expression := extractMathExpression(query)

	if expression != "" && isValidMathExpression(expression) {
		// Math query
		workflow.AddTask("math", "math", expression, []string{})
		workflow.AddTask("summary", "summary", "Provide the numerical answer to this calculation.", []string{"math"})
		workflow.SetFinalTask("summary")
	} else {
		// Research query
		workflow.AddTask("research", "research", query, []string{})

		// Simplified prompt for the summary task
		summaryPrompt := "Create a concise summary of the information provided."

		workflow.AddTask("summary", "summary", summaryPrompt, []string{"research"})
		workflow.SetFinalTask("summary")
	}

	return workflow
}

// Helper function to extract mathematical expressions
func extractMathExpression(query string) string {
	// Simple extraction: find the first digit and return the rest
	for i, char := range query {
		if char >= '0' && char <= '9' {
			return query[i:]
		}
	}
	return ""
}

// Helper function to validate mathematical expressions
func isValidMathExpression(expr string) bool {
	validChars := "0123456789+-*/() "
	for _, char := range expr {
		if !strings.ContainsRune(validChars, char) {
			return false
		}
	}
	return true
}

// Helper function to truncate strings
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Helper function to print workflow errors
func printWorkflowErrors(workflow *orchestration.Workflow) {
	fmt.Println("\nWorkflow Errors:")
	for taskID, err := range workflow.Errors {
		fmt.Printf("- Task %s: %v\n", taskID, err)
	}

	// Check for missing results
	if workflow.FinalTaskID != "" && workflow.Results[workflow.FinalTaskID] == "" {
		fmt.Printf("\nDEBUG - Final task '%s' has no result but no error was recorded\n", workflow.FinalTaskID)

		// Check if dependencies have results
		for _, task := range workflow.Tasks {
			if task.ID == workflow.FinalTaskID {
				for _, depID := range task.Dependencies {
					if result, ok := workflow.Results[depID]; ok {
						fmt.Printf("DEBUG - Dependency '%s' has result: %s\n", depID, truncateString(result, 100))
					} else {
						fmt.Printf("DEBUG - Dependency '%s' has no result\n", depID)
					}
				}
			}
		}
	}
}

// Add this function to the file to debug and fix task execution issues
func debugWorkflowExecution(workflow *orchestration.Workflow) {
	fmt.Println("\nDEBUG - Workflow Execution Details:")

	// Print all tasks and their status
	fmt.Println("Tasks Status:")
	for _, task := range workflow.Tasks {
		statusStr := string(task.Status)
		resultLen := 0
		if result, ok := workflow.Results[task.ID]; ok {
			resultLen = len(result)
		}

		fmt.Printf("- Task %s (Agent: %s): Status=%s, Result Length=%d\n",
			task.ID, task.AgentID, statusStr, resultLen)

		// Print dependencies
		if len(task.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %v\n", task.Dependencies)
			for _, depID := range task.Dependencies {
				if depResult, ok := workflow.Results[depID]; ok {
					fmt.Printf("  - Dependency %s has result of length %d\n", depID, len(depResult))
					if len(depResult) > 0 {
						fmt.Printf("    Preview: %s\n", truncateString(depResult, 50))
					}
				} else {
					fmt.Printf("  - Dependency %s has no result\n", depID)
				}
			}
		}
	}

	// Print final task info
	fmt.Printf("\nFinal Task: %s\n", workflow.FinalTaskID)
	if result, ok := workflow.Results[workflow.FinalTaskID]; ok {
		fmt.Printf("Final Task Result Length: %d\n", len(result))
		if len(result) > 0 {
			fmt.Printf("Final Task Result Preview: %s\n", truncateString(result, 50))
		}
	} else {
		fmt.Println("Final Task has no result")
	}
}
