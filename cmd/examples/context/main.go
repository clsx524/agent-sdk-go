package main

import (
	"fmt"
	"time"

	pkgcontext "github.com/Ingenimax/agent-sdk-go/pkg/context"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools"
	"github.com/Ingenimax/agent-sdk-go/pkg/tools/websearch"
)

func main() {
	// Create a new context
	ctx := pkgcontext.New()

	// Set organization and conversation IDs
	ctx = ctx.WithOrganizationID("example-org")
	ctx = ctx.WithConversationID("example-conversation")
	ctx = ctx.WithUserID("example-user")
	ctx = ctx.WithRequestID("example-request")

	// Add memory
	memory := memory.NewConversationBuffer()
	ctx = ctx.WithMemory(memory)

	// Add tools
	toolRegistry := tools.NewRegistry()
	searchTool := websearch.New("api-key", "engine-id")
	toolRegistry.Register(searchTool)
	ctx = ctx.WithTools(toolRegistry)

	// Add LLM
	openaiClient := openai.NewClient("api-key")
	ctx = ctx.WithLLM(openaiClient)

	// Add environment variables
	ctx = ctx.WithEnvironment("temperature", 0.7)
	ctx = ctx.WithEnvironment("max_tokens", 1000)

	// Use the context
	fmt.Println("Context created with:")
	orgID, _ := ctx.OrganizationID()
	fmt.Printf("- Organization ID: %s\n", orgID)
	conversationID, _ := ctx.ConversationID()
	fmt.Printf("- Conversation ID: %s\n", conversationID)
	userID, _ := ctx.UserID()
	fmt.Printf("- User ID: %s\n", userID)
	requestID, _ := ctx.RequestID()
	fmt.Printf("- Request ID: %s\n", requestID)

	// Create a context with timeout
	ctxWithTimeout, cancel := ctx.WithTimeout(5 * time.Second)
	defer cancel()

	// Simulate a long-running operation
	select {
	case <-time.After(1 * time.Second):
		fmt.Println("Operation completed successfully")
	case <-ctxWithTimeout.Done():
		fmt.Println("Operation timed out")
	}

	// Access components from context
	if _, ok := ctx.Memory(); ok {
		fmt.Println("Memory found in context")
		// Use memory...
	}

	if tools, ok := ctx.Tools(); ok {
		fmt.Println("Tools found in context:")
		for _, tool := range tools.List() {
			fmt.Printf("- %s: %s\n", tool.Name(), tool.Description())
		}
	}

	if _, ok := ctx.LLM(); ok {
		fmt.Println("LLM found in context")
		// Use LLM...
	}

	if temp, ok := ctx.Environment("temperature"); ok {
		fmt.Printf("Temperature: %v\n", temp)
	}
}
