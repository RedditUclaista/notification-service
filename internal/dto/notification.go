package dto

import "github.com/RedditUclaista/notification-service/internal/entities"

type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type NotificationHistoryResponse struct {
	Data []entities.Notification `json:"data"`
	Meta Meta                  `json:"meta"`
}
