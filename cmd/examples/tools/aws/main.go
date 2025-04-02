package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/config"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/multitenancy"
	"github.com/Ingenimax/agent-sdk-go/pkg/structuredoutput"
	toolsregistry "github.com/Ingenimax/agent-sdk-go/pkg/tools"
	awstool "github.com/Ingenimax/agent-sdk-go/pkg/tools/aws"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type AWSResponse struct {
	UserResponse string `json:"user_response" description:"The response to the user's query"`
	BucketCount  int    `json:"bucket_count" description:"The number of S3 buckets found"`
}

func main() {
	// Create a logger
	logger := logging.New()

	// Get configuration
	cfg := config.Get()

	// Create AWS configuration
	awsCfg := aws.Config{
		Region: "us-east-1",
		Credentials: credentials.NewStaticCredentialsProvider(
			"aaaaa",
			"bbbbb", ""),
	}

	// Create a new agent with OpenAI
	openaiClient := openai.NewClient(cfg.LLM.OpenAI.APIKey,
		openai.WithLogger(logger))

	// Create tools registry
	toolRegistry := toolsregistry.NewRegistry()

	// Add AWS tool - view only mode
	awsTool, err := awstool.New(
		awstool.WithConfig(awsCfg),
		awstool.WithViewOnly(true),
	)

	if err != nil {
		logger.Error(context.Background(), "Failed to create AWS tool", map[string]interface{}{"error": err.Error()})
		return
	}
	toolRegistry.Register(awsTool)

	responseFormat := structuredoutput.NewResponseFormat(AWSResponse{})

	// Create the agent with the tools
	agent, err := agent.NewAgent(
		agent.WithLLM(openaiClient),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithTools(toolRegistry.List()...),
		agent.WithRequirePlanApproval(false),
		agent.WithSystemPrompt(`
		You are a helpful AI assistant that can help users manage their AWS resources.
		When users ask about AWS resources, use the AWS tool to find relevant information.
		You should always use the AWS tool to interact with AWS services.
		`),
		agent.WithName("AWSAssistant"),
		agent.WithResponseFormat(*responseFormat),
	)
	if err != nil {
		logger.Error(context.Background(), "Failed to create agent", map[string]interface{}{"error": err.Error()})
		return
	}

	// Log created agent
	logger.Info(context.Background(), "Created agent with tools", map[string]interface{}{"tools": toolRegistry.List()})

	// Create a context with organization ID and conversation ID
	ctx := context.Background()
	ctx = multitenancy.WithOrgID(ctx, "default-org")
	ctx = context.WithValue(ctx, memory.ConversationIDKey, "conversation-123")

	// Example queries to test the agent
	queries := []string{
		"List all my S3 buckets",
		"What are the names of my S3 buckets?",
	}

	// Run the agent with each query
	for _, query := range queries {
		fmt.Printf("\nQuery: %s\n", query)
		response, err := agent.Run(ctx, query)
		if err != nil {
			logger.Error(ctx, "Failed to run agent", map[string]interface{}{"error": err.Error()})
			continue
		}

		var awsResponse AWSResponse
		err = json.Unmarshal([]byte(response), &awsResponse)
		if err != nil {
			logger.Error(ctx, "Failed to unmarshal response", map[string]interface{}{"error": err.Error()})
			continue
		}

		// Print markdown response with line breaks and formatting
		lines := strings.Split(awsResponse.UserResponse, "\n")
		for _, line := range lines {
			fmt.Printf("Response: %s\n", strings.TrimSpace(line))
		}
		fmt.Printf("Number of buckets: %d\n", awsResponse.BucketCount)
	}
}
