package agent

import (
	"context"

	"weather-agent-mcp/weather"

	"github.com/tmc/langchaingo/llms"
)

// интерфейс для логирования.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// основная структура агента для обработки запросов о погоде.
type WeatherAgent struct {
	llm    llms.Model                   // языковая модель
	tool   *weather.WeatherForecastTool // инструмент для работы агента
	logger Logger                       // логгер
}

// NewWeatherAgent создаёт и инициализирует новый экземпляр WeatherAgent.
func NewWeatherAgent(llm llms.Model, l Logger) *WeatherAgent {
	l.Info("[WeatherAgent] Agent initialized")
	weatherTool := weather.NewWeatherForecastTool()
	return &WeatherAgent{
		llm:    llm,
		tool:   weatherTool,
		logger: l,
	}
}

// ProcessWeather обрабатывает запрос о прогнозе погоды.
func (wa *WeatherAgent) ProcessWeather(ctx context.Context, input string) (string, error) {
	wa.logger.Info("[WeatherAgent] Starting weather processing", "input_preview", input)
	// отдаем на обработку промта цепочкой агента
	result, err := weather.HandleWeatherRequest(ctx, wa.llm, wa.tool, wa.logger, input)
	if err != nil {
		wa.logger.Error("[WeatherAgent] Weather processing failed", "error", err)
		return "", err
	}

	wa.logger.Info("[WeatherAgent] Weather processing completed successfully")
	return result, nil
}
