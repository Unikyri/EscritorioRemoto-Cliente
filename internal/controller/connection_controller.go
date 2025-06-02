package controller

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/state"
	"EscritorioRemoto-Cliente/internal/model/valueobjects"
	"fmt"
	"time"
)

// ConnectionController maneja las operaciones de conexión
type ConnectionController struct {
	connectionService ConnectionService
	stateContext      *state.ConnectionStateContext
	eventManager      *observer.EventManager
}

// ConnectionService interface para el servicio de conexión
type ConnectionService interface {
	Connect(serverURL string) error
	Disconnect() error
	IsConnected() bool
	GetServerURL() string
	SendHeartbeat() error
	GetConnectionInfo() *ConnectionInfo
	GetAPIClient() interface{}
}

// ConnectionInfo representa información de conexión
type ConnectionInfo struct {
	IsConnected    bool      `json:"is_connected"`
	ServerURL      string    `json:"server_url"`
	ConnectedAt    *time.Time `json:"connected_at,omitempty"`
	LastHeartbeat  *time.Time `json:"last_heartbeat,omitempty"`
	ConnectionTime string    `json:"connection_time"`
}

// ConnectRequest representa la solicitud de conexión
type ConnectRequest struct {
	ServerURL string `json:"server_url"`
}

// ConnectResponse representa la respuesta de conexión
type ConnectResponse struct {
	Success        bool                       `json:"success"`
	Status         *valueobjects.ConnectionStatus `json:"status,omitempty"`
	ConnectionInfo *ConnectionInfo            `json:"connection_info,omitempty"`
	Error          string                     `json:"error,omitempty"`
}

// DisconnectResponse representa la respuesta de desconexión
type DisconnectResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// StatusResponse representa la respuesta de estado
type StatusResponse struct {
	Status         *valueobjects.ConnectionStatus `json:"status"`
	ConnectionInfo *ConnectionInfo                `json:"connection_info"`
	CanConnect     bool                          `json:"can_connect"`
	CanDisconnect  bool                          `json:"can_disconnect"`
}

// NewConnectionController crea un nuevo controlador de conexión
func NewConnectionController(connectionService ConnectionService) *ConnectionController {
	return &ConnectionController{
		connectionService: connectionService,
		stateContext:      state.NewConnectionStateContext(),
		eventManager:      observer.GetInstance(),
	}
}

// Connect maneja la solicitud de conexión al servidor
func (cc *ConnectionController) Connect(request ConnectRequest) ConnectResponse {
	// Validar request
	if err := cc.validateConnectRequest(request); err != nil {
		return ConnectResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Verificar si puede conectar
	if !cc.stateContext.CanConnect() {
		return ConnectResponse{
			Success: false,
			Error:   "Cannot connect in current state",
		}
	}

	// Cambiar estado a conectando usando State Pattern
	err := cc.stateContext.Connect(request.ServerURL)
	if err != nil {
		return ConnectResponse{
			Success: false,
			Error:   fmt.Sprintf("State transition failed: %v", err),
		}
	}

	// Ejecutar conexión a través del servicio
	err = cc.connectionService.Connect(request.ServerURL)
	if err != nil {
		// Manejar error en el estado
		cc.stateContext.HandleError(err.Error())
		
		return ConnectResponse{
			Success: false,
			Error:   fmt.Sprintf("Connection failed: %v", err),
		}
	}

	// Cambiar a estado conectado
	connectedState := &state.ConnectedState{}
	cc.stateContext.SetState(connectedState)
	cc.stateContext.SetStatus(valueobjects.NewConnectedStatus(request.ServerURL))

	// Publicar evento de conexión exitosa
	cc.eventManager.Publish(observer.Event{
		Type: "connection_established",
		Data: map[string]interface{}{
			"server_url":    request.ServerURL,
			"connected_at":  time.Now().UTC(),
			"state":         "CONNECTED",
		},
	})

	return ConnectResponse{
		Success:        true,
		Status:         cc.stateContext.GetStatus(),
		ConnectionInfo: cc.getConnectionInfo(),
	}
}

// Disconnect maneja la solicitud de desconexión del servidor
func (cc *ConnectionController) Disconnect() DisconnectResponse {
	// Verificar si puede desconectar
	if !cc.stateContext.CanDisconnect() {
		return DisconnectResponse{
			Success: false,
			Error:   "Cannot disconnect in current state",
		}
	}

	// Ejecutar desconexión usando State Pattern
	err := cc.stateContext.Disconnect()
	if err != nil {
		return DisconnectResponse{
			Success: false,
			Error:   fmt.Sprintf("State transition failed: %v", err),
		}
	}

	// Ejecutar desconexión a través del servicio
	err = cc.connectionService.Disconnect()
	if err != nil {
		// Manejar error pero continuar con el proceso
		cc.eventManager.Publish(observer.Event{
			Type: "disconnect_warning",
			Data: map[string]interface{}{
				"warning": err.Error(),
			},
		})
	}

	// Publicar evento de desconexión
	cc.eventManager.Publish(observer.Event{
		Type: "connection_terminated",
		Data: map[string]interface{}{
			"reason":         "user_requested",
			"disconnected_at": time.Now().UTC(),
		},
	})

	return DisconnectResponse{
		Success: true,
		Message: "Disconnected successfully",
	}
}

// GetStatus retorna el estado actual de conexión
func (cc *ConnectionController) GetStatus() StatusResponse {
	return StatusResponse{
		Status:         cc.stateContext.GetStatus(),
		ConnectionInfo: cc.getConnectionInfo(),
		CanConnect:     cc.stateContext.CanConnect(),
		CanDisconnect:  cc.stateContext.CanDisconnect(),
	}
}

// SendHeartbeat envía un heartbeat al servidor
func (cc *ConnectionController) SendHeartbeat() error {
	if !cc.connectionService.IsConnected() {
		return fmt.Errorf("not connected to server")
	}

	err := cc.connectionService.SendHeartbeat()
	if err != nil {
		// Manejar error de heartbeat
		cc.stateContext.HandleError(fmt.Sprintf("Heartbeat failed: %v", err))
		return err
	}

	// Publicar evento de heartbeat exitoso
	cc.eventManager.Publish(observer.Event{
		Type: "heartbeat_sent",
		Data: map[string]interface{}{
			"timestamp": time.Now().UTC(),
		},
	})

	return nil
}

// HandleConnectionError maneja errores de conexión
func (cc *ConnectionController) HandleConnectionError(errorMsg string) {
	cc.stateContext.HandleError(errorMsg)
	
	// Publicar evento de error
	cc.eventManager.Publish(observer.Event{
		Type: "connection_error_handled",
		Data: map[string]interface{}{
			"error_message": errorMsg,
			"timestamp":     time.Now().UTC(),
		},
	})
}

// validateConnectRequest valida la solicitud de conexión
func (cc *ConnectionController) validateConnectRequest(request ConnectRequest) error {
	if request.ServerURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}
	
	// Validación básica de URL
	if len(request.ServerURL) < 7 { // http://
		return fmt.Errorf("invalid server URL format")
	}
	
	return nil
}

// getConnectionInfo obtiene información de conexión
func (cc *ConnectionController) getConnectionInfo() *ConnectionInfo {
	serviceInfo := cc.connectionService.GetConnectionInfo()
	if serviceInfo == nil {
		return &ConnectionInfo{
			IsConnected: false,
		}
	}
	
	return serviceInfo
} 