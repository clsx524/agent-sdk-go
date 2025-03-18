package task

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
	"github.com/google/uuid"
)

// Service defines the interface for task management
type Service interface {
	// CreateTask creates a new task
	CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error)
	// GetTask gets a task by ID
	GetTask(ctx context.Context, taskID string) (*Task, error)
	// ListTasks returns tasks filtered by the provided criteria
	ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, error)
	// ApproveTaskPlan approves or rejects a task plan
	ApproveTaskPlan(ctx context.Context, taskID string, req ApproveTaskPlanRequest) (*Task, error)
	// UpdateTask updates an existing task with new steps or modifications
	UpdateTask(ctx context.Context, taskID string, updates []TaskUpdate) (*Task, error)
	// AddTaskLog adds a log entry to a task
	AddTaskLog(ctx context.Context, taskID string, message string, level string) error
}

// TaskPlanner defines the interface for planning a task
type TaskPlanner interface {
	// CreatePlan creates a plan for a task
	CreatePlan(ctx context.Context, task *Task) (string, error)
}

// TaskExecutor defines the interface for executing a task's plan
type TaskExecutor interface {
	// ExecuteStep executes a single step in a task's plan
	ExecuteStep(ctx context.Context, task *Task, step *Step) error
	// ExecuteTask executes all steps in a task's plan
	ExecuteTask(ctx context.Context, task *Task) error
}

// InMemoryTaskService implements the Service interface with an in-memory storage
type InMemoryTaskService struct {
	tasks         map[string]*Task
	mutex         sync.RWMutex
	logger        logging.Logger
	taskHistories map[string][]string
	planner       TaskPlanner
	executor      TaskExecutor
}

// NewInMemoryTaskService creates a new in-memory task service
func NewInMemoryTaskService(logger logging.Logger, planner TaskPlanner, executor TaskExecutor) *InMemoryTaskService {
	return &InMemoryTaskService{
		tasks:         make(map[string]*Task),
		taskHistories: make(map[string][]string),
		mutex:         sync.RWMutex{},
		logger:        logger,
		planner:       planner,
		executor:      executor,
	}
}

// CreateTask creates a new task
func (s *InMemoryTaskService) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	taskID := uuid.New().String()
	s.logger.Info(ctx, "Creating new task", map[string]interface{}{
		"task_id": taskID,
	})

	task := &Task{
		ID:          taskID,
		Description: req.Description,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		UserID:      req.UserID,
		Logs:        []LogEntry{},
		Metadata:    req.Metadata,
	}

	// Add initial log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Message:   "Task created",
		Level:     "info",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)

	// Store the task
	s.mutex.Lock()
	s.tasks[taskID] = task
	s.taskHistories[taskID] = []string{req.Description}
	s.mutex.Unlock()

	s.logger.Info(ctx, "Task created successfully", map[string]interface{}{
		"task_id": taskID,
	})

	// Start planning in a goroutine if planner is available
	if s.planner != nil {
		go s.planTask(context.Background(), task)
	}

	return task, nil
}

