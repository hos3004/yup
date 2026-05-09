package model

import "time"

type User struct {
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name,omitempty"`
	AuthToken   string    `json:"auth_token,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Device struct {
	DeviceID       string    `json:"device_id"`
	UserID         string    `json:"user_id"`
	PublicCurveKey string    `json:"public_curve_key"`
	PublicEdKey    string    `json:"public_ed_key"`
	CreatedAt      time.Time `json:"created_at"`
}

type KeyBundle struct {
	DeviceID    string   `json:"device_id"`
	CurveKey    string   `json:"curve_key"`
	EdKey       string   `json:"ed_key"`
	OneTimeKeys []string `json:"one_time_keys"`
	Signature   string   `json:"signature,omitempty"`
}

type Message struct {
	ID             string     `json:"id"`
	SenderUsername string     `json:"sender_username"`
	RecipientUser  string     `json:"recipient_user"`
	Ciphertext     string     `json:"ciphertext"`
	MessageType    int        `json:"message_type"`
	SenderCurveKey string     `json:"sender_curve_key"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
}

type Envelope struct {
	ID             string    `json:"id"`
	SenderUsername string    `json:"sender_username"`
	Ciphertext     string    `json:"ciphertext"`
	MessageType    int       `json:"message_type"`
	SenderCurveKey string    `json:"sender_curve_key"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type DeviceToken struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeviceTokenRequest is the request format for POST /api/v1/devices
type DeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// KeyBundleResponse is the response format for GET /keys/{username}
type KeyBundleResponse struct {
	DeviceID      string   `json:"device_id"`
	CurveKey      string   `json:"curve_key"`
	EdKey         string   `json:"ed_key"`
	OneTimeKeys   []string `json:"one_time_keys"`
	NoOtkAvailable bool    `json:"no_otk_available,omitempty"`
}
