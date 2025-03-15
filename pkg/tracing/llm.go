package tracing

import (
	"context"

	"github.com/Ingenimax/agent-sdk-go/pkg/interfaces"
)

// GenerateWithTools implements interfaces.LLM.GenerateWithTools for LLMMiddleware
func (m *LLMMiddleware) GenerateWithTools(ctx context.Context, prompt string, tools []interfaces.Tool, options ...interfaces.GenerateOption) (string, error) {
	return m.llm.GenerateWithTools(ctx, prompt, tools, options...)
}
