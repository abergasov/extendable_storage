package routes

import (
	"extendable_storage/internal/logger"
	"extendable_storage/internal/service/receiver"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type Server struct {
	appAddr    string
	log        logger.AppLogger
	service    receiver.DataReceiver
	httpEngine *fiber.App
}

// InitAppRouter initializes the HTTP Server.
func InitAppRouter(log logger.AppLogger, service receiver.DataReceiver, address string) *Server {
	app := &Server{
		appAddr:    address,
		httpEngine: fiber.New(fiber.Config{}),
		service:    service,
		log:        log.With(slog.String("service", "http")),
	}
	app.httpEngine.Use(recover.New())
	app.initRoutes()
	return app
}

func (s *Server) initRoutes() {
	s.httpEngine.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.SendString("pong")
	})
}

// Run starts the HTTP Server.
func (s *Server) Run() error {
	s.log.Info("Starting HTTP server", slog.String("port", s.appAddr))
	return s.httpEngine.Listen(s.appAddr)
}

func (s *Server) Stop() error {
	return s.httpEngine.Shutdown()
}