// GetTask gets a task by ID
func (s *InMemoryTaskService) GetTask(ctx context.Context, taskID string) (*Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	s.logger.Info(ctx, "Getting task", map[string]interface{}{
		"task_id": taskID,
	})

	task, ok := s.tasks[taskID]
	if !ok {
		s.logger.Error(ctx, "Task not found", map[string]interface{}{
			"task_id": taskID,
		})
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// ListTasks returns tasks filtered by the provided criteria
func (s *InMemoryTaskService) ListTasks(ctx context.Context, filter TaskFilter) ([]*Task, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	tasks := make([]*Task, 0)
	for _, task := range s.tasks {
		// Apply filters
		if filter.UserID != "" && task.UserID != filter.UserID {
			continue
		}

		if len(filter.Status) > 0 {
			statusMatch := false
			for _, status := range filter.Status {
				if task.Status == status {
					statusMatch = true
					break
				}
			}
			if !statusMatch {
				continue
			}
		}

		if filter.CreatedAfter != nil && task.CreatedAt.Before(*filter.CreatedAfter) {
			continue
		}

		if filter.CreatedBefore != nil && task.CreatedAt.After(*filter.CreatedBefore) {
			continue
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ApproveTaskPlan approves or rejects a task plan
func (s *InMemoryTaskService) ApproveTaskPlan(ctx context.Context, taskID string, req ApproveTaskPlanRequest) (*Task, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.logger.Info(ctx, "Approving task plan", map[string]interface{}{
		"task_id": taskID,
	})

	task, ok := s.tasks[taskID]
	if !ok {
		s.logger.Error(ctx, "Task not found", map[string]interface{}{
			"task_id": taskID,
		})
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	if task.Status != StatusApproval {
		s.logger.Error(ctx, "Task is not awaiting approval", map[string]interface{}{
			"task_id": taskID,
			"status":  string(task.Status),
		})
		return nil, fmt.Errorf("task is not awaiting approval")
	}

	if !req.Approved {
		// Handle rejection with feedback
		task.Status = StatusPlanning
		task.UpdatedAt = time.Now()

		// Add log entry
		logEntry := LogEntry{
			ID:        uuid.New().String(),
			TaskID:    taskID,
			Message:   fmt.Sprintf("Plan rejected with feedback: %s", req.Feedback),
			Level:     "info",
			Timestamp: time.Now(),
		}
		task.Logs = append(task.Logs, logEntry)

		// Store feedback in history
		if historyEntries, ok := s.taskHistories[taskID]; ok {
			s.taskHistories[taskID] = append(historyEntries, fmt.Sprintf("FEEDBACK: %s", req.Feedback))
		}

		// Replan with feedback if planner is available
		if s.planner != nil {
			go s.replanTask(context.Background(), task, req.Feedback)
		}

		return task, nil
	}

	// Mark plan as approved
	now := time.Now()
	task.Plan.IsApproved = true
	task.Plan.ApprovedAt = &now
	task.Status = StatusExecuting
	task.UpdatedAt = now

	// Add log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Message:   "Plan approved, starting execution",
		Level:     "info",
		Timestamp: now,
	}
	task.Logs = append(task.Logs, logEntry)

	s.logger.Info(ctx, "Plan approved, starting execution", map[string]interface{}{
		"task_id": taskID,
	})

	// Start execution in a goroutine if executor is available
	if s.executor != nil {
		go s.executor.ExecuteTask(context.Background(), task)
	}

	return task, nil
}

// UpdateTask updates an existing task with new steps or modifications
func (s *InMemoryTaskService) UpdateTask(ctx context.Context, taskID string, updates []TaskUpdate) (*Task, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		s.logger.Error(ctx, "Task not found", map[string]interface{}{
			"task_id": taskID,
		})
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	// Process updates
	for _, update := range updates {
		switch update.Type {
		case "add_step":
			newStep := Step{
				ID:          uuid.New().String(),
				PlanID:      task.Plan.ID,
				Description: update.Description,
				Status:      StepStatusPending,
				Order:       len(task.Plan.Steps) + 1,
			}
			task.Plan.Steps = append(task.Plan.Steps, newStep)
		case "modify_step":
			for i, step := range task.Plan.Steps {
				if step.ID == update.StepID {
					if update.Description != "" {
						task.Plan.Steps[i].Description = update.Description
					}
					if update.Status != "" {
						statusValue := Status(update.Status)
						task.Plan.Steps[i].Status = statusValue
					}
					break
				}
			}
		case "remove_step":
			for i, step := range task.Plan.Steps {
				if step.ID == update.StepID {
					// Remove step and reorder remaining steps
					task.Plan.Steps = append(task.Plan.Steps[:i], task.Plan.Steps[i+1:]...)
					for j := i; j < len(task.Plan.Steps); j++ {
						task.Plan.Steps[j].Order = j + 1
					}
					break
				}
			}
		}
	}

	// Update task
	task.UpdatedAt = time.Now()

	// Add log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Message:   "Task plan updated",
		Level:     "info",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)

	return task, nil
}

// AddTaskLog adds a log entry to a task
func (s *InMemoryTaskService) AddTaskLog(ctx context.Context, taskID string, message string, level string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    taskID,
		Message:   message,
		Level:     level,
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)
	return nil
}

// planTask plans a task using the provided planner
func (s *InMemoryTaskService) planTask(_ context.Context, task *Task) {
	s.logger.Info(context.Background(), "Planning task", map[string]interface{}{
		"task_id": task.ID,
	})

	// Update task status
	s.mutex.Lock()
	task.Status = StatusPlanning
	task.UpdatedAt = time.Now()

	// Add log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Message:   "Planning started",
		Level:     "info",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)
	s.mutex.Unlock()

	ctx := context.Background()

	// Create a plan using the planner
	planContent, err := s.planner.CreatePlan(ctx, task)
	if err != nil {
		s.handlePlanningFailure(task, err)
		return
	}

	// Parse the plan and create a model
	plan := &Plan{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Steps:     []Step{},
		CreatedAt: time.Now(),
	}

	// Add the plan to the task
	s.mutex.Lock()
	task.Plan = plan
	task.Status = StatusApproval
	task.UpdatedAt = time.Now()

	// Add log entry
	logEntry = LogEntry{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Message:   "Plan created, awaiting approval",
		Level:     "info",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)

	// Store the original plan content as metadata
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["original_plan"] = planContent
	s.mutex.Unlock()

	s.logger.Info(ctx, "Task planning completed", map[string]interface{}{
		"task_id": task.ID,
	})
}

// handlePlanningFailure handles failures during the planning phase
func (s *InMemoryTaskService) handlePlanningFailure(task *Task, err error) {
	s.logger.Error(context.Background(), "Task planning failed", map[string]interface{}{
		"task_id": task.ID,
		"error":   err.Error(),
	})

	s.mutex.Lock()
	defer s.mutex.Unlock()

	task.Status = StatusFailed
	task.UpdatedAt = time.Now()

	// Add log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Message:   fmt.Sprintf("Planning failed: %s", err.Error()),
		Level:     "error",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)
}

// replanTask replans a task with feedback
func (s *InMemoryTaskService) replanTask(_ context.Context, task *Task, feedback string) {
	s.logger.Info(context.Background(), "Replanning task with feedback", map[string]interface{}{
		"task_id":  task.ID,
		"feedback": feedback,
	})

	// Create an updated description that includes the feedback
	updatedTask := *task
	if updatedTask.Metadata == nil {
		updatedTask.Metadata = make(map[string]interface{})
	}
	updatedTask.Metadata["feedback"] = feedback

	// Create a new plan with the feedback incorporated
	ctx := context.Background()
	planContent, err := s.planner.CreatePlan(ctx, &updatedTask)
	if err != nil {
		s.handlePlanningFailure(task, err)
		return
	}

	// Parse the generated plan and convert it to our task model
	plan := &Plan{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Steps:     []Step{},
		CreatedAt: time.Now(),
	}

	// Update task with plan
	s.mutex.Lock()
	task.Plan = plan
	task.Status = StatusApproval
	task.UpdatedAt = time.Now()

	// Store the plan content
	if task.Metadata == nil {
		task.Metadata = make(map[string]interface{})
	}
	task.Metadata["updated_plan"] = planContent

	// Add log entry
	logEntry := LogEntry{
		ID:        uuid.New().String(),
		TaskID:    task.ID,
		Message:   "Plan updated with feedback, awaiting approval",
		Level:     "info",
		Timestamp: time.Now(),
	}
	task.Logs = append(task.Logs, logEntry)
	s.mutex.Unlock()

	s.logger.Info(ctx, "Task replanning completed", map[string]interface{}{
		"task_id": task.ID,
	})
}
