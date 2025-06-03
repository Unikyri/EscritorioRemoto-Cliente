package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// RemoteControlRequestHandler es el callback para manejar solicitudes de control remoto
type RemoteControlRequestHandler func(request RemoteControlRequest)

// SessionEventHandler es el callback para manejar eventos de sesión
type SessionEventHandler func(eventType string, data interface{})

// InputCommandHandler es el callback para manejar comandos de input entrantes
type InputCommandHandler func(command InputCommand)

// FileTransferRequestHandler es el callback para manejar solicitudes de transferencia de archivos
type FileTransferRequestHandler func(request FileTransferRequest)

// FileChunkHandler es el callback para manejar chunks de archivos recibidos
type FileChunkHandler func(chunk FileChunk)

// APIClient maneja la comunicación WebSocket con el servidor
type APIClient struct {
	serverURL   string
	conn        *websocket.Conn
	isConnected bool
	mutex       sync.RWMutex
	writeMutex  sync.Mutex // Mutex dedicado para escrituras al WebSocket

	// Channels para manejar respuestas
	authResponse chan ClientAuthResponse
	regResponse  chan PCRegistrationResponse

	// Handler para solicitudes de control remoto
	remoteControlHandler RemoteControlRequestHandler

	// Handler para eventos de sesión
	sessionEventHandler SessionEventHandler

	// Handler para comandos de input entrantes
	inputCommandHandler InputCommandHandler

	// Handlers para transferencia de archivos
	fileTransferRequestHandler FileTransferRequestHandler
	fileChunkHandler           FileChunkHandler

	// Configuración
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// NewAPIClient crea un nuevo cliente API
func NewAPIClient(serverURL string) *APIClient {
	return &APIClient{
		serverURL:      serverURL,
		authResponse:   make(chan ClientAuthResponse, 1),
		regResponse:    make(chan PCRegistrationResponse, 1),
		connectTimeout: 10 * time.Second,
		readTimeout:    90 * time.Second,
		writeTimeout:   10 * time.Second,
	}
}

// SetRemoteControlHandler establece el handler para solicitudes de control remoto
func (c *APIClient) SetRemoteControlHandler(handler RemoteControlRequestHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.remoteControlHandler = handler
}

// SetSessionEventHandler establece el handler para eventos de sesión
func (c *APIClient) SetSessionEventHandler(handler SessionEventHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.sessionEventHandler = handler
}

// SetInputCommandHandler establece el handler para comandos de input entrantes
func (c *APIClient) SetInputCommandHandler(handler InputCommandHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.inputCommandHandler = handler
}

// SetFileTransferRequestHandler establece el handler para solicitudes de transferencia de archivos
func (c *APIClient) SetFileTransferRequestHandler(handler FileTransferRequestHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.fileTransferRequestHandler = handler
}

// SetFileChunkHandler establece el handler para chunks de archivos recibidos
func (c *APIClient) SetFileChunkHandler(handler FileChunkHandler) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.fileChunkHandler = handler
}

// Connect establece la conexión WebSocket con el servidor
func (c *APIClient) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isConnected {
		return nil // Ya conectado
	}

	// Parsear URL
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	// Establecer esquema WebSocket
	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	// Agregar path del WebSocket
	u.Path = "/ws/client"

	log.Printf("Connecting to WebSocket: %s", u.String())

	// Conectar con timeout
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = c.connectTimeout

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true

	// Iniciar goroutine para leer mensajes
	go c.readMessages()

	// Iniciar keep-alive con pings
	go c.startKeepAlive()

	log.Println("WebSocket connection established")
	return nil
}

// Disconnect cierra la conexión WebSocket
func (c *APIClient) Disconnect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isConnected || c.conn == nil {
		return nil
	}

	// Enviar mensaje de cierre usando mutex de escritura
	c.writeMutex.Lock()
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	c.writeMutex.Unlock()

	if err != nil {
		log.Printf("Error sending close message: %v", err)
	}

	// Cerrar conexión
	c.conn.Close()
	c.conn = nil
	c.isConnected = false

	log.Println("WebSocket connection closed")
	return nil
}

// IsConnected verifica si está conectado
func (c *APIClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isConnected
}

// GetServerURL obtiene la URL del servidor
func (c *APIClient) GetServerURL() string {
	return c.serverURL
}

