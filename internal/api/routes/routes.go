package routes

import (
	"rize-api/internal/api/handlers"
	"rize-api/internal/api/middleware"

	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	Auth      *handlers.AuthHandler
	Session   *handlers.SessionHandler
	Dashboard *handlers.DashboardHandler
	Report    *handlers.ReportHandler
	Settings  *handlers.SettingsHandler
}

func Register(app *fiber.App, h Handlers, jwtSecret string) {
	// Auth — public
	auth := app.Group("/auth")
	auth.Post("/magic", h.Auth.SendMagicLink)
	auth.Get("/verify", h.Auth.VerifyMagicLink)
	auth.Post("/logout", h.Auth.Logout)
	auth.Get("/me", middleware.Auth(jwtSecret), h.Auth.Me)

	// Protected routes
	protected := app.Group("", middleware.Auth(jwtSecret))

	// Sit sessions
	sit := protected.Group("/sessions/sit")
	sit.Post("/start", h.Session.StartSit)
	sit.Post("/end", h.Session.EndSit)
	sit.Get("/active", h.Session.GetActiveSit)
	sit.Get("/today", h.Session.GetTodaySit)

	// Move sessions
	move := protected.Group("/sessions/move")
	move.Post("", h.Session.LogMove)
	move.Post("/start", h.Session.StartMove)
	move.Post("/:id/end", h.Session.EndMove)
	move.Get("/today", h.Session.GetTodayMove)
	move.Delete("/:id", h.Session.DeleteMove)

	// Dashboard
	dashboard := protected.Group("/dashboard")
	dashboard.Get("/today", h.Dashboard.Today)
	dashboard.Get("/score", h.Dashboard.Score)

	// Reports
	reports := protected.Group("/reports")
	reports.Get("/today", h.Report.Today)
	reports.Get("/history", h.Report.History)
	reports.Post("/generate", h.Report.Generate)

	// Settings
	settings := protected.Group("/settings")
	settings.Patch("", h.Settings.Update)
	settings.Post("/push", h.Settings.SavePush)
	settings.Delete("/push", h.Settings.DeletePush)
	settings.Get("/vapid-key", h.Settings.GetVAPIDKey)
}
