package http

import (
	"github.com/labstack/echo/v5"
)

func SetupRoutes(app *echo.Echo, handler *NotificationHandler) {
	api := app.Group("/api/v1/notifications")

	api.POST("/devices", handler.RegisterDevice)
	api.GET("", handler.GetNotifications)
	api.PATCH("/:id/read", handler.MarkAsRead)
}
