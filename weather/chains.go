package weather

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tmc/langchaingo/chains"
	"github.com/tmc/langchaingo/llms"
)

// AccumulatingSequentialChain — кастомная цепочка для последовательного выполнения нескольких цепочек.
// В отличие от стандартной SequentialChain, накапливает ключи между шагами,
// передавая результаты каждого предыдущего этапа следующему
// здесь мы просто реализуем интерфейс Сhain
type AccumulatingSequentialChain struct {
	chains []chains.Chain
}

// Call выполняет все цепочки последовательно, передавая результаты от одной к другой.
func (c *AccumulatingSequentialChain) Call(ctx context.Context, inputs map[string]any, options ...chains.ChainCallOption) (map[string]any, error) {
	// Инициализируем результат входными данными
	result := inputs

	// Последовательно выполняем каждую цепочку
	for _, chain := range c.chains {
		// Вызываем текущую цепочку с накопленными результатами
		stepResult, err := chains.Call(ctx, chain, result, options...)
		if err != nil {
			// Возвращаем ошибку при неудаче любой цепочки
			return nil, err
		}

		// Объединяем результаты с предыдущими (накопление ключей)
		result = mergeMaps(result, stepResult)
	}

	return result, nil
}

// GetInputKeys возвращает входные ключи первой цепочки
func (c *AccumulatingSequentialChain) GetInputKeys() []string {
	if len(c.chains) == 0 {
		return []string{}
	}
	// Возвращаем ключи первой цепочки в последовательности
	return []string{"userInput"}
}

// GetOutputKeys возвращает выходные ключи последней цепочки
func (c *AccumulatingSequentialChain) GetOutputKeys() []string {
	if len(c.chains) == 0 {
		return []string{}
	}
	// Возвращаем ключи последней цепочки в последовательности
	return []string{"text"}
}

// NewAccumulatingSequentialChain создаёт новую последовательную цепочку с накоплением результатов
func NewAccumulatingSequentialChain(chains []chains.Chain) *AccumulatingSequentialChain {
	return &AccumulatingSequentialChain{
		chains: chains,
	}
}

// createExtractArgsChain создаёт цепочку для извлечения аргументов из запроса
func createExtractArgsChain(llm llms.Model, l Logger) chains.Chain {
	return chains.NewTransform(
		func(ctx context.Context, input map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
			// Извлекаем пользовательский запрос из входных данных
			userInput, ok := input["userInput"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid userInput type")
			}

			// Логируем начало извлечения аргументов
			l.Info("[WeatherArgsExtract] Extracting city and days from user input", "input", userInput)

			// Формируем промпт для LLM используя шаблон
			// WeatherPromptTemplate.Format подставляет userInput в шаблон
			prompt, err := WeatherPromptTemplate.Format(map[string]any{"userInput": userInput})
			if err != nil {
				l.Error("[WeatherArgsExtract] Failed to format prompt", "error", err)
				return nil, err
			}

			// Создаём сообщение для LLM
			messages := []llms.MessageContent{
				llms.TextParts(llms.ChatMessageTypeHuman, prompt),
			}

			// Настраиваем ToolChoice для принудительного вызова weather_forecast
			toolChoice := llms.ToolChoice{
				Type:     "function",
				Function: &llms.FunctionReference{Name: "weather_forecast"},
			}

			l.Info("[WeatherArgsExtract] Calling LLM with ToolChoice", "function", "weather_forecast")

			// Вызываем LLM с инструментами и ToolChoice
			resp, err := llm.GenerateContent(ctx, messages,
				llms.WithTools(AvailableTools),
				llms.WithToolChoice(toolChoice),
			)
			if err != nil {
				l.Error("[WeatherArgsExtract] LLM.GenerateContent failed", "error", err)
				return nil, err
			}

			// Проверяем, что LLM вернула ответ
			if len(resp.Choices) == 0 {
				return nil, fmt.Errorf("no response from LLM")
			}

			// Получаем ответ от LLM
			llmResponse := resp.Choices[0].Content
			l.Info("[WeatherArgsExtract] LLM response received", "content_preview", llmResponse)

			// Извлекаем JSON из ответа LLM
			jsonStr := extractJSON(llmResponse)
			l.Debug("[WeatherArgsExtract] Extracted JSON", "json", jsonStr)

			// Возвращаем сырой JSON для следующего этапа парсинга
			return map[string]any{
				"raw_args": jsonStr,
			}, nil
		},
		[]string{"userInput"},
		[]string{"raw_args"},
	)
}