// ConnectAndAuthenticate establece conexión y autentica al usuario
func (c *APIClient) ConnectAndAuthenticate(username, password string) (*ClientAuthResponse, error) {
	// Conectar si no está conectado
	if !c.IsConnected() {
		if err := c.Connect(); err != nil {
			return nil, fmt.Errorf("failed to connect: %w", err)
		}
	}

	// Enviar solicitud de autenticación
	authReq := WebSocketMessage{
		Type: MessageTypeClientAuth,
		Data: ClientAuthRequest{
			Username: username,
			Password: password,
		},
	}

	if err := c.sendMessage(authReq); err != nil {
		return nil, fmt.Errorf("failed to send auth request: %w", err)
	}

	// Esperar respuesta con timeout
	select {
	case response := <-c.authResponse:
		return &response, nil
	case <-time.After(c.readTimeout):
		return nil, fmt.Errorf("authentication timeout")
	}
}

// RegisterPC registra el PC en el servidor
func (c *APIClient) RegisterPC(pcIdentifier string) (*PCRegistrationResponse, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("not connected to server")
	}

	// Obtener IP del cliente (por ahora usar 127.0.0.1 para desarrollo local)
	clientIP := "127.0.0.1"

	// Enviar solicitud de registro
	regReq := WebSocketMessage{
		Type: MessageTypePCRegistration,
		Data: PCRegistrationRequest{
			PCIdentifier: pcIdentifier,
			IP:           clientIP,
		},
	}

	if err := c.sendMessage(regReq); err != nil {
		return nil, fmt.Errorf("failed to send registration request: %w", err)
	}

	// Esperar respuesta con timeout
	select {
	case response := <-c.regResponse:
		return &response, nil
	case <-time.After(c.readTimeout):
		return nil, fmt.Errorf("registration timeout")
	}
}

// SendHeartbeat envía un heartbeat al servidor
func (c *APIClient) SendHeartbeat() error {
	c.mutex.RLock()
	connected := c.isConnected
	c.mutex.RUnlock()

	if !connected {
		return fmt.Errorf("not connected to server")
	}

	hbReq := WebSocketMessage{
		Type: MessageTypeHeartbeat,
		Data: HeartbeatRequest{
			Timestamp: time.Now().Unix(),
		},
	}

	err := c.sendMessage(hbReq)
	if err != nil {
		// Solo marcar como desconectado si es un error grave de WebSocket
		if websocket.IsCloseError(err, websocket.CloseAbnormalClosure, websocket.CloseGoingAway) {
			c.mutex.Lock()
			c.isConnected = false
			if c.conn != nil {
				c.conn.Close()
				c.conn = nil
			}
			c.mutex.Unlock()
			log.Printf("Heartbeat failed with close error, marking as disconnected: %v", err)
		} else {
			// Para otros errores, solo logear pero no desconectar
			log.Printf("Heartbeat failed but connection may still be active: %v", err)
		}
		return err
	}

	return nil
}

// sendMessage envía un mensaje WebSocket
func (c *APIClient) sendMessage(message WebSocketMessage) error {
	// Usar mutex de lectura para verificar estado
	c.mutex.RLock()
	if !c.isConnected || c.conn == nil {
		c.mutex.RUnlock()
		return fmt.Errorf("not connected")
	}
	conn := c.conn
	c.mutex.RUnlock()

	// Usar mutex de escritura para operaciones de escritura
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	// Establecer timeout de escritura
	conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))

	return conn.WriteJSON(message)
}

// readMessages lee mensajes del WebSocket en un goroutine
func (c *APIClient) readMessages() {
	defer func() {
		c.mutex.Lock()
		c.isConnected = false
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		c.mutex.Unlock()
		log.Println("WebSocket read goroutine ended")
	}()

	// Configurar ping handler para mantener conexión viva
	c.conn.SetPingHandler(func(appData string) error {
		log.Println("Received ping, sending pong")

		// Usar mutex de escritura para la respuesta pong
		c.writeMutex.Lock()
		defer c.writeMutex.Unlock()

		return c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})

	consecutiveErrors := 0
	maxConsecutiveErrors := 3

	for {
		c.mutex.RLock()
		conn := c.conn
		connected := c.isConnected
		c.mutex.RUnlock()

		if !connected || conn == nil {
			break
		}

		// Establecer timeout de lectura más largo
		conn.SetReadDeadline(time.Now().Add(c.readTimeout))

		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			consecutiveErrors++

			// Verificar si la conexión está cerrada definitivamente
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
				break
			} else if netErr, ok := err.(*websocket.CloseError); ok {
				log.Printf("WebSocket closed: %v", netErr)
				break
			} else if consecutiveErrors >= maxConsecutiveErrors {
				// Si hay muchos errores consecutivos, cerrar la conexión
				log.Printf("Too many consecutive WebSocket errors (%d), closing connection. Last error: %v", consecutiveErrors, err)
				break
			} else {
				// Para timeouts y otros errores temporales, continuar pero con límite
				log.Printf("WebSocket read timeout or temporary error (%d/%d): %v", consecutiveErrors, maxConsecutiveErrors, err)
				time.Sleep(time.Duration(consecutiveErrors) * time.Second) // Backoff progresivo
				continue
			}
		}

		// Reset contador de errores si se lee exitosamente
		consecutiveErrors = 0

		// Procesar mensaje
		c.handleMessage(message)
	}
}

