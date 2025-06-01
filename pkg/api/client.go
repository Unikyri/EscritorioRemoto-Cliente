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
		readTimeout:    90 * time.Second,
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

	// Configurar ping handler para mantener conexión viva
	c.conn.SetPingHandler(func(appData string) error {
		log.Println("Received ping, sending pong")
		return c.conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Second))
	})

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
			// Solo cerrar en errores graves, no en timeouts de lectura
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				log.Printf("WebSocket error: %v", err)
				break
			} else if netErr, ok := err.(*websocket.CloseError); ok {
				log.Printf("WebSocket closed: %v", netErr)
				break
			} else {
				// Para timeouts y otros errores temporales, continuar
				log.Printf("WebSocket read timeout or temporary error: %v", err)
				continue
			}
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

		// Enviar ping
		err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(10*time.Second))
		if err != nil {
			log.Printf("Failed to send ping: %v", err)
			// No cerrar conexión por error de ping, solo logear
		} else {
			log.Println("Sent ping to server")
		}
	}
}
