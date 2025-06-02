package valueobjects

import (
	"fmt"
	"time"
)

// ConnectionStatus representa el estado de conexión como Value Object
type ConnectionStatus struct {
	status         string
	timestamp      time.Time
	errorMessage   string
	serverURL      string
	connectionTime *time.Time
}

// Constantes para los estados válidos
const (
	StatusDisconnected = "DISCONNECTED"
	StatusConnecting   = "CONNECTING"
	StatusConnected    = "CONNECTED"
	StatusError        = "ERROR"
	StatusReconnecting = "RECONNECTING"
)

// NewConnectionStatus crea un nuevo ConnectionStatus
func NewConnectionStatus(status string) (*ConnectionStatus, error) {
	if !isValidStatus(status) {
		return nil, fmt.Errorf("invalid connection status: %s", status)
	}

	return &ConnectionStatus{
		status:    status,
		timestamp: time.Now().UTC(),
	}, nil
}

// NewDisconnectedStatus crea un estado desconectado
func NewDisconnectedStatus() *ConnectionStatus {
	return &ConnectionStatus{
		status:    StatusDisconnected,
		timestamp: time.Now().UTC(),
	}
}

// NewConnectingStatus crea un estado conectando
func NewConnectingStatus(serverURL string) *ConnectionStatus {
	return &ConnectionStatus{
		status:    StatusConnecting,
		timestamp: time.Now().UTC(),
		serverURL: serverURL,
	}
}

// NewConnectedStatus crea un estado conectado
func NewConnectedStatus(serverURL string) *ConnectionStatus {
	now := time.Now().UTC()
	return &ConnectionStatus{
		status:         StatusConnected,
		timestamp:      now,
		serverURL:      serverURL,
		connectionTime: &now,
	}
}

// NewErrorStatus crea un estado de error
func NewErrorStatus(errorMessage string) *ConnectionStatus {
	return &ConnectionStatus{
		status:       StatusError,
		timestamp:    time.Now().UTC(),
		errorMessage: errorMessage,
	}
}

// NewReconnectingStatus crea un estado reconectando
func NewReconnectingStatus(serverURL string) *ConnectionStatus {
	return &ConnectionStatus{
		status:    StatusReconnecting,
		timestamp: time.Now().UTC(),
		serverURL: serverURL,
	}
}

// Getters
func (cs *ConnectionStatus) Status() string {
	return cs.status
}

func (cs *ConnectionStatus) Timestamp() time.Time {
	return cs.timestamp
}

func (cs *ConnectionStatus) ErrorMessage() string {
	return cs.errorMessage
}

func (cs *ConnectionStatus) ServerURL() string {
	return cs.serverURL
}

func (cs *ConnectionStatus) ConnectionTime() *time.Time {
	return cs.connectionTime
}

// Métodos de consulta
func (cs *ConnectionStatus) IsDisconnected() bool {
	return cs.status == StatusDisconnected
}

func (cs *ConnectionStatus) IsConnecting() bool {
	return cs.status == StatusConnecting
}

func (cs *ConnectionStatus) IsConnected() bool {
	return cs.status == StatusConnected
}

func (cs *ConnectionStatus) IsError() bool {
	return cs.status == StatusError
}

func (cs *ConnectionStatus) IsReconnecting() bool {
	return cs.status == StatusReconnecting
}

// CanTransitionTo verifica si es válida la transición a otro estado
func (cs *ConnectionStatus) CanTransitionTo(newStatus string) error {
	if !isValidStatus(newStatus) {
		return fmt.Errorf("invalid target status: %s", newStatus)
	}

	// Definir transiciones válidas
	validTransitions := map[string][]string{
		StatusDisconnected: {StatusConnecting, StatusError},
		StatusConnecting:   {StatusConnected, StatusError, StatusDisconnected},
		StatusConnected:    {StatusDisconnected, StatusError, StatusReconnecting},
		StatusError:        {StatusDisconnected, StatusConnecting, StatusReconnecting},
		StatusReconnecting: {StatusConnected, StatusError, StatusDisconnected},
	}

	allowedStates, exists := validTransitions[cs.status]
	if !exists {
		return fmt.Errorf("unknown current status: %s", cs.status)
	}

	for _, allowed := range allowedStates {
		if newStatus == allowed {
			return nil
		}
	}

	return fmt.Errorf("invalid transition from %s to %s", cs.status, newStatus)
}

// Equals compara dos ConnectionStatus
func (cs *ConnectionStatus) Equals(other *ConnectionStatus) bool {
	if other == nil {
		return false
	}
	return cs.status == other.status
}

// String implementa fmt.Stringer
func (cs *ConnectionStatus) String() string {
	return cs.status
}

// GetConnectionDuration retorna la duración de la conexión si está conectado
func (cs *ConnectionStatus) GetConnectionDuration() time.Duration {
	if cs.connectionTime == nil || !cs.IsConnected() {
		return 0
	}
	return time.Since(*cs.connectionTime)
}

// isValidStatus verifica si un string es un estado válido
func isValidStatus(status string) bool {
	validStatuses := []string{
		StatusDisconnected,
		StatusConnecting,
		StatusConnected,
		StatusError,
		StatusReconnecting,
	}

	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}
