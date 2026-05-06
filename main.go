package main

import (
	"context"
	"fmt"
	"time"

	"weather-agent-mcp/agent"
	"weather-agent-mcp/weather"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/compress"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

var acessToken string

func main() {
	//инициализируем логгер
	l := newLogger(0, true)
	//получаем переменные для работы приложения
	token, ip, port, _, err := getConfig()
	if err != nil {
		l.Error("не удалось получить токен авторизации!", "code", 404)
		panic(err)
	}
	// получаем acessToken
	acessToken, err := getToken(token)
	if err != nil {
		l.Error("не удалось получить токен доступа!", "code", 404)
		//	panic(err)
	}
	// подключаемся к LLM
	llm, err := getGigaChatLLM(acessToken)
	if err != nil {
		l.Error("не удалось подключиться к ллм модели!", "code", 500)
	}

	// Инициализируем инструменты и шаблоны weather-пакета
	weather.InitTools()
	weather.InitTemplates()

	// Инициализируем MCP сессию для weather инструмента
	ctx := context.Background()
	if err := weather.InitMCPSession(ctx); err != nil {
		l.Warn("не удалось инициализировать MCP weather сессию", "error", err)
		l.Info("weather инструмент будет недоступен")
	} else {
		l.Info("MCP weather сессия успешно инициализирована")
		// Регистрируем закрытие MCP сессии при остановке
		defer func() {
			if err := weather.CloseMCPSession(); err != nil {
				l.Error("ошибка закрытия MCP сессии", "error", err)
			}
		}()
	}

	// Создаём агента
	agentInstance := agent.NewWeatherAgent(llm, l)
	app := fiber.New(fiber.Config{
		AppName:        "llm Service",
		ServerHeader:   "llm Service", // добавляем заголовок для идентификации сервера
		CaseSensitive:  true,          // включаем чувствительность к регистру в URL
		StrictRouting:  true,
		RequestMethods: []string{"POST"}, // включаем строгую маршрутизацию
	})
	api := app.Group("/api") // /api
	//не даем падать нашему сервису при панике
	api.Use(recover.New())
	api.Use(cors.New(cors.Config{
		AllowHeaders: []string{"Origin, Content-Type, Accept, Authorization"},
		AllowMethods: []string{"POST"},
	}))
	api.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed, // 1
	}))
	// Передаём агент в роуты
	newLLMRoutes(api, l, agentInstance)
	routes := app.GetRoutes()
	for _, route := range routes {
		fmt.Printf("%s %s\n", route.Method, route.Path)
	}
	t := 3 * time.Second

	err = serveServer(app, ip, port, t, l)
	if err != nil {
		l.Error("Server ListenAndServe error")
		panic(err)
	}
}
