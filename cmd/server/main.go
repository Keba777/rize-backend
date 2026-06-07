package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"rize-api/internal/api/handlers"
	"rize-api/internal/api/routes"
	"rize-api/internal/config"
	"rize-api/internal/cron"
	"rize-api/internal/db"
	"rize-api/internal/repository"
	"rize-api/internal/services"
	"rize-api/pkg/mailer"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/joho/godotenv"
)

func main() {
	if _, err := os.Stat(".env"); err == nil {
		godotenv.Load() //nolint:errcheck
	}

	cfg := config.Load()

	pool, err := db.NewPool(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(pool); err != nil {
		log.Fatalf("db migrate: %v", err)
	}

	// Repositories
	userRepo := repository.NewUserRepository(pool)
	sitRepo := repository.NewSitRepository(pool)
	moveRepo := repository.NewMoveRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	pushRepo := repository.NewPushRepository(pool)

	// Services
	m := mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.FromEmail)
	frontendURL := strings.Split(cfg.AllowedOrigins, ",")[0]
	backendURL := fmt.Sprintf("http://localhost:%s", cfg.Port)

	authSvc := services.NewAuthService(userRepo, m, cfg.JWTSecret, cfg.JWTTTLHours, cfg.MagicLinkTTLMin, backendURL, frontendURL, cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	sessionSvc := services.NewSessionService(sitRepo, moveRepo)
	reportSvc := services.NewReportService(reportRepo, sitRepo, moveRepo, userRepo)
	pushSvc := services.NewPushService(pushRepo, userRepo, reportRepo, sitRepo, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDEmail)

	// Handlers
	h := routes.Handlers{
		Auth:      handlers.NewAuthHandler(authSvc, cfg.JWTTTLHours, frontendURL),
		Session:   handlers.NewSessionHandler(sessionSvc),
		Dashboard: handlers.NewDashboardHandler(sitRepo, moveRepo, reportRepo, userRepo),
		Report:    handlers.NewReportHandler(reportSvc),
		Settings:  handlers.NewSettingsHandler(userRepo, pushSvc),
	}

	// Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowMethods:     "GET, POST, PATCH, DELETE, OPTIONS",
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Health
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	routes.Register(app, h, cfg.JWTSecret)

	// Cron
	cron.Setup(reportSvc, pushSvc)

	log.Printf("rize-api listening on :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
