package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/config"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/orchestration"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/calculator"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/websearch"
)

func main() {
	// Create a logger
	logger := logging.New()
	ctx := context.Background()

	logger.Info(ctx, "Starting LLM Orchestration example", nil)

	// Check if OPENAI_API_KEY is set
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Error(ctx, "OPENAI_API_KEY environment variable is not set", nil)
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
		fmt.Println("Please set it before running this example:")
		fmt.Println("export OPENAI_API_KEY=your_api_key")
		os.Exit(1)
	}
	logger.Info(ctx, "OPENAI_API_KEY is set", nil)

	// Force reload config to ensure it picks up the API key
	config.Reload()

	// Create OpenAI client
	logger.Info(ctx, "Creating OpenAI client", map[string]interface{}{"model": "gpt-4o-mini"})
	openaiClient := openai.NewClient(apiKey,
		openai.WithModel("gpt-4o-mini"),
		openai.WithLogger(logger),
	)

	// Create agent registry
	registry := orchestration.NewAgentRegistry()

	// Create specialized agents
	logger.Info(ctx, "Creating and registering specialized agents", nil)
	createAndRegisterAgents(registry, openaiClient, logger)

	// Create orchestrator
	orchestrator := orchestration.NewLLMOrchestrator(registry, openaiClient)
	orchestrator.WithLogger(logger)

	// Create context with a default organization ID and conversation ID
	logger.Info(ctx, "Setting up context with organization ID and conversation ID", nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	ctx = multitenancy.WithOrgID(ctx, "default")
	ctx = memory.WithConversationID(ctx, "example-conversation")
	defer cancel()

	// Handle user queries
	for {
		// Get user input
		fmt.Print("\nEnter your query (or 'exit' to quit): ")
		var query string
		// Read the entire line, not just the first word
		reader := bufio.NewReader(os.Stdin)
		query, err := reader.ReadString('\n')
		if err != nil {
			logger.Error(ctx, "Error reading input", map[string]interface{}{"error": err.Error()})
			continue
		}

		// Trim whitespace
		query = strings.TrimSpace(query)

		if query == "exit" {
			logger.Info(ctx, "User requested exit", nil)
			break
		}

		// Execute the query
		fmt.Println("\nProcessing your request...")
		logger.Info(ctx, "Processing query", map[string]interface{}{"query": query})
		startTime := time.Now()

		response, err := orchestrator.Execute(ctx, query)
		if err != nil {
			logger.Error(ctx, "Error executing query", map[string]interface{}{"error": err.Error()})
			fmt.Printf("Error: %v\n", err)
			continue
		}

		// Print the result
		duration := time.Since(startTime).Seconds()
		logger.Info(ctx, "Query processed successfully", map[string]interface{}{"duration_seconds": duration})
		fmt.Printf("\nResponse (took %.2f seconds):\n%s\n", duration, response)
	}
}

func createAndRegisterAgents(registry *orchestration.AgentRegistry, llm interfaces.LLM, logger logging.Logger) {
	// Create research agent
	researchAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithTools(createResearchTools(logger)...),
		agent.WithSystemPrompt("You are a research agent specialized in finding and summarizing information. You excel at answering factual questions and providing up-to-date information."),
	)
	if err != nil {
		logger.Error(context.Background(), "Error creating research agent", map[string]interface{}{"error": err.Error()})
	} else {
		registry.Register("research", researchAgent)
		logger.Info(context.Background(), "Research agent registered", nil)
	}

	// Create math agent
	mathAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithTools(createMathTools()...),
		agent.WithSystemPrompt("You are a math agent specialized in solving mathematical problems. You excel at calculations, equations, and numerical analysis."),
	)
	if err != nil {
		logger.Error(context.Background(), "Error creating math agent", map[string]interface{}{"error": err.Error()})
	} else {
		registry.Register("math", mathAgent)
		logger.Info(context.Background(), "Math agent registered", nil)
	}

	// Create creative agent
	creativeAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithSystemPrompt("You are a creative agent specialized in generating creative content. You excel at writing, storytelling, and creative problem-solving."),
	)
	if err != nil {
		logger.Error(context.Background(), "Error creating creative agent", map[string]interface{}{"error": err.Error()})
	} else {
		registry.Register("creative", creativeAgent)
		logger.Info(context.Background(), "Creative agent registered", nil)
	}

	// Create summarization agent
	summaryAgent, err := agent.NewAgent(
		agent.WithLLM(llm),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithSystemPrompt("You are a summarization agent specialized in condensing information. You excel at extracting key points and creating concise summaries."),
	)
	if err != nil {
		logger.Error(context.Background(), "Error creating summarization agent", map[string]interface{}{"error": err.Error()})
	} else {
		registry.Register("summary", summaryAgent)
		logger.Info(context.Background(), "Summarization agent registered", nil)
	}
}

func createResearchTools(logger logging.Logger) []interfaces.Tool {
	// Add web search tool
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	googleSearchEngineID := os.Getenv("GOOGLE_SEARCH_ENGINE_ID")

	if googleAPIKey == "" || googleSearchEngineID == "" {
		logger.Warn(context.Background(), "GOOGLE_API_KEY or GOOGLE_SEARCH_ENGINE_ID not set, web search tool will not work properly", nil)
	}

	searchTool := websearch.New(googleAPIKey, googleSearchEngineID)
	return []interfaces.Tool{searchTool}
}

func createMathTools() []interfaces.Tool {
	// Add calculator tool
	calcTool := &calculator.Calculator{}
	return []interfaces.Tool{calcTool}
}
