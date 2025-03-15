package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Ingenimax/agent-sdk-go/pkg/llm"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
)

func main() {
	// Create a logger
	logger := logging.New()
	ctx := context.Background()

	// Get API key from environment
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Error(ctx, "OPENAI_API_KEY environment variable is required", nil)
		os.Exit(1)
	}

	// Create client
	client := openai.NewClient(
		apiKey,
		openai.WithModel("gpt-4o-mini"),
		openai.WithLogger(logger),
	)

	// Test text generation
	resp, err := client.Generate(
		ctx,
		"Write a haiku about programming",
		openai.WithTemperature(0.7),
		openai.WithMaxTokens(50),
	)
	if err != nil {
		logger.Error(ctx, "Failed to generate", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
	fmt.Printf("Generated text: %s\n\n", resp)

	// Test chat
	messages := []llm.Message{
		{
			Role:    "system",
			Content: "You are a helpful programming assistant.",
		},
		{
			Role:    "user",
			Content: "What's the best way to handle errors in Go?",
		},
	}

	resp, err = client.Chat(ctx, messages, nil)
	if err != nil {
		logger.Error(ctx, "Failed to chat", map[string]interface{}{"error": err.Error()})
		os.Exit(1)
	}
	fmt.Printf("Chat response: %s\n", resp)
}
