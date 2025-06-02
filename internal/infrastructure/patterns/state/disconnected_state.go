package state

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/model/valueobjects"
	"fmt"
)

// DisconnectedState implementa el estado desconectado
type DisconnectedState struct{}

// Connect intenta conectar desde estado desconectado
func (s *DisconnectedState) Connect(ctx *ConnectionStateContext, serverURL string) error {
	if serverURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Cambiar a estado conectando
	connectingState := &ConnectingState{}
	ctx.SetState(connectingState)
	
	// Crear nuevo status de conectando
	connectingStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusConnecting)
	if err != nil {
		return fmt.Errorf("failed to create connecting status: %w", err)
	}
	ctx.SetStatus(connectingStatus)

	// Publicar evento de cambio de estado
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "connection_state_changed",
		Data: map[string]interface{}{
			"from_state": "DISCONNECTED",
			"to_state":   "CONNECTING",
			"server_url": serverURL,
		},
	})

	return nil
}

// Disconnect no puede desconectar si ya está desconectado
func (s *DisconnectedState) Disconnect(ctx *ConnectionStateContext) error {
	return fmt.Errorf("already disconnected")
}

// HandleError maneja errores en estado desconectado
func (s *DisconnectedState) HandleError(ctx *ConnectionStateContext, errorMsg string) error {
	// En estado desconectado, los errores se registran pero no cambian el estado
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "connection_error_in_disconnected_state",
		Data: map[string]interface{}{
			"error_message": errorMsg,
			"current_state": "DISCONNECTED",
		},
	})

	return nil
}

// GetStatus retorna el estado actual
func (s *DisconnectedState) GetStatus() string {
	return valueobjects.StatusDisconnected
}

// CanConnect verifica si puede conectar
func (s *DisconnectedState) CanConnect() bool {
	return true
}

// CanDisconnect verifica si puede desconectar
func (s *DisconnectedState) CanDisconnect() bool {
	return false
}

// ConnectingState implementa el estado conectando
type ConnectingState struct{}

// Connect no puede conectar si ya está conectando
func (s *ConnectingState) Connect(ctx *ConnectionStateContext, serverURL string) error {
	return fmt.Errorf("already connecting")
}

// Disconnect puede cancelar la conexión
func (s *ConnectingState) Disconnect(ctx *ConnectionStateContext) error {
	// Cambiar a estado desconectado
	disconnectedState := &DisconnectedState{}
	ctx.SetState(disconnectedState)
	
	// Crear nuevo status de desconectado
	disconnectedStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusDisconnected)
	if err != nil {
		return fmt.Errorf("failed to create disconnected status: %w", err)
	}
	ctx.SetStatus(disconnectedStatus)

	// Publicar evento de cancelación
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "connection_cancelled",
		Data: map[string]interface{}{
			"from_state": "CONNECTING",
			"to_state":   "DISCONNECTED",
		},
	})

	return nil
}

// HandleError maneja errores durante la conexión
func (s *ConnectingState) HandleError(ctx *ConnectionStateContext, errorMsg string) error {
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
			"from_state":    "CONNECTING",
			"to_state":      "ERROR",
		},
	})

	return nil
}

// GetStatus retorna el estado actual
func (s *ConnectingState) GetStatus() string {
	return valueobjects.StatusConnecting
}

// CanConnect verifica si puede conectar
func (s *ConnectingState) CanConnect() bool {
	return false
}

// CanDisconnect verifica si puede desconectar
func (s *ConnectingState) CanDisconnect() bool {
	return true // Puede cancelar la conexión
}

// ErrorState implementa el estado de error
type ErrorState struct{}

// Connect puede intentar reconectar desde error
func (s *ErrorState) Connect(ctx *ConnectionStateContext, serverURL string) error {
	// Cambiar a estado conectando
	connectingState := &ConnectingState{}
	ctx.SetState(connectingState)
	
	// Crear nuevo status de conectando
	connectingStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusConnecting)
	if err != nil {
		return fmt.Errorf("failed to create connecting status: %w", err)
	}
	ctx.SetStatus(connectingStatus)

	return nil
}

// Disconnect desde error va a desconectado
func (s *ErrorState) Disconnect(ctx *ConnectionStateContext) error {
	// Cambiar a estado desconectado
	disconnectedState := &DisconnectedState{}
	ctx.SetState(disconnectedState)
	
	// Crear nuevo status de desconectado
	disconnectedStatus, err := valueobjects.NewConnectionStatus(valueobjects.StatusDisconnected)
	if err != nil {
		return fmt.Errorf("failed to create disconnected status: %w", err)
	}
	ctx.SetStatus(disconnectedStatus)

	return nil
}

// HandleError maneja errores adicionales en estado de error
func (s *ErrorState) HandleError(ctx *ConnectionStateContext, errorMsg string) error {
	// En estado de error, solo registrar el error adicional
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "additional_error_in_error_state",
		Data: map[string]interface{}{
			"error_message": errorMsg,
		},
	})

	return nil
}

// GetStatus retorna el estado actual
func (s *ErrorState) GetStatus() string {
	return valueobjects.StatusError
}

// CanConnect verifica si puede conectar
func (s *ErrorState) CanConnect() bool {
	return true // Puede intentar reconectar
}

// CanDisconnect verifica si puede desconectar
func (s *ErrorState) CanDisconnect() bool {
	return true
} 