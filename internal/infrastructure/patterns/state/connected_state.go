package state

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/model/valueobjects"
	"fmt"
)

// ConnectedState implementa el estado conectado
type ConnectedState struct{}

// Connect no puede conectar si ya está conectado
func (s *ConnectedState) Connect(ctx *ConnectionStateContext, serverURL string) error {
	return fmt.Errorf("already connected to server")
}

// Disconnect desconecta del servidor
func (s *ConnectedState) Disconnect(ctx *ConnectionStateContext) error {
	// Cambiar a estado desconectado
	disconnectedState := &DisconnectedState{}
	ctx.SetState(disconnectedState)
	
	// Crear nuevo status de desconectado
	disconnectedStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusDisconnected)
	if err != nil {
		return fmt.Errorf("failed to create disconnected status: %w", err)
	}
	ctx.SetStatus(disconnectedStatus)

	// Publicar evento de desconexión
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "connection_terminated",
		Data: map[string]interface{}{
			"from_state": "CONNECTED",
			"to_state":   "DISCONNECTED",
			"reason":     "user_requested",
		},
	})

	return nil
}

// HandleError maneja errores cuando está conectado
func (s *ConnectedState) HandleError(ctx *ConnectionStateContext, errorMsg string) error {
	// Cambiar a estado de error
	errorState := &ErrorState{}
	ctx.SetState(errorState)
	
	// Crear nuevo status de error
	errorStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusError)
	if err != nil {
		return fmt.Errorf("failed to create error status: %w", err)
	}
	ctx.SetStatus(errorStatus)

	// Publicar evento de error
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "connection_error_occurred",
		Data: map[string]interface{}{
			"error_message": errorMsg,
			"from_state":    "CONNECTED",
			"to_state":      "ERROR",
		},
	})

	return nil
}

// GetStatus retorna el estado actual
func (s *ConnectedState) GetStatus() string {
	return valueobjects.StatusConnected
}

// CanConnect verifica si puede conectar
func (s *ConnectedState) CanConnect() bool {
	return false // No puede conectar si ya está conectado
}

// CanDisconnect verifica si puede desconectar
func (s *ConnectedState) CanDisconnect() bool {
	return true // Puede desconectar cuando está conectado
} 