// handleMessage procesa los mensajes recibidos
func (c *APIClient) handleMessage(message WebSocketMessage) {
	log.Printf("🔍 DEBUG: Received message type: %s", message.Type)

	switch message.Type {
	case MessageTypeClientAuthResp:
		var response ClientAuthResponse
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &response); err == nil {
				select {
				case c.authResponse <- response:
				default:
					log.Println("Auth response channel full, dropping message")
				}
			}
		}

	case MessageTypePCRegistrationResp:
		var response PCRegistrationResponse
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &response); err == nil {
				select {
				case c.regResponse <- response:
				default:
					log.Println("Registration response channel full, dropping message")
				}
			}
		}

	case MessageTypeHeartbeatResp:
		// Heartbeat response, no action needed
		log.Println("Heartbeat response received")

	case MessageTypeRemoteControlRequest:
		log.Printf("🔍 DEBUG: Processing remote control request")
		// Manejar solicitud de control remoto
		var request RemoteControlRequest
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &request); err == nil {
				c.mutex.RLock()
				handler := c.remoteControlHandler
				c.mutex.RUnlock()

				if handler != nil {
					log.Printf("Received remote control request from admin: %s (Session: %s)",
						request.AdminUsername, request.SessionID)
					handler(request)
				} else {
					log.Println("Received remote control request but no handler set")
				}
			} else {
				log.Printf("Failed to unmarshal remote control request: %v", err)
			}
		} else {
			log.Printf("Failed to marshal remote control request data: %v", err)
		}

	case MessageTypeSessionStarted:
		log.Printf("🔍 DEBUG: Processing session started event")
		log.Println("Remote control session started")
		// Emitir evento para activar indicador UI
		var sessionData map[string]interface{}
		if data, err := json.Marshal(message.Data); err == nil {
			json.Unmarshal(data, &sessionData)
		}

		// Crear handler que emita evento a través de callback
		c.emitSessionEvent("session_started", sessionData)

	case MessageTypeSessionEnded, "control_session_ended":
		log.Printf("🔍 DEBUG: Processing session ended event - type: %s", message.Type)
		if message.Type == MessageTypeSessionEnded {
			log.Println("Remote control session ended")
		} else {
			log.Println("Remote control session ended by admin")
		}

		log.Printf("🔍 DEBUG: Checking session event handler...")
		c.mutex.RLock()
		hasHandler := c.sessionEventHandler != nil
		c.mutex.RUnlock()
		log.Printf("🔍 DEBUG: Session event handler available: %v", hasHandler)

		// Emitir evento para desactivar indicador UI
		var sessionData map[string]interface{}
		if data, err := json.Marshal(message.Data); err == nil {
			json.Unmarshal(data, &sessionData)
			log.Printf("🔍 DEBUG: Session data: %+v", sessionData)
		}

		// Crear handler que emita evento a través de callback
		log.Printf("🔍 DEBUG: Calling emitSessionEvent with 'session_ended'")
		c.emitSessionEvent("session_ended", sessionData)

	case MessageTypeSessionFailed:
		log.Printf("🔍 DEBUG: Processing session failed event")
		log.Println("Remote control session failed")
		// Emitir evento para manejar fallo de sesión
		var sessionData map[string]interface{}
		if data, err := json.Marshal(message.Data); err == nil {
			json.Unmarshal(data, &sessionData)
		}

		// Crear handler que emita evento a través de callback
		c.emitSessionEvent("session_failed", sessionData)

	case MessageTypeInputCommand:
		log.Printf("🔍 DEBUG: Processing input command")
		// Manejar comando de input entrante
		var command InputCommand
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &command); err == nil {
				c.mutex.RLock()
				handler := c.inputCommandHandler
				c.mutex.RUnlock()

				if handler != nil {
					log.Printf("🎮 Received input command: type=%s, action=%s",
						command.EventType, command.Action)
					handler(command)
				} else {
					log.Println("Received input command but no handler set")
				}
			} else {
				log.Printf("Failed to unmarshal input command: %v", err)
			}
		} else {
			log.Printf("Failed to marshal input command data: %v", err)
		}

	case "video_recording_finalized":
		log.Printf("🔍 DEBUG: Processing video recording finalized confirmation")
		// Confirmación del backend de que la grabación fue procesada exitosamente
		log.Println("✅ Video recording finalized confirmation received from backend")

	case MessageTypeFileTransferRequest:
		log.Printf("📁 DEBUG: Processing file transfer request")
		// Manejar solicitud de transferencia de archivo
		var request FileTransferRequest
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &request); err == nil {
				c.mutex.RLock()
				handler := c.fileTransferRequestHandler
				c.mutex.RUnlock()

				if handler != nil {
					log.Printf("📁 Received file transfer request: %s (%.2f MB, %d chunks)",
						request.FileName, request.FileSizeMB, request.TotalChunks)
					handler(request)
				} else {
					log.Println("📁 Received file transfer request but no handler set")
				}
			} else {
				log.Printf("❌ Failed to unmarshal file transfer request: %v", err)
			}
		} else {
			log.Printf("❌ Failed to marshal file transfer request data: %v", err)
		}

	case MessageTypeFileChunk:
		log.Printf("📦 DEBUG: Processing file chunk")
		// Manejar chunk de archivo
		var chunk FileChunk
		if data, err := json.Marshal(message.Data); err == nil {
			if err := json.Unmarshal(data, &chunk); err == nil {
				c.mutex.RLock()
				handler := c.fileChunkHandler
				c.mutex.RUnlock()

				if handler != nil {
					log.Printf("📦 Received file chunk %d/%d for transfer %s",
						chunk.ChunkIndex+1, chunk.TotalChunks, chunk.TransferID)
					handler(chunk)
				} else {
					log.Println("📦 Received file chunk but no handler set")
				}
			} else {
				log.Printf("❌ Failed to unmarshal file chunk: %v", err)
			}
		} else {
			log.Printf("❌ Failed to marshal file chunk data: %v", err)
		}

	default:
		log.Printf("🔍 DEBUG: Unknown message type: %s", message.Type)
	}
}

