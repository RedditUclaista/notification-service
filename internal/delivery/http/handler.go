package http

import (
	"net/http"
	"strconv"

	"github.com/RedditUclaista/notification-service/internal/dto"
	"github.com/RedditUclaista/notification-service/internal/usecases"
	"github.com/labstack/echo/v5"
)

type NotificationHandler struct {
	useCase usecases.NotificationUseCase
}

func NewNotificationHandler(uc usecases.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{
		useCase: uc,
	}
}

func (h *NotificationHandler) RegisterDevice(c *echo.Context) error {
	req := new(dto.DeviceRegisterRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid JSON format"})
	}

	if req.UserID == "" || req.FCMToken == "" || req.Platform == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing required fields (user_id, fcm_token, platform)"})
	}

	err := h.useCase.RegisterDevice(c.Request().Context(), req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	res := dto.DeviceRegisterResponse{
		Message: "device registered successfully",
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *NotificationHandler) GetNotifications(c *echo.Context) error {
	// Extraemos el userID autenticado desde la cabecera 'X-User-Id' o del query param 'user_id'
	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		userID = c.QueryParam("user_id")
	}

	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing authenticated user ID in 'X-User-Id' header or 'user_id' query parameter"})
	}

	// Paginación
	limitStr := c.QueryParam("limit")
	limit := 10
	if limitStr != "" {
		if val, err := strconv.Atoi(limitStr); err == nil && val > 0 {
			limit = val
		}
	}

	offsetStr := c.QueryParam("offset")
	offset := 0
	if offsetStr != "" {
		if val, err := strconv.Atoi(offsetStr); err == nil && val >= 0 {
			offset = val
		}
	}

	res, err := h.useCase.GetNotifications(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, res)
}

func (h *NotificationHandler) MarkAsRead(c *echo.Context) error {
	notificationID := c.Param("id")
	if notificationID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing notification ID"})
	}

	// Extraemos el userID autenticado desde la cabecera 'X-User-Id' o del query param 'user_id'
	userID := c.Request().Header.Get("X-User-Id")
	if userID == "" {
		userID = c.QueryParam("user_id")
	}

	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Missing authenticated user ID in 'X-User-Id' header or 'user_id' query parameter"})
	}

	err := h.useCase.MarkAsRead(c.Request().Context(), notificationID, userID)
	if err != nil {
		// Retornamos un 403 Forbidden o 404 para representar que no coincide con el dueño
		return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "notification marked as read"})
}
