package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ingenimax/agent-sdk-go/pkg/guardrails"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/websearch"
)

// SimpleLogger implements a simple logger for the guardrails
type SimpleLogger struct{}

func (l *SimpleLogger) Debug(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", message, fields)
}

func (l *SimpleLogger) Info(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Printf("[INFO] %s %v\n", message, fields)
}

func (l *SimpleLogger) Warn(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Printf("[WARN] %s %v\n", message, fields)
}

func (l *SimpleLogger) Error(ctx context.Context, message string, fields map[string]interface{}) {
	fmt.Printf("[ERROR] %s %v\n", message, fields)
}

func main() {
	// Create a logger
	logger := &SimpleLogger{}

	// Create guardrails
	contentFilter := guardrails.NewContentFilter(
		[]string{"hate", "violence", "profanity", "sexual"},
		guardrails.RedactAction,
	)

	tokenLimit := guardrails.NewTokenLimit(
		100,
		nil, // Use simple token counter
		guardrails.RedactAction,
		"end",
	)

	piiFilter := guardrails.NewPiiFilter(
		guardrails.RedactAction,
	)

	toolRestriction := guardrails.NewToolRestriction(
		[]string{"web_search", "calculator"},
		guardrails.BlockAction,
	)

	rateLimit := guardrails.NewRateLimit(
		10, // 10 requests per minute
		guardrails.BlockAction,
	)

	// Create a guardrails pipeline
	pipeline := guardrails.NewPipeline(
		[]guardrails.Guardrail{
			contentFilter,
			tokenLimit,
			piiFilter,
			toolRestriction,
			rateLimit,
		},
		logger,
	)

	// Create an LLM with guardrails
	openaiClient := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	llmWithGuardrails := guardrails.NewLLMMiddleware(openaiClient, pipeline)

	// Create a tool with guardrails
	tool := websearch.New(
		os.Getenv("GOOGLE_API_KEY"),
		os.Getenv("GOOGLE_SEARCH_ENGINE_ID"),
	)
	toolWithGuardrails := guardrails.NewToolMiddleware(tool, pipeline)

	// Test LLM with guardrails
	fmt.Println("=== Testing LLM with Guardrails ===")
	testLLM(llmWithGuardrails)

	// Test tool with guardrails
	fmt.Println("\n=== Testing Tool with Guardrails ===")
	testTool(toolWithGuardrails)
}

func testLLM(llm *guardrails.LLMMiddleware) {
	ctx := context.Background()

	// Test content filter
	prompt := "Tell me about violence and hate speech"
	fmt.Printf("Prompt: %s\n", prompt)
	response, err := llm.Generate(ctx, prompt, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response)
	}

	// Test PII filter
	prompt = "My email is john.doe@example.com and my phone number is 123-456-7890"
	fmt.Printf("\nPrompt: %s\n", prompt)
	response, err = llm.Generate(ctx, prompt, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response)
	}

	// Test token limit
	prompt = "Tell me a very long story about a programmer who is trying to implement guardrails for an AI system. The programmer is working day and night to make sure the AI system is safe and secure. The programmer is also trying to make sure the AI system is useful and helpful. The programmer is also trying to make sure the AI system is ethical and fair."
	fmt.Printf("\nPrompt: %s\n", prompt)
	response, err = llm.Generate(ctx, prompt, nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Response: %s\n", response)
	}
}

func testTool(tool *guardrails.ToolMiddleware) {
	ctx := context.Background()

	// Test content filter
	input := `{"query": "Tell me about violence and hate speech"}`
	fmt.Printf("Input: %s\n", input)
	output, err := tool.Run(ctx, input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Output: %s\n", output)
	}

	// Test tool restriction
	input = `Use tool aws_s3 to list all buckets`
	fmt.Printf("\nInput: %s\n", input)
	output, err = tool.Run(ctx, input)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Output: %s\n", output)
	}

	// Test rate limit (run multiple requests)
	fmt.Println("\nTesting rate limit...")
	for i := 0; i < 15; i++ {
		input = fmt.Sprintf(`{"query": "Test query %d"}`, i)
		_, err := tool.Run(ctx, input)
		if err != nil {
			fmt.Printf("Request %d: Error: %v\n", i, err)
		} else {
			fmt.Printf("Request %d: Success\n", i)
		}
	}
}
