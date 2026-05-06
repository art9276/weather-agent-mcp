package main

import (
	"context"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
)

// AgentLogger реализует интерфейс callbacks.Handler
type AgentLogger struct {
	l Logger
}

func NewAgentLogger(l Logger) *AgentLogger {
	return &AgentLogger{l: l}
}

// HandleText обрабатывает текстовый вывод
func (h *AgentLogger) HandleText(ctx context.Context, text string) {
	h.l.Debug("[Callback] Text", "text", text)
}

// HandleLLMStart вызывается при старте LLM
func (h *AgentLogger) HandleLLMStart(ctx context.Context, prompts []string) {
	h.l.Info("[Callback] LLM Start", "prompts_count", len(prompts))
}

// HandleLLMGenerateContentStart вызывается перед генерацией контента
func (h *AgentLogger) HandleLLMGenerateContentStart(ctx context.Context, ms []llms.MessageContent) {
	h.l.Debug("[Callback] LLM Generate Content Start", "messages_count", len(ms))
}

// HandleLLMGenerateContentEnd вызывается после генерации контента
func (h *AgentLogger) HandleLLMGenerateContentEnd(ctx context.Context, res *llms.ContentResponse) {
	if res != nil && len(res.Choices) > 0 {
		h.l.Debug("[Callback] LLM Generate Content End", "content_preview", res.Choices[0].Content[:min(len(res.Choices[0].Content), 50)])
	}
}

// HandleLLMError вызывается при ошибке LLM
func (h *AgentLogger) HandleLLMError(ctx context.Context, err error) {
	h.l.Error("[Callback] LLM Error", "error", err)
}

// HandleChainStart вызывается при старте цепочки
func (h *AgentLogger) HandleChainStart(ctx context.Context, inputs map[string]any) {
	h.l.Info("[Callback] Chain Start")
}

// HandleChainEnd вызывается при завершении цепочки
func (h *AgentLogger) HandleChainEnd(ctx context.Context, outputs map[string]any) {
	h.l.Info("[Callback] Chain End")
}

// HandleChainError вызывается при ошибке цепочки
func (h *AgentLogger) HandleChainError(ctx context.Context, err error) {
	h.l.Error("[Callback] Chain Error", "error", err)
}

// HandleToolStart вызывается при старте инструмента
func (h *AgentLogger) HandleToolStart(ctx context.Context, input string) {
	// В этом методе имя инструмента часто недоступно напрямую в новых версиях,
	// но мы логируем входные данные
	h.l.Info("[Callback] Tool Start", "input_preview", truncate(input, 100))
}

// HandleToolEnd вызывается при завершении инструмента
func (h *AgentLogger) HandleToolEnd(ctx context.Context, output string) {
	h.l.Info("[Callback] Tool End", "output_preview", truncate(output, 100))
}

// HandleToolError вызывается при ошибке инструмента
func (h *AgentLogger) HandleToolError(ctx context.Context, err error) {
	h.l.Error("[Callback] Tool Error", "error", err)
}

// HandleAgentAction вызывается при действии агента
func (h *AgentLogger) HandleAgentAction(ctx context.Context, action schema.AgentAction) {
	h.l.Info("[Callback] Agent Action",
		"tool", action.Tool,
		"input", action.ToolInput)
}

// HandleAgentFinish вызывается при завершении агента
func (h *AgentLogger) HandleAgentFinish(ctx context.Context, finish schema.AgentFinish) {
	h.l.Info("[Callback] Agent Finish",
		"output", finish.Log,
		"return_values", finish.ReturnValues)
}

// HandleRetrieverStart вызывается при старте retriever
func (h *AgentLogger) HandleRetrieverStart(ctx context.Context, query string) {
	h.l.Debug("[Callback] Retriever Start", "query", query)
}

// HandleRetrieverEnd вызывается при завершении retriever
func (h *AgentLogger) HandleRetrieverEnd(ctx context.Context, query string, documents []schema.Document) {
	h.l.Debug("[Callback] Retriever End", "docs_count", len(documents))
}

// HandleStreamingFunc вызывается при потоковой передаче
func (h *AgentLogger) HandleStreamingFunc(ctx context.Context, chunk []byte) {
	h.l.Debug("[Callback] Streaming", "chunk_size", len(chunk))
}

// Finish завершает обработку callback
func (h *AgentLogger) Finish() {
	// Пустая реализация, если требуется интерфейсом
}

// Вспомогательная функция
func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
