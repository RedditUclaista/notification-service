package entities

import "time"

type DeviceFCMToken struct {
	UserID      string    `json:"user_id"`
	FCMToken    string    `json:"fcm_token"`
	Platform    string    `json:"platform"`
	LastUpdated time.Time `json:"last_updated"`
}
