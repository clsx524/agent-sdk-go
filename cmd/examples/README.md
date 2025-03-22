# Agent SDK Go Examples

This directory contains examples demonstrating how to use different components of the Agent SDK.

## Available Examples

- [**Task Execution**](./task_execution/README.md): Demonstrates how to use the task execution capabilities, including basic tasks, API tasks, and mock workflow tasks.
- [**AI Planning**](./ai_planning/README.md): Shows how to use AI-based planning to generate sophisticated task plans with different AI planner implementations.

## Running Examples

Each example can be run directly from its own directory:

```bash
cd task_execution
go run main.go
```

Or:

```bash
cd ai_planning
go run main.go
```

## Adding New Examples

When adding new examples:

1. Create a new directory for the example
2. Include a detailed README.md explaining what the example demonstrates
3. Make sure the example is well-commented and includes proper error handling
4. Update this README.md to include a reference to your new example

## Example Structure

Each example should follow this structure:

- A main.go file that demonstrates the feature
- A README.md file that explains the example
- Any additional files needed for the example

## Best Practices

When creating examples:

- Keep examples focused on one feature or concept
- Include comprehensive error handling
- Add comments explaining key concepts
- Make sure examples run standalone without external dependencies when possible
- Include sample output in the README.md 