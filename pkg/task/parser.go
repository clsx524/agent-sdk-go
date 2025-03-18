package task

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Ingenimax/agent-sdk-go/pkg/logging"
)

// PlanParser is an interface for parsing different types of plans
type PlanParser interface {
	// ParsePlan parses a plan content into Steps
	ParsePlan(ctx context.Context, planContent string) ([]Step, error)
	// GetDescription extracts the plan description from the content
	GetDescription(ctx context.Context, planContent string) (string, error)
}

// CompositePlanParser tries multiple parsers in sequence until one succeeds
type CompositePlanParser struct {
	parsers []PlanParser
	logger  logging.Logger
}

// NewCompositePlanParser creates a new composite plan parser
func NewCompositePlanParser(logger logging.Logger, parsers ...PlanParser) *CompositePlanParser {
	return &CompositePlanParser{
		parsers: parsers,
		logger:  logger,
	}
}

// AddParser adds a parser to the composite parser
func (p *CompositePlanParser) AddParser(parser PlanParser) {
	p.parsers = append(p.parsers, parser)
}

// ParsePlan tries each parser in sequence until one succeeds
func (p *CompositePlanParser) ParsePlan(ctx context.Context, planContent string) ([]Step, error) {
	var lastErr error
	for _, parser := range p.parsers {
		steps, err := parser.ParsePlan(ctx, planContent)
		if err == nil && len(steps) > 0 {
			return steps, nil
		}
		lastErr = err
	}

	// If none of the parsers succeeded, return an error
	if lastErr != nil {
		return nil, fmt.Errorf("all parsers failed: %w", lastErr)
	}
	return nil, errors.New("no steps could be extracted from the plan")
}

// GetDescription tries each parser in sequence until one succeeds
func (p *CompositePlanParser) GetDescription(ctx context.Context, planContent string) (string, error) {
	var lastErr error
	for _, parser := range p.parsers {
		desc, err := parser.GetDescription(ctx, planContent)
		if err == nil && desc != "" {
			return desc, nil
		}
		lastErr = err
	}

	// If none of the parsers succeeded, return a default description
	if lastErr != nil {
		p.logger.Warn(ctx, "All description parsers failed", map[string]interface{}{
			"error": lastErr.Error(),
		})
	}
	return "Deployment plan", nil
}

// JSONPlanParser parses plans in JSON format
type JSONPlanParser struct {
	logger logging.Logger
}

// NewJSONPlanParser creates a new JSON plan parser
func NewJSONPlanParser(logger logging.Logger) *JSONPlanParser {
	return &JSONPlanParser{
		logger: logger,
	}
}

// ParsePlan parses a JSON plan into Steps
func (p *JSONPlanParser) ParsePlan(ctx context.Context, planContent string) ([]Step, error) {
	// First check if this is simple text without JSON markers
	if !strings.Contains(planContent, "{") && !strings.Contains(planContent, "}") {
		p.logger.Debug(ctx, "Content does not contain JSON markers", nil)
		return nil, errors.New("content does not appear to be JSON")
	}

	// Check if this looks like JSON
	if !strings.Contains(planContent, "\"description\"") ||
		!strings.Contains(planContent, "\"plan\"") ||
		!strings.Contains(planContent, "\"steps\"") {
		p.logger.Debug(ctx, "Content does not match expected JSON structure", nil)
		return nil, errors.New("content does not appear to be in expected JSON format")
	}

	p.logger.Debug(ctx, "Detected JSON format in plan content", nil)

	// Define a struct to match the JSON structure
	type JSONStep struct {
		Description string `json:"description"`
		Status      string `json:"status"`
		Order       int    `json:"order"`
	}

	type JSONPlan struct {
		Steps []JSONStep `json:"steps"`
	}

	type DeploymentPlan struct {
		Description string   `json:"description"`
		Plan        JSONPlan `json:"plan"`
	}

	// Try to extract the JSON part
	jsonStr := extractJSONFromText(planContent)
	if jsonStr == "" {
		return nil, errors.New("could not extract JSON from content")
	}

	// Pretty print the JSON for logging
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  "); err == nil {
		p.logger.Debug(ctx, "Extracted JSON from plan", map[string]interface{}{
			"json": "\n" + prettyJSON.String(),
		})
	}

	// Try to parse it
	var deploymentPlan DeploymentPlan
	err := json.Unmarshal([]byte(jsonStr), &deploymentPlan)
	if err != nil {
		p.logger.Warn(ctx, "Failed to parse JSON plan", map[string]interface{}{
			"error": err.Error(),
			"json":  jsonStr,
		})
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Successfully parsed JSON
	p.logger.Info(ctx, "Successfully parsed JSON plan", map[string]interface{}{
		"description": deploymentPlan.Description,
		"step_count":  len(deploymentPlan.Plan.Steps),
	})

	// Convert to our model
	steps := make([]Step, len(deploymentPlan.Plan.Steps))
	for i, step := range deploymentPlan.Plan.Steps {
		// Use Status constants where applicable
		var status Status
		switch strings.ToLower(step.Status) {
		case "pending":
			status = StepStatusPending
		case "executing":
			status = StepStatusExecuting
		case "completed":
			status = StepStatusCompleted
		case "failed":
			status = StepStatusFailed
		default:
			status = StepStatusPending
		}

		steps[i] = Step{
			Description: step.Description,
			Status:      status,
			Order:       step.Order,
		}
	}

	return steps, nil
}

