# API Server Example

This example demonstrates how to use the execution plan functionality in an API server. It shows how to implement a workflow similar to the one described in the bash script, where users can:

1. Start a conversation with the agent
2. Provide requirements
3. Review and modify execution plans
4. Approve plans for execution
5. Monitor execution
6. Discuss results

## Overview

The API server provides the following endpoints:

- `POST /api/v1/chat`: Send messages to the agent
- `GET /api/v1/tasks/{id}`: Get details of a specific task
- `GET /api/v1/tasks`: List all tasks

## How It Works

1. The user starts a conversation by sending a message to the `/api/v1/chat` endpoint.
2. The agent generates an execution plan based on the user's request.
3. The plan is presented to the user for review.
4. The user can modify the plan by sending modification requests to the `/api/v1/chat` endpoint.
5. The user can approve the plan by sending an approval message to the `/api/v1/chat` endpoint.
6. The agent executes the plan and reports the results.
7. The user can check the status of the task using the `/api/v1/tasks/{id}` endpoint.

## Implementation Details

The API server is implemented using the standard Go HTTP package. It maintains a map of agents for each user and a map of conversations. When a user sends a message, the server:

1. Gets or creates a conversation for the user
2. Gets or creates an agent for the user
3. Processes the message with the agent
4. Returns the agent's response along with any task IDs

If the agent generates an execution plan, the task ID is included in the response. The user can then:

- Get details of the plan using the `/api/v1/tasks/{id}` endpoint
- Modify the plan by sending a modification request to the `/api/v1/chat` endpoint
- Approve the plan by sending an approval message to the `/api/v1/chat` endpoint

## Running the Example

To run this example:

```bash
go run cmd/examples/api_server/main.go
```

The server will listen on port 8080.

## Example Workflow

Here's an example of how to use the API server:

1. Start a conversation:

```bash
curl -X POST "http://localhost:8080/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user123",
    "message": "I need to deploy a new web application on AWS with a load balancer and auto-scaling."
  }'
```

2. Provide requirements:

```bash
curl -X POST "http://localhost:8080/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "CONVERSATION_ID",
    "user_id": "user123",
    "message": "It's a Node.js application. I want to deploy it in us-west-2. Auto-scaling should be based on CPU usage (>70%)."
  }'
```

3. Check the task:

```bash
curl -X GET "http://localhost:8080/api/v1/tasks/TASK_ID"
```

4. Modify the plan:

```bash
curl -X POST "http://localhost:8080/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "CONVERSATION_ID",
    "user_id": "user123",
    "message": "I'd like to make a few changes. Can we use t3.medium instances instead? Also, I want to add CloudFront for content delivery."
  }'
```

5. Approve the plan:

```bash
curl -X POST "http://localhost:8080/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "CONVERSATION_ID",
    "user_id": "user123",
    "message": "I approve the plan. Please proceed with the execution."
  }'
```

6. Check the task status:

```bash
curl -X GET "http://localhost:8080/api/v1/tasks/TASK_ID"
```

7. Discuss results:

```bash
curl -X POST "http://localhost:8080/api/v1/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "CONVERSATION_ID",
    "user_id": "user123",
    "message": "Great! The deployment is complete. How can I access my application now?"
  }'
```

## Customization

You can customize the API server by:

1. Adding authentication and authorization
2. Implementing persistent storage for conversations and tasks
3. Adding more endpoints for specific operations
4. Integrating with real LLM providers and tools
5. Adding error handling and logging

## Integration with Your Application

To integrate this API server with your application:

1. Use the API endpoints to communicate with the agent
2. Store conversation IDs and task IDs in your application
3. Display execution plans to users and allow them to modify and approve them
4. Monitor task execution and display results to users 