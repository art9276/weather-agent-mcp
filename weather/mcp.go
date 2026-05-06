package weather

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpSession хранит сессию MCP клиента
var (
	mcpSession     *mcp.ClientSession
	mcpSessionOnce sync.Once
	mcpSessionMu   sync.Mutex
)

// InitMCPSession инициализирует MCP сессию (один раз при старте приложения)
func InitMCPSession(ctx context.Context) error {
	var initErr error
	mcpSessionOnce.Do(func() {
		mcpSessionMu.Lock()
		defer mcpSessionMu.Unlock()

		// Создаём MCP клиента
		client := mcp.NewClient(
			&mcp.Implementation{Name: "weather-client", Version: "1.0.0"},
			nil,
		)

		// Подключаемся к MCP weather серверу через stdio
		// Сервер должен быть запущен отдельно: npx -y @dangahagan/weather-mcp@latest
		cmd := exec.Command("npx", "-y", "@dangahagan/weather-mcp@latest")
		transport := &mcp.CommandTransport{Command: cmd}

		var err error
		mcpSession, err = client.Connect(ctx, transport, nil)
		if err != nil {
			initErr = fmt.Errorf("failed to connect to MCP weather server: %w", err)
			return
		}
	})
	return initErr
}

// getMCPSession возвращает существующую MCP сессию
func getMCPSession() (*mcp.ClientSession, error) {
	mcpSessionMu.Lock()
	defer mcpSessionMu.Unlock()

	if mcpSession == nil {
		return nil, fmt.Errorf("MCP session not initialized. Call InitMCPSession first")
	}

	return mcpSession, nil
}

// CloseMCPSession закрывает MCP сессию (вызывать при остановке приложения)
func CloseMCPSession() error {
	mcpSessionMu.Lock()
	defer mcpSessionMu.Unlock()

	if mcpSession != nil {
		mcpSession.Close()
		mcpSession = nil
	}
	return nil
}

// fetchWeatherViaMCP получает прогноз погоды через MCP сервер
func fetchWeatherViaMCP(ctx context.Context, city string, days int) ([]DailyForecast, error) {
	session, err := getMCPSession()
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP session: %w", err)
	}

	// Сначала ищем координаты города через search_location
	searchParams := &mcp.CallToolParams{
		Name: "search_location",
		Arguments: map[string]any{
			"query": city,
			"limit": 1,
		},
	}

	searchResult, err := session.CallTool(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("search_location failed: %w", err)
	}

	if searchResult.IsError {
		return nil, fmt.Errorf("search_location returned error: %v", searchResult)
	}

	// Парсим результат поиска из Markdown формата
	lat, lon := parseLocationFromMarkdown(searchResult)

	if lat == 0 && lon == 0 {
		return nil, fmt.Errorf("city '%s' not found", city)
	}

	// Получаем прогноз через get_forecast
	forecastParams := &mcp.CallToolParams{
		Name: "get_forecast",
		Arguments: map[string]any{
			"latitude":    lat,
			"longitude":   lon,
			"days":        days,
			"granularity": "daily",
		},
	}

	forecastResult, err := session.CallTool(ctx, forecastParams)
	if err != nil {
		return nil, fmt.Errorf("get_forecast failed: %w", err)
	}

	if forecastResult.IsError {
		return nil, fmt.Errorf("get_forecast returned error: %v", forecastResult)
	}

	// Парсим результат прогноза
	return parseForecastResult(forecastResult, days)
}

// parseLocationFromMarkdown парсит координаты из Markdown вывода search_location
func parseLocationFromMarkdown(result *mcp.CallToolResult) (float64, float64) {
	var lat, lon float64

	if len(result.Content) > 0 {
		textContent, ok := result.Content[0].(*mcp.TextContent)
		if ok {
			// Ищем строку вида: *Latitude: 55.75222, Longitude: 37.61556*
			latLonRegex := regexp.MustCompile(`Latitude:\s*([+-]?\d+\.?\d*),\s*Longitude:\s*([+-]?\d+\.?\d*)`)
			matches := latLonRegex.FindStringSubmatch(textContent.Text)
			if len(matches) >= 3 {
				lat, _ = strconv.ParseFloat(matches[1], 64)
				lon, _ = strconv.ParseFloat(matches[2], 64)
			}
		}
	}

	return lat, lon
}

// parseForecastResult парсит результат MCP get_forecast в нашу структуру
func parseForecastResult(result *mcp.CallToolResult, days int) ([]DailyForecast, error) {
	forecasts := make([]DailyForecast, 0, days)

	for _, content := range result.Content {
		textContent, ok := content.(*mcp.TextContent)
		if !ok {
			continue
		}

		// Парсим Markdown формат MCP weather сервера
		parsed := parseWeatherMarkdown(textContent.Text, days)
		if len(parsed) > 0 {
			forecasts = parsed
			break
		}
	}

	if len(forecasts) == 0 {
		return nil, fmt.Errorf("failed to parse forecast data")
	}

	return forecasts, nil
}

