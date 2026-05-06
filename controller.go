package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"

	"weather-agent-mcp/agent"
)

const (
	weatherProcessRoute = "/weather/process"
)

// AgentRequest представляет запрос к агенту с произвольным промптом.
type AgentRequest struct {
	Prompt string `json:"prompt"` // Текстовый запрос пользователя
}

// llmRoutes хранит зависимости для обработки запросов к LLM.
type llmRoutes struct {
	l     *slog.Logger        // l — логгер для записи событий
	agent *agent.WeatherAgent // agent — агент для обработки запросов о погоде
}

// newLLMRoutes регистрирует все маршруты для работы с LLM.
func newLLMRoutes(api fiber.Router, l *slog.Logger, agent *agent.WeatherAgent) {
	// Инициализируем структуру с зависимостями
	lr := llmRoutes{l, agent}

	// Регистрируем обработчик
	api.Post(weatherProcessRoute, lr.ProcessWeather)
}

// ProcessWeather обрабатывает запрос о прогнозе погоды.
func (lr *llmRoutes) ProcessWeather(c fiber.Ctx) error {
	// Создаём контекст с таймаутом 60 секунд для получения прогноза
	ctx, cancel := context.WithTimeout(c.Context(), 60*time.Second)
	defer cancel()

	// Парсим JSON из тела запроса в структуру AgentRequest
	var req AgentRequest
	raw := c.BodyRaw()
	if err := json.Unmarshal(raw, &req); err != nil {
		// Возвращаем HTTP 400 при невалидном JSON
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Невалидный JSON",
		})
	}

	// Проверяем, что поле prompt не пустое
	if req.Prompt == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Поле 'prompt' обязательно",
		})
	}

	// Логируем полученный запрос о погоде с длиной промпта
	lr.l.Info("[ProcessWeather] Processing weather request", "prompt_length", len(req.Prompt))

	// Передаём запрос агенту для обработки
	response, err := lr.agent.ProcessWeather(ctx, req.Prompt)
	if err != nil {
		// Логируем ошибку обработки
		lr.l.Error("[ProcessWeather] Agent execution failed", "error", err)

		// Возвращаем HTTP 500 с описанием ошибки
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Ошибка обработки запроса: " + err.Error(),
		})
	}

	// Логируем успешную обработку с длиной ответа
	lr.l.Info("[ProcessWeather] Success", "response_length", len(response))

	// Отправляем ответ клиенту
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success":  true,
		"response": response,
		"message":  "Запрос успешно обработан",
	})
}
