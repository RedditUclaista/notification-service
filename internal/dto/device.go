package dto

type DeviceRegisterRequest struct {
	UserID   string `json:"user_id"`
	FCMToken string `json:"fcm_token"`
	Platform string `json:"platform"`
}

type DeviceRegisterResponse struct {
	Message string `json:"message"`
}
