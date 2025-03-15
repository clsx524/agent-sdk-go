package calculator

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// Calculator is a tool for performing mathematical calculations
type Calculator struct{}

// Name returns the name of the tool
func (c *Calculator) Name() string {
	return "calculator"
}

// Description returns the description of the tool
func (c *Calculator) Description() string {
	return "Performs mathematical calculations. Input should be a valid mathematical expression."
}

// Execute performs the calculation
func (c *Calculator) Execute(ctx context.Context, input string) (string, error) {
	// Clean the input
	input = strings.TrimSpace(input)

	// If the input looks like JSON, attempt to unmarshal and extract the "expression" field
	if strings.HasPrefix(input, "{") {
		var data map[string]string
		err := json.Unmarshal([]byte(input), &data)
		if err == nil {
			if expr, ok := data["expression"]; ok {
				input = strings.TrimSpace(expr)
				fmt.Printf("DEBUG - Extracted expression from JSON: %s\n", input)
			}
		}
	}

	// Debug: Log the input before parsing
	fmt.Printf("DEBUG - Parsing expression: %s\n", input)

	// Parse the expression
	expr, err := parser.ParseExpr(input)
	if err != nil {
		fmt.Printf("DEBUG - Parse error: %v\n", err)
		return "", fmt.Errorf("failed to parse expression: %w", err)
	}

	// Debug: Log the parsed expression
	fmt.Printf("DEBUG - Parsed expression: %v\n", expr)

	// Evaluate the expression
	result, err := evaluateExpr(expr)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", result), nil
}

// evaluateExpr evaluates an AST expression
func evaluateExpr(expr ast.Expr) (float64, error) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT || e.Kind == token.FLOAT {
			return strconv.ParseFloat(e.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal type: %v", e.Kind)

	case *ast.BinaryExpr:
		x, err := evaluateExpr(e.X)
		if err != nil {
			return 0, err
		}

		y, err := evaluateExpr(e.Y)
		if err != nil {
			return 0, err
		}

		switch e.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO:
			if y == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return x / y, nil
		default:
			return 0, fmt.Errorf("unsupported operator: %v", e.Op)
		}

	case *ast.ParenExpr:
		return evaluateExpr(e.X)

	case *ast.UnaryExpr:
		x, err := evaluateExpr(e.X)
		if err != nil {
			return 0, err
		}

		switch e.Op {
		case token.SUB:
			return -x, nil
		case token.ADD:
			return x, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %v", e.Op)
		}

	default:
		return 0, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// Parameters returns the parameters for the calculator tool
func (c *Calculator) Parameters() map[string]interfaces.ParameterSpec {
	return map[string]interfaces.ParameterSpec{
		"expression": {
			Description: "The mathematical expression to evaluate",
			Required:    true,
			Type:        "string",
		},
	}
}

// Run executes the calculator tool with the given parameters
func (c *Calculator) Run(ctx context.Context, input string) (string, error) {
	return c.Execute(ctx, input)
}

// NewCalculator creates a new calculator tool
func NewCalculator() interfaces.Tool {
	return &Calculator{}
}
