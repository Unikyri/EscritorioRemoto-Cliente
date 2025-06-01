package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"EscritorioRemoto-Cliente/pkg/api"
	"EscritorioRemoto-Cliente/pkg/session"
	"EscritorioRemoto-Cliente/pkg/utils"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx                  context.Context
	apiClient            *api.APIClient
	sessionManager       *session.SessionManager
	heartbeatTicker      *time.Ticker
	lastConnectionStatus bool
}

// ConnectionStatus representa el estado de conexión
type ConnectionStatus struct {
	IsConnected    bool   `json:"isConnected"`
	Status         string `json:"status"`
	LastHeartbeat  int64  `json:"lastHeartbeat"`
	ServerURL      string `json:"serverUrl"`
	ConnectionTime int64  `json:"connectionTime"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		lastConnectionStatus: false,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Inicializar componentes
	a.sessionManager = session.NewSessionManager()
	a.apiClient = api.NewAPIClient("http://localhost:8080") // URL del servidor

	log.Println("App initialized successfully")

	// Si hay una sesión válida, intentar reconectar
	if a.sessionManager.IsAuthenticated() {
		go a.tryReconnect()
	}

	// Iniciar monitoreo de conexión
	go a.startConnectionMonitoring()
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	// Detener heartbeat
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}

	// Desconectar del servidor
	if a.apiClient != nil {
		a.apiClient.Disconnect()
	}

	log.Println("App shutdown completed")
}

// GetConnectionStatus obtiene el estado actual de la conexión
func (a *App) GetConnectionStatus() ConnectionStatus {
	isConnected := a.apiClient.IsConnected()

	status := "disconnected"
	if isConnected {
		status = "connected"
	}

	return ConnectionStatus{
		IsConnected:    isConnected,
		Status:         status,
		LastHeartbeat:  time.Now().Unix(),
		ServerURL:      a.apiClient.GetServerURL(),
		ConnectionTime: time.Now().Unix(),
		ErrorMessage:   "",
	}
}

// startConnectionMonitoring monitorea el estado de conexión y emite eventos
func (a *App) startConnectionMonitoring() {
	ticker := time.NewTicker(5 * time.Second) // Verificar cada 5 segundos (menos agresivo)
	defer ticker.Stop()

	for range ticker.C {
		currentStatus := a.apiClient.IsConnected()

		// Si el estado cambió, emitir evento
		if currentStatus != a.lastConnectionStatus {
			a.lastConnectionStatus = currentStatus

			connectionStatus := a.GetConnectionStatus()

			// Emitir evento de Wails
			runtime.EventsEmit(a.ctx, "connection_status_update", connectionStatus)

			log.Printf("Connection status changed to: %s", connectionStatus.Status)
		}
	}
}

// HandleClientLogin maneja el login del cliente
func (a *App) HandleClientLogin(username, password string) api.AuthResultDTO {
	log.Printf("Attempting login for user: %s", username)

	// Validar parámetros
	if username == "" || password == "" {
		return api.AuthResultDTO{
			Success: false,
			Error:   "Username and password are required",
		}
	}

	// Intentar autenticación
	response, err := a.apiClient.ConnectAndAuthenticate(username, password)
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return api.AuthResultDTO{
			Success: false,
			Error:   fmt.Sprintf("Authentication failed: %v", err),
		}
	}

	// Verificar respuesta del servidor
	if !response.Success {
		log.Printf("Server rejected authentication: %s", response.Error)
		return api.AuthResultDTO{
			Success: false,
			Error:   response.Error,
		}
	}

	// Guardar token en sesión
	err = a.sessionManager.StoreToken(response.Token, response.UserID, username)
	if err != nil {
		log.Printf("Failed to store session: %v", err)
		return api.AuthResultDTO{
			Success: false,
			Error:   "Failed to save session",
		}
	}

	log.Printf("Authentication successful for user: %s", username)

	// Iniciar heartbeat
	a.startHeartbeat()

	return api.AuthResultDTO{
		Success: true,
		Token:   response.Token,
		UserID:  response.UserID,
	}
}

// HandlePCRegistration maneja el registro del PC
func (a *App) HandlePCRegistration() api.PCRegistrationResultDTO {
	log.Println("Attempting PC registration")

	// Verificar que esté autenticado
	if !a.sessionManager.IsAuthenticated() {
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   "Not authenticated",
		}
	}

	// Verificar conexión
	if !a.apiClient.IsConnected() {
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   "Not connected to server",
		}
	}

	// Obtener identificador del PC
	pcIdentifier, err := utils.GetPCIdentifier()
	if err != nil {
		log.Printf("Failed to get PC identifier: %v", err)
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   fmt.Sprintf("Failed to get PC identifier: %v", err),
		}
	}

	log.Printf("Using PC identifier: %s", pcIdentifier)

	// Registrar PC
	response, err := a.apiClient.RegisterPC(pcIdentifier)
	if err != nil {
		log.Printf("PC registration failed: %v", err)
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   fmt.Sprintf("Registration failed: %v", err),
		}
	}

	// Verificar respuesta del servidor
	if !response.Success {
		log.Printf("Server rejected PC registration: %s", response.Error)
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   response.Error,
		}
	}

	// Guardar PC ID en sesión
	err = a.sessionManager.StorePCID(response.PCID)
	if err != nil {
		log.Printf("Failed to store PC ID: %v", err)
		return api.PCRegistrationResultDTO{
			Success: false,
			Error:   "Failed to save PC ID",
		}
	}

	log.Printf("PC registration successful. PC ID: %s", response.PCID)

	return api.PCRegistrationResultDTO{
		Success: true,
		PCID:    response.PCID,
	}
}

// GetSessionInfo obtiene información de la sesión actual
func (a *App) GetSessionInfo() session.SessionData {
	return a.sessionManager.GetSessionData()
}

// GetSystemInfo obtiene información del sistema
func (a *App) GetSystemInfo() map[string]string {
	return utils.GetSystemInfo()
}

// IsConnected verifica si está conectado al servidor
func (a *App) IsConnected() bool {
	return a.apiClient.IsConnected()
}

// Logout cierra la sesión
func (a *App) Logout() error {
	log.Println("Logging out")

	// Detener heartbeat
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
		a.heartbeatTicker = nil
	}

	// Desconectar del servidor
	if a.apiClient != nil {
		a.apiClient.Disconnect()
	}

	// Limpiar sesión
	err := a.sessionManager.ClearSession()
	if err != nil {
		log.Printf("Failed to clear session: %v", err)
		return err
	}

	log.Println("Logout completed")
	return nil
}

// startHeartbeat inicia el envío periódico de heartbeats
func (a *App) startHeartbeat() {
	// Detener heartbeat anterior si existe
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
	}

	// Crear nuevo ticker para heartbeat cada 30 segundos
	a.heartbeatTicker = time.NewTicker(30 * time.Second)

	go func() {
		for range a.heartbeatTicker.C {
			if a.apiClient.IsConnected() {
				err := a.apiClient.SendHeartbeat()
				if err != nil {
					log.Printf("Failed to send heartbeat: %v", err)

					// Solo emitir evento si es un error grave que cambió el estado
					if !a.apiClient.IsConnected() && a.lastConnectionStatus {
						connectionStatus := a.GetConnectionStatus()
						connectionStatus.ErrorMessage = fmt.Sprintf("Heartbeat failed: %v", err)
						runtime.EventsEmit(a.ctx, "connection_status_update", connectionStatus)
						a.lastConnectionStatus = false
					}
				}
			} else {
				log.Println("Skipping heartbeat: not connected")
			}
		}
	}()

	log.Println("Heartbeat started")
}

// tryReconnect intenta reconectar usando la sesión guardada
func (a *App) tryReconnect() {
	log.Println("Attempting to reconnect with saved session")

	sessionData := a.sessionManager.GetSessionData()
	if sessionData.Token == "" {
		log.Println("No valid session found")
		return
	}

	// Intentar conectar
	err := a.apiClient.Connect()
	if err != nil {
		log.Printf("Failed to reconnect: %v", err)
		return
	}

	// Si la conexión es exitosa, iniciar heartbeat
	a.startHeartbeat()
	log.Println("Reconnected successfully")
}
