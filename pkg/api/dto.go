package api

// DTOs para la comunicación con el servidor

// AuthResultDTO representa el resultado de la autenticación
type AuthResultDTO struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PCRegistrationResultDTO representa el resultado del registro del PC
type PCRegistrationResultDTO struct {
	Success bool   `json:"success"`
	PCID    string `json:"pcId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// WebSocket Message Types
const (
	MessageTypeClientAuth         = "CLIENT_AUTH_REQUEST"
	MessageTypeClientAuthResp     = "CLIENT_AUTH_RESPONSE"
	MessageTypePCRegistration     = "PC_REGISTRATION_REQUEST"
	MessageTypePCRegistrationResp = "PC_REGISTRATION_RESPONSE"
	MessageTypeHeartbeat          = "HEARTBEAT"
	MessageTypeHeartbeatResp      = "HEARTBEAT_RESPONSE"
)

// Base message structure
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Client Authentication Messages
type ClientAuthRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ClientAuthResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"`
	UserID  string `json:"userId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// PC Registration Messages
type PCRegistrationRequest struct {
	PCIdentifier string `json:"pcIdentifier"`
	IP           string `json:"ip,omitempty"` // Optional, can be detected from connection
}

type PCRegistrationResponse struct {
	Success bool   `json:"success"`
	PCID    string `json:"pcId,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Heartbeat Messages
type HeartbeatRequest struct {
	Timestamp int64 `json:"timestamp"`
}

type HeartbeatResponse struct {
	Timestamp int64  `json:"timestamp"`
	Status    string `json:"status"`
}
