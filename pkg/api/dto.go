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

	// Remote Control Messages
	MessageTypeRemoteControlRequest = "remote_control_request"
	MessageTypeSessionAccepted      = "session_accepted"
	MessageTypeSessionRejected      = "session_rejected"
	MessageTypeSessionStarted       = "session_started"
	MessageTypeSessionEnded         = "session_ended"
	MessageTypeSessionFailed        = "session_failed"

	// Screen Streaming Messages
	MessageTypeScreenFrame            = "screen_frame"
	MessageTypeInputCommand           = "input_command"
	MessageTypeVideoFrameUpload       = "video_frame_upload"
	MessageTypeVideoRecordingComplete = "video_recording_complete"

	// File Transfer Messages
	MessageTypeFileTransferRequest = "file_transfer_request"
	MessageTypeFileChunk           = "file_chunk"
	MessageTypeFileTransferAck     = "file_transfer_acknowledgement"
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

// Remote Control Messages
type RemoteControlRequest struct {
	SessionID     string `json:"session_id"`
	AdminUserID   string `json:"admin_user_id"`
	ClientPCID    string `json:"client_pc_id"`
	AdminUsername string `json:"admin_username,omitempty"`
}

type SessionAcceptedMessage struct {
	SessionID string `json:"session_id"`
}

type SessionRejectedMessage struct {
	SessionID string `json:"session_id"`
	Reason    string `json:"reason,omitempty"`
}

// ScreenFrame represents a captured screen frame
type ScreenFrame struct {
	SessionID   string `json:"session_id"`
	Timestamp   int64  `json:"timestamp"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Format      string `json:"format"`            // "jpeg", "png", etc.
	Quality     int    `json:"quality,omitempty"` // For JPEG compression (1-100)
	FrameData   []byte `json:"frame_data"`        // Base64 encoded image data
	SequenceNum int64  `json:"sequence_num"`
}

// InputCommand represents a remote input command (mouse/keyboard)
type InputCommand struct {
	SessionID string                 `json:"session_id"`
	Timestamp int64                  `json:"timestamp"`
	EventType string                 `json:"event_type"` // "mouse", "keyboard"
	Action    string                 `json:"action"`     // "move", "click", "scroll", "keydown", "keyup", "type"
	Payload   map[string]interface{} `json:"payload"`    // Event-specific data
}

// Mouse Event Payload Fields
type MouseEventPayload struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button,omitempty"` // "left", "right", "middle"
	Delta  int    `json:"delta,omitempty"`  // For scroll events
}

// Keyboard Event Payload Fields
type KeyboardEventPayload struct {
	Key       string   `json:"key"`                 // Key identifier
	Code      string   `json:"code,omitempty"`      // Physical key code
	Text      string   `json:"text,omitempty"`      // For typing text
	Modifiers []string `json:"modifiers,omitempty"` // ["ctrl", "alt", "shift", "meta"]
}

// VideoFrameUpload representa un frame de video individual para subir
type VideoFrameUpload struct {
	SessionID  string `json:"session_id"`
	VideoID    string `json:"video_id"`
	FrameIndex int    `json:"frame_index"`
	Timestamp  int64  `json:"timestamp"`
	FrameData  []byte `json:"frame_data"` // Base64 encoded JPEG data
}

// VideoRecordingCompletePayload contiene los metadatos de una grabación finalizada
type VideoRecordingCompletePayload struct {
	VideoID         string  `json:"video_id"`
	SessionID       string  `json:"session_id"`
	TotalFrames     int     `json:"total_frames"`
	FPS             float64 `json:"fps"`
	DurationSeconds float64 `json:"duration_seconds"`
	Timestamp       int64   `json:"timestamp"`
}

// FileTransferRequest represents a file transfer request from server to client
type FileTransferRequest struct {
	TransferID      string  `json:"transfer_id"`
	SessionID       string  `json:"session_id"`
	FileName        string  `json:"file_name"`
	FileSizeMB      float64 `json:"file_size_mb"`
	TotalChunks     int     `json:"total_chunks"`
	DestinationPath string  `json:"destination_path"`
	Timestamp       int64   `json:"timestamp"`
}

// FileChunk represents a chunk of file data being transferred
type FileChunk struct {
	TransferID    string `json:"transfer_id"`
	SessionID     string `json:"session_id"`
	ChunkIndex    int    `json:"chunk_index"`
	TotalChunks   int    `json:"total_chunks"`
	ChunkData     []byte `json:"chunk_data"`
	IsLastChunk   bool   `json:"is_last_chunk"`
	ChunkChecksum string `json:"chunk_checksum,omitempty"`
	Timestamp     int64  `json:"timestamp"`
}

// FileTransferAcknowledgement represents the client's response after receiving a file
type FileTransferAcknowledgement struct {
	TransferID   string `json:"transfer_id"`
	SessionID    string `json:"session_id"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	FileChecksum string `json:"file_checksum,omitempty"`
	Timestamp    int64  `json:"timestamp"`
}
