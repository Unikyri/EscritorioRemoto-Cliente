package state

import (
	"EscritorioRemoto-Cliente/internal/model/valueobjects"
)

// ConnectionStateContext maneja el contexto del estado de conexión
type ConnectionStateContext struct {
	currentState ConnectionState
	status       *valueobjects.ConnectionStatus
}

// ConnectionState interface para el patrón State
type ConnectionState interface {
	Connect(ctx *ConnectionStateContext, serverURL string) error
	Disconnect(ctx *ConnectionStateContext) error
	HandleError(ctx *ConnectionStateContext, errorMsg string) error
	GetStatus() string
	CanConnect() bool
	CanDisconnect() bool
}

// NewConnectionStateContext crea un nuevo contexto de estado
func NewConnectionStateContext() *ConnectionStateContext {
	// Crear status inicial de desconectado
	disconnectedStatus, _ := valueobjects.NewConnectionStatus(valueobjects.StatusDisconnected)
	
	ctx := &ConnectionStateContext{
		status: disconnectedStatus,
	}
	
	// Establecer estado inicial
	ctx.currentState = &DisconnectedState{}
	
	return ctx
}

// Connect delega al estado actual
func (ctx *ConnectionStateContext) Connect(serverURL string) error {
	return ctx.currentState.Connect(ctx, serverURL)
}

// Disconnect delega al estado actual
func (ctx *ConnectionStateContext) Disconnect() error {
	return ctx.currentState.Disconnect(ctx)
}

// HandleError delega al estado actual
func (ctx *ConnectionStateContext) HandleError(errorMsg string) error {
	return ctx.currentState.HandleError(ctx, errorMsg)
}

// GetStatus retorna el status actual
func (ctx *ConnectionStateContext) GetStatus() *valueobjects.ConnectionStatus {
	return ctx.status
}

// CanConnect verifica si puede conectar en el estado actual
func (ctx *ConnectionStateContext) CanConnect() bool {
	return ctx.currentState.CanConnect()
}

// CanDisconnect verifica si puede desconectar en el estado actual
func (ctx *ConnectionStateContext) CanDisconnect() bool {
	return ctx.currentState.CanDisconnect()
}

// SetState cambia el estado actual (usado internamente por los estados)
func (ctx *ConnectionStateContext) SetState(state ConnectionState) {
	ctx.currentState = state
}

// SetStatus cambia el status actual (usado internamente por los estados)
func (ctx *ConnectionStateContext) SetStatus(status *valueobjects.ConnectionStatus) {
	ctx.status = status
}

// GetCurrentStateName retorna el nombre del estado actual
func (ctx *ConnectionStateContext) GetCurrentStateName() string {
	return ctx.currentState.GetStatus()
} 