package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/Ingenimax/agent-sdk-go/pkg/config"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
)

func main() {
	// Create a logger
	logger := logging.New()

	cfg := config.Get()
	// Create context with organization ID and conversation ID
	ctx := context.Background()
	ctx = multitenancy.WithOrgID(ctx, "example-org")
	ctx = memory.WithConversationID(ctx, "conversation-123")

	// Example 1: Conversation Buffer Memory
	fmt.Println("=== Conversation Buffer Memory ===")
	bufferMemory := memory.NewConversationBuffer(
		memory.WithMaxSize(5),
	)
	testMemory(ctx, bufferMemory, logger)

	// Example 2: Conversation Summary Memory
	fmt.Println("\n=== Conversation Summary Memory ===")

	llmClient := openai.NewClient(cfg.LLM.OpenAI.APIKey,
		openai.WithModel(cfg.LLM.OpenAI.Model),
		openai.WithLogger(logger),
	)

	summaryMemory := memory.NewConversationSummary(
		llmClient,
		memory.WithMaxBufferSize(3),
	)
	testMemory(ctx, summaryMemory, logger)

	// Example 3: Vector Store Retriever Memory
	fmt.Println("\n=== Vector Store Retriever Memory ===")
	vectorStore, err := setupVectorStore(logger)
	if err != nil {
		logger.Info(ctx, "Skipping vector store example", map[string]interface{}{"error": err.Error()})
	} else {
		retrieverMemory := memory.NewVectorStoreRetriever(vectorStore)
		testMemory(ctx, retrieverMemory, logger)
	}

	// Example 4: Redis Memory
	fmt.Println("\n=== Redis Memory ===")
	redisClient, err := setupRedisClient()
	if err != nil {
		logger.Info(ctx, "Skipping Redis example", map[string]interface{}{"error": err.Error()})
	} else {
		redisMemory := memory.NewRedisMemory(
			redisClient,
			memory.WithTTL(1*time.Hour),
		)
		testMemory(ctx, redisMemory, logger)

		// Close Redis client
		if err := redisClient.Close(); err != nil {
			logger.Error(ctx, "Error closing Redis client", map[string]interface{}{"error": err.Error()})
		}
	}
}

func testMemory(ctx context.Context, mem interfaces.Memory, logger logging.Logger) {
	// Add messages
	messages := []interfaces.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UnixNano(),
			},
		},
		{
			Role:    "user",
			Content: "Hello, how are you?",
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UnixNano(),
			},
		},
		{
			Role:    "assistant",
			Content: "I'm doing well, thank you for asking! How can I help you today?",
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UnixNano(),
			},
		},
		{
			Role:    "user",
			Content: "Tell me about the weather.",
			Metadata: map[string]interface{}{
				"timestamp": time.Now().UnixNano(),
			},
		},
	}

	for _, msg := range messages {
		if err := mem.AddMessage(ctx, msg); err != nil {
			logger.Error(ctx, "Failed to add message", map[string]interface{}{"error": err.Error()})
			return
		}
	}

	// Get all messages
	allMessages, err := mem.GetMessages(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get messages", map[string]interface{}{"error": err.Error()})
		return
	}

	fmt.Println("All messages:")
	for i, msg := range allMessages {
		fmt.Printf("%d. %s: %s\n", i+1, msg.Role, msg.Content)
	}

	// Get user messages only
	userMessages, err := mem.GetMessages(ctx, interfaces.WithRoles("user"))
	if err != nil {
		logger.Error(ctx, "Failed to get user messages", map[string]interface{}{"error": err.Error()})
		return
	}

	fmt.Println("\nUser messages only:")
	for i, msg := range userMessages {
		fmt.Printf("%d. %s: %s\n", i+1, msg.Role, msg.Content)
	}

	// Get last 2 messages
	lastMessages, err := mem.GetMessages(ctx, interfaces.WithLimit(2))
	if err != nil {
		logger.Error(ctx, "Failed to get last messages", map[string]interface{}{"error": err.Error()})
		return
	}

	fmt.Println("\nLast 2 messages:")
	for i, msg := range lastMessages {
		fmt.Printf("%d. %s: %s\n", i+1, msg.Role, msg.Content)
	}

	// Clear memory
	if err := mem.Clear(ctx); err != nil {
		logger.Error(ctx, "Failed to clear memory", map[string]interface{}{"error": err.Error()})
		return
	}

	// Verify memory is cleared
	clearedMessages, err := mem.GetMessages(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get messages after clearing", map[string]interface{}{"error": err.Error()})
		return
	}

	fmt.Println("\nAfter clearing:")
	if len(clearedMessages) == 0 {
		fmt.Println("Memory cleared successfully")
	} else {
		fmt.Printf("Memory not cleared, %d messages remaining\n", len(clearedMessages))
	}
}

func setupVectorStore(logger logging.Logger) (interfaces.VectorStore, error) {
	// Check if we have the necessary environment variables
	// This is a placeholder - in a real application, you would
	// configure and return a real vector store

	// For example, to use a simple in-memory vector store:
	// return vectorstore.NewInMemory(), nil

	return nil, fmt.Errorf("vector store setup not implemented - skipping example")
}

func setupRedisClient() (*redis.Client, error) {
	// Check if Redis is available
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default Redis address
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test connection
	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis at %s: %w", redisAddr, err)
	}

	return client, nil
}
