# Adapters Package

This package provides standardized adapters for various services to implement the interfaces defined in the `interfaces` package.

Note: The OpenAIAdapter has been removed as the OpenAI client now directly implements the `interfaces.LLM` interface. Use the OpenAI client directly:

```go
import (
    "github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
)

// Create OpenAI client
openaiClient := openai.NewClient(apiKey, 
    openai.WithModel("gpt-4o-mini"),
)

// Use the client with any component that expects an interfaces.LLM
// For example, with an agent:
agent, err := agent.NewAgent(
    agent.WithLLM(openaiClient),
    // ... other options
)
```

The OpenAI client provides helper functions for generating options:

```go
// Set temperature
openaiClient.Generate(ctx, prompt, openai.WithTemperature(0.7))

// Set max tokens
openaiClient.Generate(ctx, prompt, openai.WithMaxTokens(1000))

// Set multiple options
openaiClient.Generate(ctx, prompt, 
    openai.WithTemperature(0.7),
    openai.WithMaxTokens(1000),
    openai.WithTopP(0.9),
)
``` 