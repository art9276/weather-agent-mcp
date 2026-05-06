package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
	"weather-agent-mcp/weather"

	"github.com/gofiber/fiber/v3"
)

var defaultSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
}

func serveServer(app *fiber.App, address string, port string, period time.Duration, l Logger) error {
	signals := defaultSignals
	addr := address + ":" + port
	err := graceStartApp(
		app,
		addr,
		signals,
		period,
		l,
	)
	if err != nil {
		l.Error("Server failed", "error", err)
		return err
	}
	return nil
}

func graceStartApp(
	app *fiber.App,
	addr string,
	signals []os.Signal,
	shutdownTimeout time.Duration,
	l Logger,
) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)

	serverErr := make(chan error, 1)
	go func() {
		l.Info("Starting server on " + addr)
		if err := app.Listen(addr); err != nil {
			serverErr <- err
		}
	}()

	// Ждём либо ошибку запуска, либо сигнал завершения
	select {
	case err := <-serverErr:
		// Сервер не смог запуститься
		l.Error("Server failed to start", "error", err)
		return err
	case sig := <-c:
		l.Info("Received signal, shutting down gracefully", "signal", sig)
		// Выполняем graceful shutdown
		weather.CloseMCPSession() // закрываем сессию с mcp
		if err := app.ShutdownWithTimeout(shutdownTimeout); err != nil {
			l.Warn("Server shutdown error", "error", err)
			return err
		}
		l.Info("Server stopped gracefully")
		return nil
	}
}