// GetDescription extracts the description from a JSON plan
func (p *JSONPlanParser) GetDescription(ctx context.Context, planContent string) (string, error) {
	// Check if this looks like JSON
	if !strings.Contains(planContent, "\"description\"") {
		return "", errors.New("content does not appear to contain a JSON description")
	}

	// Try to extract the JSON part
	jsonStr := extractJSONFromText(planContent)
	if jsonStr == "" {
		return "", errors.New("could not extract JSON from content")
	}

	// Try to parse it
	var result struct {
		Description string `json:"description"`
	}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return "", fmt.Errorf("failed to parse JSON description: %w", err)
	}

	if result.Description == "" {
		return "", errors.New("empty description in JSON")
	}

	return result.Description, nil
}

// MarkdownPlanParser parses plans in Markdown format
type MarkdownPlanParser struct {
	logger logging.Logger
}

// NewMarkdownPlanParser creates a new Markdown plan parser
func NewMarkdownPlanParser(logger logging.Logger) *MarkdownPlanParser {
	return &MarkdownPlanParser{
		logger: logger,
	}
}

// ParsePlan parses a Markdown plan into Steps
func (p *MarkdownPlanParser) ParsePlan(ctx context.Context, planContent string) ([]Step, error) {
	// Check if this looks like Markdown
	if !strings.Contains(planContent, "#") && !strings.Contains(planContent, "-") {
		return nil, errors.New("content does not appear to be in Markdown format")
	}

	p.logger.Debug(ctx, "Detected Markdown format in plan content", nil)

	// Extract steps using various patterns
	var steps []Step

	// Try to find numbered steps (1. Step description)
	numberPattern := regexp.MustCompile(`(?m)^\s*(\d+)\.\s+(.+)$`)
	matches := numberPattern.FindAllStringSubmatch(planContent, -1)
	if len(matches) > 0 {
		for i, match := range matches {
			if len(match) >= 3 {
				steps = append(steps, Step{
					Description: strings.TrimSpace(match[2]),
					Status:      StepStatusPending,
					Order:       i + 1,
				})
			}
		}
	}

	// If no numbered steps found, try bullet points
	if len(steps) == 0 {
		bulletPattern := regexp.MustCompile(`(?m)^\s*[-*]\s+(.+)$`)
		matches = bulletPattern.FindAllStringSubmatch(planContent, -1)
		if len(matches) > 0 {
			for i, match := range matches {
				if len(match) >= 2 {
					steps = append(steps, Step{
						Description: strings.TrimSpace(match[1]),
						Status:      StepStatusPending,
						Order:       i + 1,
					})
				}
			}
		}
	}

	// If we found steps, return them
	if len(steps) > 0 {
		p.logger.Info(ctx, "Successfully parsed Markdown plan", map[string]interface{}{
			"step_count": len(steps),
		})
		return steps, nil
	}

	return nil, errors.New("no steps found in Markdown content")
}

