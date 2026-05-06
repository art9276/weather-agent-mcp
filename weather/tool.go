package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/tmc/langchaingo/llms"
)

// WeatherForecastTool — инструмент для получения прогноза погоды через MCP.
// Здесь мы реализуем интерфейс Tool и добавляем методы Call, Name и Description
// Tool подключается к внешнему MCP-серверу (@dangahagan/weather-mcp) для получения реальных данных.
// Возвращает прогноз по дням с температурой, условиями, влажностью и скоростью ветра.
type WeatherForecastTool struct {
}

// NewWeatherForecastTool создаёт новый экземпляр инструмента прогноза погоды.
func NewWeatherForecastTool() *WeatherForecastTool {
	return &WeatherForecastTool{}
}

// Call выполняет запрос к инструменту weather_forecast.
func (w *WeatherForecastTool) Call(ctx context.Context, input string) (string, error) {
	// Генерируем уникальную сигнатуру инструмента для отслеживания
	// Включает временную метку (миллисекунды) и короткую версию UUID
	timestamp := time.Now().UnixMilli()
	shortUUID := uuid.New().String()[:8]
	toolSignature := fmt.Sprintf("WEATHER_TOOL_%d_%s", timestamp, shortUUID)

	// Парсим входной JSON для извлечения параметров запроса
	var args struct {
		City string `json:"City"` // Город для прогноза
		Days int    `json:"days"` // Количество дней
	}
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return "", fmt.Errorf("invalid input format: %v", err)
	}

	// Ограничиваем количество дней минимальным значением 1
	if args.Days <= 0 {
		args.Days = 1
	}

	// Ограничиваем максимальное количество дней значением 16
	// Это ограничение MCP-сервера
	if args.Days > 16 {
		args.Days = 16
	}

	// Получаем прогноз погоды через MCP-сессию
	// fetchWeatherViaMCP подключается к внешнему weather-серверу
	forecasts, err := fetchWeatherViaMCP(ctx, args.City, args.Days)
	if err != nil {
		return "", fmt.Errorf("failed to get weather forecast: %w", err)
	}

	// Формируем структуру ответа с прогнозом
	weatherResponse := WeatherForecast{
		City:     args.City, // Город, для которого получен прогноз
		Days:     args.Days, // Количество дней прогноза
		Forecast: forecasts, // Массив ежедневных прогнозов
	}

	// Создаём карту для добавления метаданных инструмента
	jsonMap := make(map[string]any)

	// Маршалим основной ответ в JSON-байты
	bytes, _ := json.Marshal(weatherResponse)

	// Размаршалим обратно в карту для добавления полей
	json.Unmarshal(bytes, &jsonMap) // Игнорируем ошибку, данные уже валидны

	// Добавляем сигнатуру и временную метку инструмента
	jsonMap["_tool_sig"] = toolSignature
	jsonMap["_tool_ts"] = timestamp

	// Маршалим финальный результат с метаданными
	finalBytes, err := json.Marshal(jsonMap)
	if err != nil {
		// При ошибке возвращаем основной ответ без метаданных
		return string(bytes), nil
	}

	return string(finalBytes), nil
}

// Name возвращает название инструмента.
func (w *WeatherForecastTool) Name() string {
	return "weather_forecast"
}

// Description возвращает описание инструмента.
func (w *WeatherForecastTool) Description() string {
	return "Получить прогноз погоды на указанное количество дней для заданного города. Входные данные: JSON с полями City (город) и days (количество дней, от 1 до 16). Выходные данные: JSON с прогнозом по дням, включая температуру, условия, влажность и скорость ветра."
}

// generateForecasts получает прогноз погоды через MCP-сервер.
func generateForecasts(ctx context.Context, city string, days int) ([]DailyForecast, error) {
	return fetchWeatherViaMCP(ctx, city, days)
}

// AvailableTools - инструменты, доступные для weather-агента
var AvailableTools = []llms.Tool{
	{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "weather_forecast",
			Description: "Получить прогноз погоды на указанное количество дней для заданного города. Входные данные: JSON с полями City (город) и days (количество дней, от 1 до 10). Выходные данные: JSON с прогнозом по дням, включая температуру, условия, влажность и скорость ветра.",
			Parameters:  nil,
		},
	},
}

// InitTools инициализирует параметры инструментов weather-агента.
func InitTools() {
	AvailableTools[0].Function.Parameters = []byte(`{
		"type": "object",
		"properties": {
			"city": {
				"type": "string",
				"description": "Город, для которого нужен прогноз"
			},
			"days": {
				"type": "integer",
				"description": "Количество дней для прогноза (от 1 до 16)",
				"minimum": 1,
				"maximum": 16
			}
		},
		"required": ["city"]
	}`)
}
