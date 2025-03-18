# Execution Plan Example

This example demonstrates how to use the execution plan functionality in the Agent SDK. The execution plan feature allows an agent to present a plan of actions to the user before executing them, giving the user the opportunity to approve, modify, or cancel the plan.

## Overview

The execution plan functionality provides several benefits:

1. **Transparency**: Users can see what actions the agent intends to take before they are executed.
2. **Control**: Users can approve, modify, or cancel the plan before execution.
3. **Safety**: Prevents the agent from taking unexpected or unwanted actions.

## How It Works

1. The agent generates an execution plan based on the user's request.
2. The plan is presented to the user for review.
3. The user can:
   - Approve the plan (by typing "approve" or "yes")
   - Modify the plan (by typing "modify" or "change" followed by the desired modifications)
   - Cancel the plan (by typing "cancel" or "no")
4. If approved, the agent executes the plan and reports the results.

## Implementation Details

The execution plan functionality is implemented in the following files:

- `pkg/agent/execution_plan.go`: Contains the core execution plan functionality.
- `pkg/agent/agent.go`: Modified to support execution plan generation and approval.

The key components are:

- `ExecutionPlan`: A struct that represents a plan of tool executions.
- `ExecutionStep`: A struct that represents a single step in an execution plan.
- `GenerateExecutionPlan`: A method that generates an execution plan based on user input.
- `ModifyExecutionPlan`: A method that modifies an execution plan based on user feedback.
- `ApproveExecutionPlan`: A method that approves and executes a plan.

## Running the Example

To run this example:

```bash
go run cmd/examples/execution_plan/main.go
```

The example uses a mock LLM for demonstration purposes. In a real application, you would use a real LLM provider like OpenAI.

## Example Conversation

Here's an example of how a conversation might flow:

```
User: What's the weather in New York and convert the temperature to Celsius?

Agent: I've created an execution plan for your request:

# Execution Plan: Search for weather information and calculate temperature conversion

## Step 1: Search for current weather information
Tool: search
Input: current weather in New York
Parameters:
- param1: weather

## Step 2: Convert temperature from Fahrenheit to Celsius
Tool: calculator
Input: convert 75F to C
Parameters:

Would you like to approve, modify, or cancel this plan?

User: modify I want to use the weather tool directly instead of search

Agent: I've updated the execution plan based on your feedback:

# Execution Plan: Search for weather information and calculate temperature conversion

## Step 1: Get current weather information directly
Tool: weather
Input: New York
Parameters:
- param1: New York

## Step 2: Convert temperature from Fahrenheit to Celsius
Tool: calculator
Input: convert 75F to C
Parameters:

Do you approve this plan? You can modify it further if needed.

User: approve

Agent: Executing the approved plan...

Step 1: Get current weather information directly
Result: Executed weather with input: New York

Step 2: Convert temperature from Fahrenheit to Celsius
Result: Executed calculator with input: convert 75F to C
```

## Customization

You can customize the execution plan functionality by:

1. Setting `requirePlanApproval` to `false` to disable execution plan approval.
2. Modifying the prompt templates in `createExecutionPlanPrompt` and `ModifyExecutionPlan`.
3. Extending the `ExecutionPlan` and `ExecutionStep` structs with additional fields.

## Integration with Your Application

To integrate execution plan functionality into your application:

1. Create an agent with `WithRequirePlanApproval(true)`.
2. Handle user responses to execution plans in your application's UI.
3. Call the appropriate methods (`ApproveExecutionPlan`, `ModifyExecutionPlan`) based on user responses. 