// startKeepAlive envía pings periódicos para mantener la conexión viva
func (c *APIClient) startKeepAlive() {
	ticker := time.NewTicker(20 * time.Second) // Ping cada 20 segundos
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.RLock()
		conn := c.conn
		connected := c.isConnected
		c.mutex.RUnlock()

		if !connected || conn == nil {
			log.Println("Keep-alive stopped: not connected")
			break
		}

		// Usar mutex de escritura para enviar ping
		c.writeMutex.Lock()
		err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(10*time.Second))
		c.writeMutex.Unlock()

		if err != nil {
			log.Printf("Failed to send ping: %v", err)
			// No cerrar conexión por error de ping, solo logear
		} else {
			log.Println("Sent ping to server")
		}
	}
}

// AcceptRemoteControlSession acepta una sesión de control remoto
func (c *APIClient) AcceptRemoteControlSession(sessionID string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	message := WebSocketMessage{
		Type: MessageTypeSessionAccepted,
		Data: SessionAcceptedMessage{
			SessionID: sessionID,
		},
	}

	return c.sendMessage(message)
}

// RejectRemoteControlSession rechaza una sesión de control remoto
func (c *APIClient) RejectRemoteControlSession(sessionID, reason string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	message := WebSocketMessage{
		Type: MessageTypeSessionRejected,
		Data: SessionRejectedMessage{
			SessionID: sessionID,
			Reason:    reason,
		},
	}

	return c.sendMessage(message)
}

// emitSessionEvent emite un evento a través de la función de callback
func (c *APIClient) emitSessionEvent(eventType string, data interface{}) {
	log.Printf("🔍 DEBUG: emitSessionEvent called with eventType: %s", eventType)

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if c.sessionEventHandler != nil {
		log.Printf("🔍 DEBUG: Calling sessionEventHandler for event: %s", eventType)
		c.sessionEventHandler(eventType, data)
		log.Printf("🔍 DEBUG: sessionEventHandler call completed for event: %s", eventType)
	} else {
		log.Printf("❌ DEBUG: No session event handler set for event: %s", eventType)
	}
}

// SendScreenFrame envía un frame de pantalla al servidor
func (c *APIClient) SendScreenFrame(frame ScreenFrame) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	message := WebSocketMessage{
		Type: MessageTypeScreenFrame,
		Data: frame,
	}

	return c.sendMessage(message)
}