// createParseArgsChain создаёт цепочку для парсинга JSON аргументов
// Преобразует сырой JSON от LLM в структурированные данные (город и дни)
func createParseArgsChain(l Logger) chains.Chain {
	return chains.NewTransform(
		func(ctx context.Context, input map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
			// Извлекаем сырой JSON из предыдущего этапа
			rawArgs, ok := input["raw_args"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid raw_args type")
			}

			// Логируем сырой вывод LLM
			l.Debug("[ParseArgs] Raw LLM output", "raw", rawArgs, 200)

			// Извлекаем JSON из строки (на случай markdown-разметки)
			jsonStr := extractJSON(rawArgs)
			l.Debug("[ParseArgs] Extracted JSON", "json", jsonStr)

			// Парсим JSON в структуру weatherArgs
			var args weatherArgs
			if err := json.Unmarshal([]byte(jsonStr), &args); err != nil {
				// При ошибке парсинга используем значения по умолчанию
				l.Warn("[ParseArgs] Failed to parse JSON, using fallback", "error", err)
				args.City = "Волгоград"
				args.Days = 1
			}

			// Устанавливаем значения по умолчанию для пустых полей
			if args.City == "" {
				args.City = "Волгоград"
			}
			if args.Days < 1 || args.Days > 7 {
				args.Days = 1
			}

			// Логируем распарсенные аргументы
			l.Info("[ParseArgs] Parsed arguments", "city", args.City, "days", args.Days)

			// Возвращаем структурированные данные для следующего этапа
			return map[string]any{
				"city": args.City,
				"days": args.Days,
			}, nil
		},
		[]string{"raw_args"},
		[]string{"city", "days"},
	)
}

// createWeatherToolChain создаёт цепочку для вызова инструмента погоды
// Получает прогноз погоды для указанного города на заданное количество дней
func createWeatherToolChain(weatherTool *WeatherForecastTool, l Logger) chains.Chain {
	return chains.NewTransform(
		func(ctx context.Context, input map[string]any, _ ...chains.ChainCallOption) (map[string]any, error) {
			// Извлекаем город из предыдущего этапа
			city, ok := input["city"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid city type")
			}

			// Преобразуем количество дней в int
			days := convertToInt(input["days"])
			if days <= 0 {
				days = 1
			}

			// Логируем вызов инструмента погоды
			l.Info("[WeatherTool] Calling weather_forecast via LLM", "city", city, "days", days)

			// Формируем JSON-аргументы для вызова инструмента
			argsJSON, _ := json.Marshal(map[string]any{
				"city": city,
				"days": days,
			})

			l.Info("[WeatherTool] Executing weather_forecast tool", "args", string(argsJSON))

			// Вызываем инструмент получения прогноза погоды
			toolResult, err := weatherTool.Call(ctx, string(argsJSON))
			if err != nil {
				l.Error("[WeatherTool] Tool call failed", "error", err)
				return nil, err
			}

			// Логируем результат работы инструмента
			l.Info("[WeatherTool] Tool completed", "result_preview", toolResult)

			// Возвращаем прогноз для следующего этапа
			return map[string]any{
				"forecast": toolResult,
			}, nil
		},
		[]string{"city", "days"},
		[]string{"forecast"},
	)
}

// convertToInt преобразует значение любого числового типа в int
func convertToInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int32:
		return int(val)
	case int64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

// createFinalResponseChain создаёт цепочку для генерации финального ответа
func createFinalResponseChain(llm llms.Model) *chains.LLMChain {
	// Создаём LLM-цепочку с шаблоном FinalWeatherPromptTemplate
	chain := chains.NewLLMChain(
		llm,
		FinalWeatherPromptTemplate,
	)
	// Устанавливаем имя выходного ключа
	chain.OutputKey = "text"
	return chain
}
