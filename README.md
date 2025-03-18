# ![Ingenimax](/docs/img/logo-header.png#gh-light-mode-only) ![Ingenimax](/docs/img/logo-header-inverted.png#gh-dark-mode-only)

# Agent SDK for Go

A flexible and extensible SDK for building AI agents in Go.

## Overview

The Agent SDK provides a comprehensive framework for building AI-powered agents in Go. It offers a modular architecture that allows you to easily integrate with various LLM providers, memory systems, vector stores, and tools.

## Features

- **Multiple LLM Providers**: Support for OpenAI, Anthropic, and Azure OpenAI
- **Memory Management**: Conversation history and context management
- **Tool Integration**: Easily extend your agent with custom tools
- **Vector Store Integration**: Connect to Weaviate, Pinecone, and other vector databases
- **Tracing and Observability**: Built-in support for Langfuse and OpenTelemetry
- **Multi-tenancy**: Support for multiple organizations and users
- **Guardrails**: Safety mechanisms to ensure responsible AI usage
- **Execution Plans**: Allow users to review and modify tool execution plans before execution

## Installation

```bash
go get github.com/Ingenimax/agent-sdk-go
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Ingenimax/agent-sdk-go/pkg/agent"
	"github.com/Ingenimax/agent-sdk-go/pkg/config"
	"github.com/Ingenimax/agent-sdk-go/pkg/llm/openai"
	"github.com/Ingenimax/agent-sdk-go/pkg/memory"
)

func main() {
	// Get configuration
	cfg := config.Get()

	// Create OpenAI client
	openaiClient := openai.NewClient(cfg.LLM.OpenAI.APIKey)

	// Create a new agent
	agent, err := agent.NewAgent(
		agent.WithLLM(openaiClient),
		agent.WithMemory(memory.NewConversationBuffer()),
		agent.WithSystemPrompt("You are a helpful AI assistant."),
	)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Run the agent
	response, err := agent.Run(context.Background(), "What is the capital of France?")
	if err != nil {
		log.Fatalf("Failed to run agent: %v", err)
	}

	fmt.Println(response)
}
```

## Configuration

The SDK uses environment variables for configuration. See [Environment Variables](docs/environment_variables.md) for a complete list.

## Examples

Check out the [examples](cmd/examples) directory for more detailed examples:

- [Simple Agent](cmd/examples/simple_agent): Basic agent setup
- [LLM Providers](cmd/examples/llm): Using different LLM providers
- [Memory Systems](cmd/examples/memory): Working with different memory systems
- [Vector Stores](cmd/examples/vectorstore): Integrating with vector databases
- [Tracing](cmd/examples/tracing): Adding observability to your agent
- [Orchestration](cmd/examples/orchestration): Coordinating multiple agents
- [Guardrails](cmd/examples/guardrails): Adding safety mechanisms
- [Embedding](cmd/examples/embedding): Working with embeddings
- [Context](cmd/examples/context): Managing context in conversations
- [Task Execution](cmd/examples/task_execution): Tasks execution
- [Execution Plan](cmd/examples/execution_plan): Allowing users to review and modify execution plans
- [API Server](cmd/examples/api_server): Implementing an API server with execution plan functionality

## Documentation

- [Agent](docs/agent.md): Core agent functionality
- [LLM Providers](docs/llm.md): Language model integrations
- [Memory](docs/memory.md): Conversation history management
- [Tools](docs/tools.md): Extending agent capabilities
- [Vector Store](docs/vectorstore.md): Semantic search and retrieval
- [Tracing](docs/tracing.md): Observability and monitoring
- [Guardrails](docs/guardrails.md): Safety and content filtering
- [Environment Variables](docs/environment_variables.md): Configuration options
- [Multitenancy](docs/multitenancy.md): Supporting multiple organizations
- [Task](docs/task.md): Tasks execution

## Architecture

The Agent SDK follows a modular architecture with the following key components:

- **Agent**: The core component that coordinates the LLM, memory, and tools
- **LLM**: Interface to language model providers (OpenAI, Anthropic, etc.)
- **Memory**: Stores conversation history and context
- **Tools**: Extend the agent's capabilities with custom tools
- **Vector Store**: Store and retrieve vector embeddings
- **Data Store**: Persist data for the agent
- **Tracing**: Monitor and debug agent behavior
- **Guardrails**: Ensure safe and responsible AI usage

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.

# StarOps Agent

A PlatformOps assistant that helps with deployment and management tasks.

## Features

- Interactive command-line interface
- Natural language understanding for deployment requests
- Multi-agent architecture with specialist agents
- Detailed deployment planning
- Task management system with approval workflow

## Task Management

The StarOps Agent includes a task management system that provides tracking and approval for deployment tasks:

### Task Commands

- `task list` - List all your tasks
- `task get <task-id>` - View details of a specific task
- `task approve <task-id>` - Approve a task's execution plan
- `task reject <task-id> <feedback>` - Reject a task's plan with feedback for improvement

### Task Workflow

1. Create a task by asking the agent to deploy something (e.g., "deploy llama 3.3 on kserve")
2. The agent will create a task and provide you with the task ID
3. Check the status with `task get <task-id>`
4. Once planning is complete, review the plan and either:
   - Approve it with `task approve <task-id>`
   - Reject it with `task reject <task-id> The plan needs more security measures`
5. If approved, the task will be executed
6. If rejected, the task will be replanned with your feedback

### Task Statuses

- `pending` - Task has been created but planning hasn't started
- `planning` - Task plan is being created
- `awaiting_approval` - Plan is ready for your review
- `executing` - Plan is being executed
- `completed` - Task has been completed successfully
- `failed` - Task has failed

## Usage

1. Run the application:
   ```
   go run main.go
   ```

2. Ask the agent to deploy something:
   ```
   Enter your query: deploy llama 3.3 on kserve
   ```

3. Follow the task workflow described above to manage your deployment.

## Development

The StarOps Agent is built with a modular architecture:

- `internal/agent` - Agent implementation and factory
- `internal/planning` - Deployment planning and project management
- `internal/services` - Task management and other services
- `internal/models` - Data models for tasks and plans
- `internal/conversation` - Session handling and user interaction
- `internal/prompts` - System prompts for different agents

## License

Copyright © 2025 Ingenimax Inc.
