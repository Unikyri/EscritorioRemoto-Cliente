package controller

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"context"
	"fmt"
	"time"
)

// AppController es el controlador principal que orquesta todos los demás
type AppController struct {
	authController       *AuthController
	connectionController *ConnectionController
	pcController         *PCController
	eventManager         *observer.EventManager
	
	// Context para manejo de Wails
	ctx context.Context
	
	// Estados de la aplicación
	isInitialized bool
	startTime     time.Time
}

// NewAppController crea un nuevo controlador principal
func NewAppController(
	authService AuthService,
	connectionService ConnectionService,
	pcService PCService,
) *AppController {
	return &AppController{
		authController:       NewAuthController(authService),
		connectionController: NewConnectionController(connectionService),
		pcController:         NewPCController(pcService),
		eventManager:         observer.GetInstance(),
		startTime:            time.Now().UTC(),
	}
}

// Initialize inicializa la aplicación
func (ac *AppController) Initialize(ctx context.Context) error {
	ac.ctx = ctx
	
	// Registrar observadores de eventos
	if err := ac.setupEventObservers(); err != nil {
		return fmt.Errorf("failed to setup event observers: %w", err)
	}
	
	// Publicar evento de inicialización
	ac.eventManager.Publish(observer.Event{
		Type: "app_initialized",
		Data: map[string]interface{}{
			"start_time": ac.startTime,
			"version":    "1.0.0", // En una implementación real vendría de config
		},
	})
	
	ac.isInitialized = true
	return nil
}

// Shutdown cierra la aplicación de forma limpia
func (ac *AppController) Shutdown() error {
	// Desconectar si está conectado
	if ac.connectionController.connectionService.IsConnected() {
		ac.connectionController.Disconnect()
	}
	
	// Logout si está autenticado
	if ac.authController.IsAuthenticated() {
		ac.authController.Logout()
	}
	
	// Publicar evento de shutdown
	ac.eventManager.Publish(observer.Event{
		Type: "app_shutdown",
		Data: map[string]interface{}{
			"shutdown_time": time.Now().UTC(),
			"uptime":        time.Since(ac.startTime),
		},
	})
	
	// Limpiar observadores
	ac.eventManager.Clear()
	
	return nil
}

// ===== MÉTODOS DELEGADOS A CONTROLADORES ESPECÍFICOS =====

// Login delega al AuthController
func (ac *AppController) Login(username, password string) LoginResponse {
	if !ac.isInitialized {
		return LoginResponse{
			Success: false,
			Error:   "Application not initialized",
		}
	}
	
	return ac.authController.Login(LoginRequest{
		Username: username,
		Password: password,
	})
}

// Logout delega al AuthController
func (ac *AppController) Logout() LogoutResponse {
	return ac.authController.Logout()
}

// IsAuthenticated delega al AuthController
func (ac *AppController) IsAuthenticated() bool {
	return ac.authController.IsAuthenticated()
}

// Connect delega al ConnectionController
func (ac *AppController) Connect(serverURL string) ConnectResponse {
	if !ac.isInitialized {
		return ConnectResponse{
			Success: false,
			Error:   "Application not initialized",
		}
	}
	
	return ac.connectionController.Connect(ConnectRequest{
		ServerURL: serverURL,
	})
}

// Disconnect delega al ConnectionController
func (ac *AppController) Disconnect() DisconnectResponse {
	return ac.connectionController.Disconnect()
}

// GetConnectionStatus delega al ConnectionController
func (ac *AppController) GetConnectionStatus() StatusResponse {
	return ac.connectionController.GetStatus()
}

// RegisterPC delega al PCController
func (ac *AppController) RegisterPC() PCRegistrationResponse {
	if !ac.isInitialized {
		return PCRegistrationResponse{
			Success: false,
			Error:   "Application not initialized",
		}
	}
	
	return ac.pcController.RegisterPC()
}

// GetPCInfo delega al PCController
func (ac *AppController) GetPCInfo() PCInfoResponse {
	return ac.pcController.GetPCInfo()
}

// ===== MÉTODOS DE ESTADO DE LA APLICACIÓN =====

// GetAppStatus retorna el estado general de la aplicación
func (ac *AppController) GetAppStatus() AppStatusResponse {
	return AppStatusResponse{
		IsInitialized:   ac.isInitialized,
		IsAuthenticated: ac.authController.IsAuthenticated(),
		IsConnected:     ac.connectionController.connectionService.IsConnected(),
		Uptime:          time.Since(ac.startTime),
		StartTime:       ac.startTime,
	}
}

// GetSystemInfo retorna información del sistema
func (ac *AppController) GetSystemInfo() map[string]string {
	return ac.pcController.GetSystemInfo()
}

// GetConnectionService retorna el servicio de conexión
func (ac *AppController) GetConnectionService() ConnectionService {
	return ac.connectionController.connectionService
}

// ===== CONFIGURACIÓN DE OBSERVADORES =====

// setupEventObservers configura los observadores de eventos
func (ac *AppController) setupEventObservers() error {
	// Crear observer para conexión
	connectionObserver := &ConnectionObserver{
		controller: ac.connectionController,
	}
	
	// Registrar observers
	eventTypes := []string{
		"connection_state_changed",
		"connection_error",
		"connection_lost",
		"login_successful",
		"logout_completed",
	}
	
	for _, eventType := range eventTypes {
		if err := ac.eventManager.Subscribe(eventType, connectionObserver); err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", eventType, err)
		}
	}
	
	return nil
}

// ===== TIPOS DE RESPUESTA =====

// AppStatusResponse representa el estado de la aplicación
type AppStatusResponse struct {
	IsInitialized   bool          `json:"is_initialized"`
	IsAuthenticated bool          `json:"is_authenticated"`
	IsConnected     bool          `json:"is_connected"`
	Uptime          time.Duration `json:"uptime"`
	StartTime       time.Time     `json:"start_time"`
}

// ===== OBSERVADOR DE CONEXIÓN =====

// ConnectionObserver observa eventos de conexión
type ConnectionObserver struct {
	controller *ConnectionController
}

// OnEvent implementa Observer.OnEvent
func (co *ConnectionObserver) OnEvent(event observer.Event) error {
	switch event.Type {
	case "connection_error":
		if data, ok := event.Data.(map[string]interface{}); ok {
			if errorMsg, exists := data["error_message"].(string); exists {
				co.controller.HandleConnectionError(errorMsg)
			}
		}
	case "connection_lost":
		// Manejar pérdida de conexión
		fmt.Printf("Connection lost: %v\n", event.Data)
	}
	return nil
}

// GetID implementa Observer.GetID
func (co *ConnectionObserver) GetID() string {
	return "connection_observer"
} 