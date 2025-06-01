package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// APIClient maneja la comunicación WebSocket con el servidor
type APIClient struct {
	serverURL   string
	conn        *websocket.Conn
	isConnected bool
	mutex       sync.RWMutex

	// Channels para manejar respuestas
	authResponse chan ClientAuthResponse
	regResponse  chan PCRegistrationResponse

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
		readTimeout:    30 * time.Second,
		writeTimeout:   10 * time.Second,
	}
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

	// Enviar mensaje de cierre
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
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

	// Enviar solicitud de registro
	regReq := WebSocketMessage{
		Type: MessageTypePCRegistration,
		Data: PCRegistrationRequest{
			PCIdentifier: pcIdentifier,
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
	if !c.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	hbReq := WebSocketMessage{
		Type: MessageTypeHeartbeat,
		Data: HeartbeatRequest{
			Timestamp: time.Now().Unix(),
		},
	}

	return c.sendMessage(hbReq)
}

// sendMessage envía un mensaje WebSocket
func (c *APIClient) sendMessage(message WebSocketMessage) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if !c.isConnected || c.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Establecer timeout de escritura
	c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))

	return c.conn.WriteJSON(message)
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

	for {
		c.mutex.RLock()
		conn := c.conn
		connected := c.isConnected
		c.mutex.RUnlock()

		if !connected || conn == nil {
			break
		}

		// Establecer timeout de lectura
		conn.SetReadDeadline(time.Now().Add(c.readTimeout))

		var message WebSocketMessage
		err := conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Procesar mensaje
		c.handleMessage(message)
	}
}

// handleMessage procesa los mensajes recibidos
func (c *APIClient) handleMessage(message WebSocketMessage) {
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

	default:
		log.Printf("Unknown message type: %s", message.Type)
	}
}
