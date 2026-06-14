package dto

import "github.com/RedditUclaista/notification-service/internal/entities"

type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type NotificationData struct {
	Notifications []entities.Notification `json:"notifications"`
}

type NotificationHistoryResponse struct {
	Data NotificationData `json:"data"`
	Meta Meta             `json:"meta"`
}
