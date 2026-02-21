package domain

type TokenRequest struct {
	DeviceID string `json:"device_id"`
}

type TokenResponse struct {
	Token string `json:"token"`
}
