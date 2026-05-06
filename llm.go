package main

import (
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	model = "GigaChat-2"
	url   = "https://gigachat.devices.sberbank.ru/api/v1"
)

func getGigaChatLLM(token string) (*openai.LLM, error) {
	llm, err := openai.New(openai.WithToken(token), openai.WithBaseURL(url), openai.WithModel(model))
	if err != nil {
		return nil, err

	}
	return llm, nil
}