// GetDescription extracts the description from a Markdown plan
func (p *MarkdownPlanParser) GetDescription(ctx context.Context, planContent string) (string, error) {
	// Try to find a title (# Title)
	titlePattern := regexp.MustCompile(`(?m)^#\s+(.+)$`)
	matches := titlePattern.FindStringSubmatch(planContent)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1]), nil
	}

	// Try to find the first paragraph
	paragraphPattern := regexp.MustCompile(`(?m)^([^#\-*].+)$`)
	matches = paragraphPattern.FindStringSubmatch(planContent)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1]), nil
	}

	return "", errors.New("no description found in Markdown content")
}

// FallbackPlanParser creates generic steps when other parsers fail
type FallbackPlanParser struct {
	logger logging.Logger
}

// NewFallbackPlanParser creates a new fallback plan parser
func NewFallbackPlanParser(logger logging.Logger) *FallbackPlanParser {
	return &FallbackPlanParser{
		logger: logger,
	}
}

// ParsePlan creates generic steps based on common sections in the content
func (p *FallbackPlanParser) ParsePlan(ctx context.Context, planContent string) ([]Step, error) {
	p.logger.Info(ctx, "Using fallback plan parser", nil)

	steps := []Step{}
	order := 1

	// Create steps based on common sections in the content
	if strings.Contains(planContent, "Requirements") || strings.Contains(planContent, "Analysis") {
		steps = append(steps, Step{
			Description: "Analyze requirements and constraints",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++
	}

	if strings.Contains(planContent, "Infrastructure") {
		steps = append(steps, Step{
			Description: "Configure infrastructure resources",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++
	}

	if strings.Contains(planContent, "Deployment") {
		steps = append(steps, Step{
			Description: "Prepare deployment manifests",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++

		steps = append(steps, Step{
			Description: "Deploy the application",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++
	}

	if strings.Contains(planContent, "Monitoring") || strings.Contains(planContent, "Observability") {
		steps = append(steps, Step{
			Description: "Configure monitoring and logging",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++
	}

	if strings.Contains(planContent, "Testing") || strings.Contains(planContent, "Validation") {
		steps = append(steps, Step{
			Description: "Test and validate the deployment",
			Status:      StepStatusPending,
			Order:       order,
		})
		order++
	}

	// If no steps were created, use a generic set
	if len(steps) == 0 {
		steps = []Step{
			{
				Description: "Analyze requirements and constraints",
				Status:      StepStatusPending,
				Order:       1,
			},
			{
				Description: "Configure infrastructure resources",
				Status:      StepStatusPending,
				Order:       2,
			},
			{
				Description: "Prepare deployment manifests",
				Status:      StepStatusPending,
				Order:       3,
			},
			{
				Description: "Deploy the application",
				Status:      StepStatusPending,
				Order:       4,
			},
			{
				Description: "Configure monitoring and logging",
				Status:      StepStatusPending,
				Order:       5,
			},
		}
	}

	p.logger.Info(ctx, "Created fallback plan", map[string]interface{}{
		"step_count": len(steps),
	})

	return steps, nil
}

// GetDescription returns a generic description
func (p *FallbackPlanParser) GetDescription(ctx context.Context, planContent string) (string, error) {
	// Try to extract something meaningful from the content
	lines := strings.Split(planContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 10 && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") {
			// Found a potential description line
			if len(line) > 100 {
				// Trim long lines
				line = line[:100] + "..."
			}
			return line, nil
		}
	}

	return "Deployment plan", nil
}

// Helper function to extract JSON from text
func extractJSONFromText(text string) string {
	// Look for JSON between curly braces
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx >= 0 && endIdx > startIdx {
		return text[startIdx : endIdx+1]
	}

	// If we couldn't find JSON markers, return empty string
	return ""
}