// parseWeatherMarkdown парсит Markdown вывод MCP weather сервера
func parseWeatherMarkdown(text string, days int) []DailyForecast {
	forecasts := make([]DailyForecast, 0, days)

	// Разбиваем на строки
	lines := strings.Split(text, "\n")
	var currentForecast *DailyForecast

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Ищем заголовок дня (## день недели, дата)
		// Формат: "## Friday, 20 March" или "## пятница, 20 марта"
		if strings.HasPrefix(line, "## ") {
			if currentForecast != nil {
				forecasts = append(forecasts, *currentForecast)
				if len(forecasts) >= days {
					break
				}
			}
			currentForecast = &DailyForecast{}

			// Извлекаем дату из строки "## Friday, 20 March"
			dateStr := strings.TrimPrefix(line, "## ")
			// Удаляем жирный текст если есть
			dateStr = strings.TrimSpace(dateStr)
			currentForecast.Date = parseRussianDate(dateStr)
			continue
		}

		if currentForecast == nil {
			continue
		}

		// Парсим температуру: **Temperature:** High 49°F / Low 27°F
		if strings.Contains(line, "**Temperature:**") {
			tempMatch := regexp.MustCompile(`High\s+(-?\d+)°F\s*/\s*Low\s+(-?\d+)°F`).FindStringSubmatch(line)
			if len(tempMatch) == 3 {
				high, _ := strconv.ParseFloat(tempMatch[1], 64)
				low, _ := strconv.ParseFloat(tempMatch[2], 64)
				// Конвертируем из Fahrenheit в Celsius
				currentForecast.Temperature = (high + low) / 2.0 * 5.0 / 9.0
			}
		}

		// Парсим условия: **Conditions:** Overcast
		if strings.Contains(line, "**Conditions:**") {
			parts := strings.SplitN(line, "**Conditions:**", 2)
			if len(parts) > 1 {
				condition := strings.TrimSpace(parts[1])
				condition = strings.Trim(condition, "*")
				currentForecast.Condition = translateCondition(condition)
			}
		}

		// Парсим влажность (если есть в выводе)
		if strings.Contains(line, "**Humidity:**") || strings.Contains(line, "**Relative Humidity:**") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				humidityStr := strings.TrimSpace(parts[1])
				humidityStr = strings.Trim(humidityStr, "%*")
				if h, err := strconv.Atoi(humidityStr); err == nil {
					currentForecast.Humidity = h
				}
			}
		}

		// Парсим ветер: **Wind:** 4 mph SSW
		if strings.Contains(line, "**Wind:**") {
			windMatch := regexp.MustCompile(`Wind:\*\*\s+(-?\d+(?:\.\d+)?)\s+mph`).FindStringSubmatch(line)
			if len(windMatch) > 1 {
				windSpeed, _ := strconv.ParseFloat(windMatch[1], 64)
				// Конвертируем из mph в km/h
				currentForecast.WindSpeed = windSpeed * 1.60934
			}
		}
	}

	// Добавляем последний прогноз
	if currentForecast != nil && len(forecasts) < days {
		forecasts = append(forecasts, *currentForecast)
	}

	return forecasts
}

// parseRussianDate преобразует дату в формате "пятница, 20 марта" или "Friday, 20 March" в YYYY-MM-DD
func parseRussianDate(dateStr string) string {
	// "пятница, 20 марта" или "Friday, 20 March" -> "2026-03-20"
	parts := strings.Split(dateStr, ", ")
	if len(parts) != 2 {
		return time.Now().Format("2006-01-02")
	}

	dayMonth := strings.Split(parts[1], " ")
	if len(dayMonth) != 2 {
		return time.Now().Format("2006-01-02")
	}

	day, _ := strconv.Atoi(dayMonth[0])
	monthStr := strings.ToLower(dayMonth[1])

	// Маппинг русских месяцев
	months := map[string]int{
		"января": 1, "февраля": 2, "марта": 3, "апреля": 4,
		"мая": 5, "июня": 6, "июля": 7, "августа": 8,
		"сентября": 9, "октября": 10, "ноября": 11, "декабря": 12,
		// Английские месяцы
		"january": 1, "february": 2, "march": 3, "april": 4,
		"may": 5, "june": 6, "july": 7, "august": 8,
		"september": 9, "october": 10, "november": 11, "december": 12,
	}

	month, ok := months[monthStr]
	if !ok {
		return time.Now().Format("2006-01-02")
	}

	year := time.Now().Year()
	// Если месяц прошёл, значит следующий год
	if month < int(time.Now().Month()) {
		year++
	}

	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

// translateCondition переводит условия погоды с английского на русский
func translateCondition(condition string) string {
	translations := map[string]string{
		"Mainly clear":  "Солнечно",
		"Clear":         "Солнечно",
		"Sunny":         "Солнечно",
		"Partly cloudy": "Облачно",
		"Overcast":      "Облачно",
		"Cloudy":        "Облачно",
		"Foggy":         "Туманно",
		"Mist":          "Туман",
		"Light rain":    "Дождливо",
		"Rain":          "Дождливо",
		"Rainy":         "Дождливо",
		"Thunderstorm":  "Гроза",
		"Light snow":    "Снег",
		"Snow":          "Снег",
		"Snowy":         "Снег",
		"Windy":         "Ветер",
	}

	if translated, ok := translations[condition]; ok {
		return translated
	}
	return "Облачно"
}