// SendScreenFrameAsync envía un frame de pantalla de forma asíncrona
func (c *APIClient) SendScreenFrameAsync(frame ScreenFrame) {
	go func() {
		if err := c.SendScreenFrame(frame); err != nil {
			log.Printf("Error sending screen frame async: %v", err)
		}
	}()
}

// SendVideoChunk envía un chunk de video al servidor
func (c *APIClient) SendVideoChunk(videoID, sessionID string, chunkNumber, totalChunks int, chunkData []byte) error {
	// Codificar chunk data en base64
	chunkDataB64 := base64.StdEncoding.EncodeToString(chunkData)

	videoChunk := map[string]interface{}{
		"video_id":      videoID,
		"session_id":    sessionID,       // Usar session_id real pasado como parámetro
		"chunk_index":   chunkNumber - 1, // Convertir a 0-based index
		"chunk_data":    chunkDataB64,    // Base64 encoded
		"is_last_chunk": chunkNumber == totalChunks,
		"file_size":     int64(len(chunkData) * totalChunks), // Estimación
		"duration":      60,                                  // Duración estimada en segundos
		"file_name":     fmt.Sprintf("session_video_%s.mp4", videoID),
		"timestamp":     time.Now().Unix(),
	}

	message := WebSocketMessage{
		Type: "video_chunk_upload",
		Data: videoChunk,
	}

	return c.sendMessage(message)
}

// SendVideoComplete envía señal de finalización del upload de video
func (c *APIClient) SendVideoComplete(videoID string, duration, frameCount int) error {
	finalizeData := map[string]interface{}{
		"video_id":    videoID,
		"duration":    duration,
		"frame_count": frameCount,
		"timestamp":   time.Now().Unix(),
	}

	message := WebSocketMessage{
		Type: "video_upload_complete",
		Data: finalizeData,
	}

	return c.sendMessage(message)
}

// SendVideoFrame envía un frame de video grabado al servidor
func (c *APIClient) SendVideoFrame(frame interface{}) error {
	videoFrame, ok := frame.(VideoFrameUpload)
	if !ok {
		return fmt.Errorf("SendVideoFrame: tipo de frame inválido")
	}
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	// Codificar frame data en base64
	frameB64 := base64.StdEncoding.EncodeToString(videoFrame.FrameData)

	msg := WebSocketMessage{
		Type: MessageTypeVideoFrameUpload,
		Data: map[string]interface{}{
			"session_id":  videoFrame.SessionID,
			"video_id":    videoFrame.VideoID,
			"frame_index": videoFrame.FrameIndex,
			"timestamp":   videoFrame.Timestamp,
			"frame_data":  frameB64,
		},
	}

	return c.sendMessage(msg)
}

// SendVideoRecordingComplete envía los metadatos de una grabación de video finalizada al servidor.
func (c *APIClient) SendVideoRecordingComplete(videoID string, sessionID string, totalFrames int, fps float64, durationSeconds float64) error {
	if !c.IsConnected() {
		return fmt.Errorf("no conectado al servidor")
	}

	payload := VideoRecordingCompletePayload{
		VideoID:         videoID,
		SessionID:       sessionID,
		TotalFrames:     totalFrames,
		FPS:             fps,
		DurationSeconds: durationSeconds,
		Timestamp:       time.Now().Unix(),
	}

	message := WebSocketMessage{
		Type: MessageTypeVideoRecordingComplete, // Nueva constante de tipo de mensaje
		Data: payload,
	}

	log.Printf("🚀 Enviando metadatos de fin de grabación: VideoID=%s, SessionID=%s, Frames=%d, FPS=%.2f, Duración=%.2fs",
		videoID, sessionID, totalFrames, fps, durationSeconds)

	return c.sendMessage(message)
}

// SendFileTransferAcknowledgement envía confirmación de recepción de archivo al servidor
func (c *APIClient) SendFileTransferAcknowledgement(transferID, sessionID string, success bool, errorMessage, filePath, fileChecksum string) error {
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	ack := FileTransferAcknowledgement{
		TransferID:   transferID,
		SessionID:    sessionID,
		Success:      success,
		ErrorMessage: errorMessage,
		FilePath:     filePath,
		FileChecksum: fileChecksum,
		Timestamp:    time.Now().Unix(),
	}

	message := WebSocketMessage{
		Type: MessageTypeFileTransferAck,
		Data: ack,
	}

	log.Printf("📤 Sending file transfer acknowledgement: Transfer=%s, Success=%v", transferID, success)
	return c.sendMessage(message)
}
