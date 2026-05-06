package weather

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
)

// интерфейс для логирования сообщений.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// HandleWeatherRequest — главная функция для обработки запроса о погоде.
func HandleWeatherRequest(ctx context.Context, llm llms.Model, tool *WeatherForecastTool, l Logger, userInput string) (string, error) {
	// Логируем начало обработки погодного запроса
	l.Info("[WeatherHandler] Starting weather forecast chain", "input", userInput)

	// Создаём цепочки для каждого этапа обработки
	// createExtractArgsChain — извлекает город и количество дней из запроса
	extractChain := createExtractArgsChain(llm, l)

	// createParseArgsChain — парсит JSON с аргументами в структуру
	parseArgsChain := createParseArgsChain(l)

	// createWeatherToolChain — вызывает инструмент погоды с полученными аргументами
	weatherToolChain := createWeatherToolChain(tool, l)

	// createFinalResponseChain — форматирует финальный ответ с рекомендациями
	finalResponseChain := createFinalResponseChain(llm)

	// Создаём последовательную цепочку из всех этапов
	accumulatingChain := NewAccumulatingSequentialChain([]chains.Chain{
		extractChain,
		parseArgsChain,
		weatherToolChain,
		finalResponseChain,
	})

	l.Info("[WeatherHandler] AccumulatingSequentialChain created, executing...")

	// Запускаем цепочку с пользовательским запросом
	result, err := accumulatingChain.Call(ctx, map[string]any{
		"userInput": userInput,
	})
	if err != nil {
		// Логируем ошибку выполнения цепочки
		l.Error("[WeatherHandler] AccumulatingSequentialChain execution failed", "error", err)
		return "", err
	}

	// Извлекаем текстовый результат из выходных данных
	finalOutput, ok := result["text"].(string)
	if !ok {
		// Логируем ошибку типа данных
		l.Error("[WeatherHandler] Invalid output type from AccumulatingSequentialChain")
		return "", fmt.Errorf("invalid output from AccumulatingSequentialChain")
	}

	// Логируем успешное завершение
	l.Info("[WeatherHandler] AccumulatingSequentialChain completed successfully")

	return finalOutput, nil
}
