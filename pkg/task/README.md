# Task Execution Package

The task execution package provides functionality for executing tasks synchronously and asynchronously, including API calls and Temporal workflows.

## Features

- Execute tasks synchronously and asynchronously
- Built-in retry mechanism with configurable retry policies
- API client for making HTTP requests
- Temporal workflow integration
- Task cancellation and status tracking
- Task adapter pattern for integrating with agent-specific models

## Usage

### Basic Task Execution

```go
// Create a task executor
executor := agentsdk.NewTaskExecutor()

// Register a task
executor.RegisterTask("hello", func(ctx context.Context, params interface{}) (interface{}, error) {
    name, ok := params.(string)
    if !ok {
        name = "World"
    }
    return fmt.Sprintf("Hello, %s!", name), nil
})

// Execute the task synchronously
result, err := executor.ExecuteSync(context.Background(), "hello", "John", nil)
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    fmt.Printf("Result: %v\n", result.Data)
}

// Execute the task asynchronously
resultChan, err := executor.ExecuteAsync(context.Background(), "hello", "Jane", nil)
if err != nil {
    fmt.Printf("Error: %v\n", err)
} else {
    result := <-resultChan
    fmt.Printf("Result: %v\n", result.Data)
}
```

### API Task Execution

```go
// Create an API client
apiClient := agentsdk.NewAPIClient("https://api.example.com", 10*time.Second)

// Register an API task
executor.RegisterTask("get_data", agentsdk.APITask(apiClient, task.APIRequest{
    Method: "GET",
    Path:   "/data",
    Query:  map[string]string{"limit": "10"},
}))

// Execute the API task with retry policy
timeout := 5 * time.Second
retryPolicy := &interfaces.RetryPolicy{
    MaxRetries:        3,
    InitialBackoff:    100 * time.Millisecond,
    MaxBackoff:        1 * time.Second,
    BackoffMultiplier: 2.0,
}

result, err := executor.ExecuteSync(context.Background(), "get_data", nil, &interfaces.TaskOptions{
    Timeout:     &timeout,
    RetryPolicy: retryPolicy,
})
```

### Using the Task Adapter Pattern

The task adapter pattern allows you to use your own agent-specific models while still leveraging the SDK's task management functionality. This pattern separates the concerns of the SDK from your agent-specific implementations.

Here's an example of how to implement a custom task adapter:

```go
// Define your agent-specific task models
type MyTask struct {
    ID          string
    Name        string
    Status      string
    CreatedAt   time.Time
    CompletedAt *time.Time
}

type MyCreateRequest struct {
    Name   string
    UserID string
}

type MyApprovalRequest struct {
    Approved bool
    Comment  string
}

type MyTaskUpdate struct {
    Type   string
    ID     string
    Status string
}

// Implement the TaskAdapter interface
type MyTaskAdapter struct {
    logger logging.Logger
}

// Create a new adapter
func NewMyTaskAdapter(logger logging.Logger) task.TaskAdapter[MyTask, MyCreateRequest, MyApprovalRequest, MyTaskUpdate] {
    return &MyTaskAdapter{
        logger: logger,
    }
}

// Implement conversion methods
func (a *MyTaskAdapter) ConvertCreateRequest(req MyCreateRequest) task.CreateTaskRequest {
    return task.CreateTaskRequest{
        Description: req.Name,
        UserID:      req.UserID,
        Metadata:    make(map[string]interface{}),
    }
}

func (a *MyTaskAdapter) ConvertApproveRequest(req MyApprovalRequest) task.ApproveTaskPlanRequest {
    return task.ApproveTaskPlanRequest{
        Approved: req.Approved,
        Feedback: req.Comment,
    }
}

func (a *MyTaskAdapter) ConvertTaskUpdates(updates []MyTaskUpdate) []task.TaskUpdate {
    sdkUpdates := make([]task.TaskUpdate, len(updates))
    for i, update := range updates {
        sdkUpdates[i] = task.TaskUpdate{
            Type:   update.Type,
            StepID: update.ID,
            Status: update.Status,
        }
    }
    return sdkUpdates
}

func (a *MyTaskAdapter) ConvertTask(sdkTask *task.Task) MyTask {
    if sdkTask == nil {
        return MyTask{}
    }
    
    return MyTask{
        ID:          sdkTask.ID,
        Name:        sdkTask.Description,
        Status:      string(sdkTask.Status),
        CreatedAt:   sdkTask.CreatedAt,
        CompletedAt: sdkTask.CompletedAt,
    }
}

func (a *MyTaskAdapter) ConvertTasks(sdkTasks []*task.Task) []MyTask {
    tasks := make([]MyTask, len(sdkTasks))
    for i, sdkTask := range sdkTasks {
        tasks[i] = a.ConvertTask(sdkTask)
    }
    return tasks
}

// Use the adapter with a task service
type MyTaskService struct {
    sdkService task.Service
    adapter    task.TaskAdapter[MyTask, MyCreateRequest, MyApprovalRequest, MyTaskUpdate]
}

func NewMyTaskService(sdkService task.Service, adapter task.TaskAdapter[MyTask, MyCreateRequest, MyApprovalRequest, MyTaskUpdate]) *MyTaskService {
    return &MyTaskService{
        sdkService: sdkService,
        adapter:    adapter,
    }
}

// Create a task using your own models
func (s *MyTaskService) CreateTask(ctx context.Context, req MyCreateRequest) (MyTask, error) {
    // Convert to SDK request
    sdkReq := s.adapter.ConvertCreateRequest(req)
    
    // Create task using SDK service
    sdkTask, err := s.sdkService.CreateTask(ctx, sdkReq)
    if err != nil {
        return MyTask{}, err
    }
    
    // Convert back to your model
    return s.adapter.ConvertTask(sdkTask), nil
}
```

### Temporal Workflow Execution

```go
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
result, err := executor.ExecuteSync(context.Background(), "example_workflow", map[string]interface{}{
    "input": "example input",
}, nil)
```

## Task Options

You can configure task execution with the following options:

```go
options := &interfaces.TaskOptions{
    // Timeout specifies the maximum duration for task execution
    Timeout: &timeout,
    
    // RetryPolicy specifies the retry policy for the task
    RetryPolicy: &interfaces.RetryPolicy{
        MaxRetries:        3,
        InitialBackoff:    100 * time.Millisecond,
        MaxBackoff:        1 * time.Second,
        BackoffMultiplier: 2.0,
    },
    
    // Metadata contains additional information for the task execution
    Metadata: map[string]interface{}{
        "purpose": "example",
    },
}
```

## Task Result

The task result contains the following information:

```go
type TaskResult struct {
    // Data contains the result data
    Data interface{}
    
    // Error contains any error that occurred during task execution
    Error error
    
    // Metadata contains additional information about the task execution
    Metadata map[string]interface{}
}
``` 