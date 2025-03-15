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

func main() {
	// Check for required API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set.")
		fmt.Println("Please set it with: export OPENAI_API_KEY=your_openai_api_key")
		os.Exit(1)
	}

	// Create LLM client
	openaiClient := openai.NewClient(apiKey)

	// Test the API key with a simple query
	fmt.Println("Testing OpenAI API key...")
	_, err := openaiClient.Generate(context.Background(), "Hello")
	if err != nil {
		fmt.Printf("Error: Failed to validate OpenAI API key: %v\n", err)
		fmt.Println("Please check that your API key is valid and has sufficient quota.")
		os.Exit(1)
	}
	fmt.Println("API key is valid!")

	// Create agent registry
	registry := orchestration.NewAgentRegistry()

	// Create general agent
	generalAgent, err := createGeneralAgent(openaiClient)
	if err != nil {
		fmt.Printf("Failed to create general agent: %v\n", err)
		os.Exit(1)
	}
	registry.Register("general", generalAgent)

	// Create research agent
	researchAgent, err := createResearchAgent(openaiClient)
	if err != nil {
		fmt.Printf("Failed to create research agent: %v\n", err)
		os.Exit(1)
	}
	registry.Register("research", researchAgent)

	// Create math agent
	mathAgent, err := createMathAgent(openaiClient)
	if err != nil {
		fmt.Printf("Failed to create math agent: %v\n", err)
		os.Exit(1)
	}
	registry.Register("math", mathAgent)

	// Create router
	router := orchestration.NewLLMRouter(openaiClient)

	// Create orchestrator
	orchestrator := orchestration.NewOrchestrator(registry, router)

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Add required IDs to context
	ctx = multitenancy.WithOrgID(ctx, "default-org")
	ctx = context.WithValue(ctx, memory.ConversationIDKey, "default-conversation")
	ctx = context.WithValue(ctx, userIDKey{}, "default-user")

	// Handle user queries
	for {
		// Get user input
		fmt.Print("\nEnter your query (or 'exit' to quit): ")
		var query string
		// Use bufio.NewReader to read the entire line including spaces
		reader := bufio.NewReader(os.Stdin)
		query, _ = reader.ReadString('\n')
		query = strings.TrimSpace(query)

		if query == "exit" {
			break
		}

		// Prepare context for routing
		routingContext := map[string]interface{}{
			"agents": map[string]string{
				"general":  "General-purpose assistant for everyday questions and tasks",
				"research": "Specialized in research, fact-finding, and information retrieval",
				"math":     "Specialized in mathematical calculations and problem-solving",
			},
		}

		// Handle the request
		fmt.Println("\nProcessing your request...")
		result, err := orchestrator.HandleRequest(ctx, query, routingContext)
		if err != nil {
			fmt.Printf("Error: %v\n", err)

			// Check for common error types and provide helpful messages
			errStr := err.Error()
			if strings.Contains(errStr, "401 Unauthorized") {
				fmt.Println("\nAPI key error detected. Please check that:")
				fmt.Println("1. Your OpenAI API key is correctly set in the environment")
				fmt.Println("2. The API key is valid and not expired")
				fmt.Println("3. Your account has sufficient credits")
				fmt.Println("\nYou can verify your API key with: echo $OPENAI_API_KEY")
			} else if strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "timeout") {
				fmt.Println("\nThe request timed out. This could be due to:")
				fmt.Println("1. OpenAI API service being slow or unavailable")
				fmt.Println("2. A complex query requiring too much processing time")
				fmt.Println("3. Network connectivity issues")
			}

			continue
		}

		// Print the result
		fmt.Printf("\nAgent: %s\n", result.AgentID)
		fmt.Printf("Response: %s\n", result.Response)
	}
}

func createGeneralAgent(llm interfaces.LLM) (*agent.Agent, error) {
	// Create memory
	mem := memory.NewConversationBuffer()

	// Create agent
	agent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(mem),
		agent.WithSystemPrompt(`You are a helpful general-purpose assistant. You can answer questions on a wide range of topics.
If you encounter a question that requires specialized knowledge in research or mathematics, you should hand off to a specialized agent.

To hand off to the research agent, respond with: [HANDOFF:research:needs specialized research]
To hand off to the math agent, respond with: [HANDOFF:math:needs mathematical calculation]

Otherwise, provide helpful and accurate responses to the user's questions.`),
	)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func createResearchAgent(llm interfaces.LLM) (*agent.Agent, error) {
	// Create memory
	mem := memory.NewConversationBuffer()

	// Create tools
	toolRegistry := tools.NewRegistry()
	searchTool := websearch.New(
		os.Getenv("GOOGLE_API_KEY"),
		os.Getenv("GOOGLE_SEARCH_ENGINE_ID"),
	)
	toolRegistry.Register(searchTool)

	// Create agent
	agent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(mem),
		agent.WithTools(toolRegistry.List()...),
		agent.WithSystemPrompt(`You are a specialized research agent. You excel at finding information and answering factual questions.
You have access to search tools to help you find information.

If you encounter a question that requires mathematical calculation, you should hand off to the math agent.
To hand off to the math agent, respond with: [HANDOFF:math:needs mathematical calculation]

Otherwise, use your tools to research and provide accurate information to the user.`),
	)
	if err != nil {
		return nil, err
	}

	return agent, nil
}

func createMathAgent(llm interfaces.LLM) (*agent.Agent, error) {
	// Create memory
	mem := memory.NewConversationBuffer()

	// Create tools
	toolRegistry := tools.NewRegistry()
	calcTool := calculator.NewCalculator()
	toolRegistry.Register(calcTool)

	// Create agent
	agent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(mem),
		agent.WithTools(toolRegistry.List()...),
		agent.WithSystemPrompt(`You are a specialized math agent. You excel at solving mathematical problems and performing calculations.
You have access to a calculator tool to help you solve complex problems.

If you encounter a question that requires research or factual information, you should hand off to the research agent.
To hand off to the research agent, respond with: [HANDOFF:research:needs specialized research]

Otherwise, use your mathematical expertise and tools to solve problems for the user.`),
	)
	if err != nil {
		return nil, err
	}

	return agent, nil
}
