package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentsdk "github.com/Ingenimax/agent-sdk-go/pkg"
	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/Ingenimax/agent-sdk-go/pkg/task"
)

func main() {
	// Create a logger
	logger := logging.New()

	// Create a task executor
	executor := agentsdk.NewTaskExecutor()

	// Register a simple task
	executor.RegisterTask("hello", func(ctx context.Context, params interface{}) (interface{}, error) {
		name, ok := params.(string)
		if !ok {
			name = "World"
		}
		return fmt.Sprintf("Hello, %s!", name), nil
	})

	// Execute the task synchronously
	logger.Info(context.Background(), "Executing task synchronously", nil)
	result, err := executor.ExecuteSync(context.Background(), "hello", "John", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Result: %v\n", result.Data)
	}

	// Execute the task asynchronously
	logger.Info(context.Background(), "Executing task asynchronously", nil)
	resultChan, err := executor.ExecuteAsync(context.Background(), "hello", "Jane", nil)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		result := <-resultChan
		fmt.Printf("Result: %v\n", result.Data)
	}

	// Create an API client
	apiClient := agentsdk.NewAPIClient("https://jsonplaceholder.typicode.com", 10*time.Second)

	// Register an API task
	executor.RegisterTask("get_todos", agentsdk.APITask(apiClient, task.APIRequest{
		Method: "GET",
		Path:   "/todos/1",
	}))

	// Execute the API task
	logger.Info(context.Background(), "Executing API task", nil)
	timeout := 5 * time.Second
	retryPolicy := &interfaces.RetryPolicy{
		MaxRetries:        3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        1 * time.Second,
		BackoffMultiplier: 2.0,
	}

	result, err = executor.ExecuteSync(context.Background(), "get_todos", nil, &interfaces.TaskOptions{
		Timeout:     &timeout,
		RetryPolicy: retryPolicy,
		Metadata: map[string]interface{}{
			"purpose": "example",
		},
	})

	if err != nil {
		logger.Error(context.Background(), "Error executing task", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		// Check if the result is a byte array and try to decode it as JSON
		if byteData, ok := result.Data.([]byte); ok {
			var todoData map[string]interface{}
			if err := json.Unmarshal(byteData, &todoData); err != nil {
				logger.Error(context.Background(), "Failed to unmarshal JSON response", map[string]interface{}{
					"error": err.Error(),
				})
				logger.Info(context.Background(), "Result (raw bytes)", map[string]interface{}{
					"data": string(byteData),
				})
			} else {
				logger.Info(context.Background(), "Result decoded", map[string]interface{}{
					"data": todoData,
				})
			}
		} else {
			logger.Info(context.Background(), "Result", map[string]interface{}{
				"data": fmt.Sprintf("%v", result.Data),
			})
		}
		logger.Info(context.Background(), "Metadata", map[string]interface{}{
			"metadata": result.Metadata,
		})
	}

	// Create a Temporal client
	temporalClient := agentsdk.NewTemporalClient(task.TemporalConfig{
		HostPort:                 "localhost:7233",
		Namespace:                "default",
		TaskQueue:                "example",
		WorkflowIDPrefix:         "example-",
		WorkflowExecutionTimeout: 10 * time.Minute,
		WorkflowRunTimeout:       5 * time.Minute,
		WorkflowTaskTimeout:      10 * time.Second,
	})

	// Register a Temporal workflow task
	executor.RegisterTask("example_workflow", agentsdk.TemporalWorkflowTask(temporalClient, "ExampleWorkflow"))

	// Execute the Temporal workflow task
	logger.Info(context.Background(), "Executing Temporal workflow task (this is a placeholder)", nil)
	result, err = executor.ExecuteSync(context.Background(), "example_workflow", map[string]interface{}{
		"input": "example input",
	}, nil)

	if err != nil {
		logger.Error(context.Background(), "Error executing task", map[string]interface{}{
			"error": err.Error(),
		})
	} else {
		logger.Info(context.Background(), "Result", map[string]interface{}{
			"data": fmt.Sprintf("%v", result.Data),
		})
		logger.Info(context.Background(), "Error", map[string]interface{}{
			"error": fmt.Sprintf("%v", result.Error),
		})
		logger.Info(context.Background(), "Metadata", map[string]interface{}{
			"metadata": result.Metadata,
		})
	}
}
