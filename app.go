package main

import (
	"context"
	"time"

	"EscritorioRemoto-Cliente/internal/controller"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/factory"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/singleton"
	"EscritorioRemoto-Cliente/pkg/api"
	"EscritorioRemoto-Cliente/pkg/remotecontrol"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct refactorizada usando MVC
type App struct {
	ctx           context.Context
	appController *controller.AppController
	eventManager  *observer.EventManager

	// Singletons
	configManager *singleton.ConfigManager

	// API Client para control remoto
	apiClient *api.APIClient

	// RemoteControlAgent para captura de pantalla y control de input
	remoteControlAgent *remotecontrol.RemoteControlAgent

	// Timer para heartbeat automático
	heartbeatTicker *time.Ticker
}

// NewApp crea una nueva instancia de App usando MVC
func NewApp() *App {
	// Inicializar singletons
	configManager := singleton.GetConfigManager()
	eventManager := observer.GetInstance()

	// Crear servicios usando Factory Pattern
	serviceFactory := factory.NewServiceFactory(configManager)
	authService := serviceFactory.CreateAuthService()
	connectionService := serviceFactory.CreateConnectionService()
	pcService := serviceFactory.CreatePCService(connectionService)

	// Crear controlador principal
	appController := controller.NewAppController(
		authService,
		connectionService,
		pcService,
	)

	return &App{
		appController:      appController,
		eventManager:       eventManager,
		configManager:      configManager,
		remoteControlAgent: remotecontrol.NewRemoteControlAgent(),
	}
}

// startup es llamado cuando la app inicia (Wails)
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Inicializar controlador principal
	if err := a.appController.Initialize(ctx); err != nil {
		runtime.LogErrorf(ctx, "Failed to initialize app controller: %v", err)
		return
	}

	// Configurar observador de UI para eventos Wails
	uiObserver := &WailsUIObserver{ctx: ctx}
	a.setupUIEventBindings(uiObserver)

	// Nota: setupRemoteControlHandler se llamará después de conectar/login
	// No se llama aquí porque el APIClient aún no existe

	runtime.LogInfof(ctx, "App initialized successfully with MVC architecture")
}

// setupRemoteControlHandler configura el handler para solicitudes de control remoto
func (a *App) setupRemoteControlHandler() {
	// Obtener el APIClient del controlador de conexión
	if connectionService := a.appController.GetConnectionService(); connectionService != nil {
		// Type assertion para acceder al APIClient
		if realService, ok := connectionService.(interface{ GetAPIClient() interface{} }); ok {
			if apiClientInterface := realService.GetAPIClient(); apiClientInterface != nil {
				if apiClient, ok := apiClientInterface.(*api.APIClient); ok && apiClient != nil {
					a.apiClient = apiClient

					// Configurar handler para solicitudes de control remoto
					apiClient.SetRemoteControlHandler(func(request api.RemoteControlRequest) {
						// Emitir evento a la UI
						runtime.EventsEmit(a.ctx, "incoming_control_request", map[string]interface{}{
							"sessionId":     request.SessionID,
							"adminUserId":   request.AdminUserID,
							"adminUsername": request.AdminUsername,
							"clientPcId":    request.ClientPCID,
						})

						runtime.LogInfof(a.ctx, "Remote control request received from admin: %s", request.AdminUsername)
					})

					// Configurar handler para eventos de sesión
					apiClient.SetSessionEventHandler(func(eventType string, data interface{}) {
						runtime.LogInfof(a.ctx, "Session event received: %s", eventType)

						switch eventType {
						case "session_started":
							runtime.EventsEmit(a.ctx, "control_session_started", data)

							// Extraer sessionID del data
							if sessionData, ok := data.(map[string]interface{}); ok {
								if sessionID, ok := sessionData["session_id"].(string); ok {
									// Iniciar RemoteControlAgent
									err := a.remoteControlAgent.StartSession(sessionID)
									if err != nil {
										runtime.LogErrorf(a.ctx, "Failed to start remote control session: %v", err)
									} else {
										runtime.LogInfof(a.ctx, "Remote control session started: %s", sessionID)
										// Iniciar goroutine para enviar frames
										go a.startScreenStreaming()
									}
								}
							}

						case "session_ended":
							runtime.EventsEmit(a.ctx, "control_session_ended", data)

							// Detener RemoteControlAgent
							err := a.remoteControlAgent.StopSession()
							if err != nil {
								runtime.LogErrorf(a.ctx, "Failed to stop remote control session: %v", err)
							} else {
								runtime.LogInfof(a.ctx, "Remote control session stopped")
							}

						case "session_failed":
							runtime.EventsEmit(a.ctx, "control_session_failed", data)

							// Detener RemoteControlAgent si está activo
							if a.remoteControlAgent.IsActive() {
								a.remoteControlAgent.StopSession()
							}
						}
					})

					// Configurar handler para comandos de input entrantes
					apiClient.SetInputCommandHandler(func(command api.InputCommand) {
						runtime.LogInfof(a.ctx, "Input command received: type=%s, action=%s",
							command.EventType, command.Action)

						// Procesar comando a través del RemoteControlAgent
						err := a.remoteControlAgent.ProcessInputCommand(command)
						if err != nil {
							runtime.LogErrorf(a.ctx, "Failed to process input command: %v", err)
						}
					})

					runtime.LogInfof(a.ctx, "Remote control handlers configured successfully")
				} else {
					runtime.LogInfof(a.ctx, "APIClient is nil, handlers will be configured after connection")
				}
			} else {
				runtime.LogInfof(a.ctx, "APIClient interface is nil, handlers will be configured after connection")
			}
		} else {
			runtime.LogInfof(a.ctx, "Connection service does not implement GetAPIClient interface")
		}
	} else {
		runtime.LogInfof(a.ctx, "Connection service is nil, handlers will be configured after connection")
	}
}

// shutdown es llamado cuando la app se cierra (Wails)
func (a *App) shutdown(ctx context.Context) {
	// Detener heartbeat automático
	a.stopHeartbeat()

	if err := a.appController.Shutdown(); err != nil {
		runtime.LogErrorf(ctx, "Error during shutdown: %v", err)
	}

	runtime.LogInfof(ctx, "App shutdown completed")
}

// ===== MÉTODOS EXPUESTOS A WAILS (Frontend) =====

// Login maneja el login del usuario - ahora conecta automáticamente al servidor
func (a *App) Login(username, password string) map[string]interface{} {
	runtime.LogInfof(a.ctx, "Starting login process for user: %s", username)

	// 1. Verificar si ya está conectado, si no, conectar al servidor
	serverURL := "http://localhost:8080" // URL completa con esquema

	// Verificar estado de conexión actual
	connectionStatus := a.appController.GetConnectionStatus()
	isConnected := false
	if connectionStatus.ConnectionInfo != nil {
		isConnected = connectionStatus.ConnectionInfo.IsConnected
	}

	if !isConnected {
		runtime.LogInfof(a.ctx, "Not connected, attempting to connect to server...")
		connectResponse := a.appController.Connect(serverURL)

		if !connectResponse.Success {
			runtime.LogErrorf(a.ctx, "Failed to connect to server: %s", connectResponse.Error)
			return map[string]interface{}{
				"success": false,
				"error":   "No se pudo conectar al servidor: " + connectResponse.Error,
			}
		}
		runtime.LogInfof(a.ctx, "Connected to server successfully")
	} else {
		runtime.LogInfof(a.ctx, "Already connected to server")
	}

	// 2. Configurar handler de control remoto después de verificar conexión
	a.setupRemoteControlHandler()

	// 3. Autenticar vía WebSocket con el servidor
	if connectionService := a.appController.GetConnectionService(); connectionService != nil {
		if realService, ok := connectionService.(interface{ GetAPIClient() interface{} }); ok {
			if apiClientInterface := realService.GetAPIClient(); apiClientInterface != nil {
				if apiClient, ok := apiClientInterface.(*api.APIClient); ok {
					runtime.LogInfof(a.ctx, "Authenticating with server...")

					authResponse, err := apiClient.ConnectAndAuthenticate(username, password)
					if err != nil {
						runtime.LogErrorf(a.ctx, "Authentication failed: %v", err)
						return map[string]interface{}{
							"success": false,
							"error":   "Error de autenticación: " + err.Error(),
						}
					}

					if !authResponse.Success {
						runtime.LogErrorf(a.ctx, "Authentication rejected: %s", authResponse.Error)
						return map[string]interface{}{
							"success": false,
							"error":   "Credenciales inválidas: " + authResponse.Error,
						}
					}

					runtime.LogInfof(a.ctx, "Authentication successful for user: %s", username)

					// 4. Realizar autenticación local solo si el servidor acepta
					localAuthResponse := a.appController.Login(username, password)
					if !localAuthResponse.Success {
						runtime.LogErrorf(a.ctx, "Local authentication failed: %s", localAuthResponse.Error)
						return map[string]interface{}{
							"success": false,
							"error":   "Error de autenticación local: " + localAuthResponse.Error,
						}
					}

					// 5. Emitir evento de login exitoso
					runtime.EventsEmit(a.ctx, "login_successful", map[string]interface{}{
						"username":  username,
						"userId":    authResponse.UserID,
						"token":     authResponse.Token,
						"serverUrl": serverURL,
					})

					// 6. Iniciar heartbeat automático
					a.startHeartbeat()

					// NOTA: Ya no registramos automáticamente el PC aquí
					// El usuario debe usar el botón "Registrar PC" en la UI
					runtime.LogInfof(a.ctx, "Login completed. Use 'Register PC' button to register this computer.")

					return map[string]interface{}{
						"success": true,
						"message": "Login exitoso",
						"user": map[string]interface{}{
							"id":       authResponse.UserID,
							"username": username,
						},
						"session_id": authResponse.Token,
						"server_url": serverURL,
					}
				}
			}
		}
	}

	return map[string]interface{}{
		"success": false,
		"error":   "Error interno: No se pudo obtener cliente API",
	}
}

// Logout maneja el logout del usuario
func (a *App) Logout() map[string]interface{} {
	runtime.LogInfof(a.ctx, "Starting logout process")

	// 1. Detener heartbeat automático
	a.stopHeartbeat()

	// 2. Desconectar del servidor
	disconnectResponse := a.appController.Disconnect()
	if !disconnectResponse.Success {
		runtime.LogWarningf(a.ctx, "Failed to disconnect cleanly: %s", disconnectResponse.Error)
	}

	// 3. Logout local
	logoutResponse := a.appController.Logout()

	// 4. Emitir evento de logout
	runtime.EventsEmit(a.ctx, "logout_completed", map[string]interface{}{
		"reason": "user_requested",
	})

	return map[string]interface{}{
		"success": logoutResponse.Success,
		"message": logoutResponse.Message,
		"error":   logoutResponse.Error,
	}
}

// Connect conecta al servidor (método simplificado)
func (a *App) Connect(serverURL string) map[string]interface{} {
	response := a.appController.Connect(serverURL)

	// Configurar handler de control remoto después de conectar
	if response.Success {
		a.setupRemoteControlHandler()
	}

	return map[string]interface{}{
		"success":      response.Success,
		"error":        response.Error,
		"is_connected": response.Success, // Si success es true, está conectado
	}
}

// Disconnect desconecta del servidor
func (a *App) Disconnect() map[string]interface{} {
	response := a.appController.Disconnect()
	return map[string]interface{}{
		"success": response.Success,
		"message": response.Message,
		"error":   response.Error,
	}
}

// GetConnectionStatus obtiene el estado de conexión
func (a *App) GetConnectionStatus() map[string]interface{} {
	response := a.appController.GetConnectionStatus()

	// Simplificar respuesta para evitar problemas de bindings
	result := map[string]interface{}{
		"success": true, // GetStatus siempre retorna info
		"error":   "",
	}

	if response.ConnectionInfo != nil {
		result["connection_info"] = map[string]interface{}{
			"is_connected":    response.ConnectionInfo.IsConnected,
			"server_url":      response.ConnectionInfo.ServerURL,
			"connection_time": response.ConnectionInfo.ConnectionTime,
		}

		if response.ConnectionInfo.ConnectedAt != nil {
			result["connected_at"] = response.ConnectionInfo.ConnectedAt.Unix()
		}
	}

	return result
}

// RegisterPC registra el PC en el servidor
func (a *App) RegisterPC() map[string]interface{} {
	response := a.appController.RegisterPC()

	result := map[string]interface{}{
		"success": response.Success,
		"error":   response.Error,
	}

	if response.PCInfo != nil {
		result["pc_info"] = map[string]interface{}{
			"identifier": response.PCInfo.Identifier(),
			"hostname":   response.PCInfo.Hostname(),
			"os":         response.PCInfo.OSName(),
			"version":    response.PCInfo.OSVersion(),
			"arch":       response.PCInfo.Architecture(),
			"ip":         response.PCInfo.IPAddress(),
		}
	}

	return result
}

// GetPCInfo obtiene información del PC
func (a *App) GetPCInfo() map[string]interface{} {
	response := a.appController.GetPCInfo()

	result := map[string]interface{}{
		"success": response.Success,
		"error":   response.Error,
	}

	if response.PCInfo != nil {
		result["pc_info"] = map[string]interface{}{
			"identifier": response.PCInfo.Identifier(),
			"hostname":   response.PCInfo.Hostname(),
			"os":         response.PCInfo.OSName(),
			"version":    response.PCInfo.OSVersion(),
			"arch":       response.PCInfo.Architecture(),
			"ip":         response.PCInfo.IPAddress(),
		}
	}

	return result
}

// GetAppStatus obtiene el estado general de la aplicación
func (a *App) GetAppStatus() map[string]interface{} {
	response := a.appController.GetAppStatus()

	return map[string]interface{}{
		"is_initialized":   response.IsInitialized,
		"is_authenticated": response.IsAuthenticated,
		"is_connected":     response.IsConnected,
		"uptime":           response.Uptime.Seconds(),
		"start_time":       response.StartTime.Unix(),
	}
}

// GetSystemInfo obtiene información del sistema
func (a *App) GetSystemInfo() map[string]interface{} {
	systemInfo := a.appController.GetSystemInfo()
	result := make(map[string]interface{})
	for key, value := range systemInfo {
		result[key] = value
	}
	return result
}

// IsAuthenticated verifica si está autenticado
func (a *App) IsAuthenticated() bool {
	return a.appController.IsAuthenticated()
}

// ===== HEARTBEAT AUTOMÁTICO =====

// startHeartbeat inicia el heartbeat automático cada 30 segundos
func (a *App) startHeartbeat() {
	// Detener heartbeat anterior si existe
	a.stopHeartbeat()

	if a.apiClient == nil {
		runtime.LogWarningf(a.ctx, "Cannot start heartbeat: API client is nil")
		return
	}

	// Crear ticker para heartbeat cada 30 segundos
	a.heartbeatTicker = time.NewTicker(30 * time.Second)

	// Iniciar goroutine para enviar heartbeats
	go func() {
		runtime.LogInfof(a.ctx, "Heartbeat automático iniciado (cada 30 segundos)")

		for range a.heartbeatTicker.C {
			if a.apiClient != nil && a.apiClient.IsConnected() {
				err := a.apiClient.SendHeartbeat()
				if err != nil {
					runtime.LogErrorf(a.ctx, "Heartbeat failed: %v", err)
				} else {
					runtime.LogDebugf(a.ctx, "Heartbeat sent successfully")
				}
			} else {
				runtime.LogWarningf(a.ctx, "Heartbeat skipped: API client not connected")
			}
		}
	}()
}

// stopHeartbeat detiene el heartbeat automático
func (a *App) stopHeartbeat() {
	if a.heartbeatTicker != nil {
		a.heartbeatTicker.Stop()
		a.heartbeatTicker = nil
		runtime.LogInfof(a.ctx, "Heartbeat automático detenido")
	}
}

// ===== MÉTODOS DE CONTROL REMOTO =====

// AcceptControlRequest acepta una solicitud de control remoto
func (a *App) AcceptControlRequest(sessionID string) map[string]interface{} {
	if a.apiClient == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "No hay conexión con el servidor",
		}
	}

	err := a.apiClient.AcceptRemoteControlSession(sessionID)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to accept control request: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	runtime.LogInfof(a.ctx, "Control request accepted for session: %s", sessionID)

	// Emitir evento de sesión aceptada
	runtime.EventsEmit(a.ctx, "control_session_accepted", map[string]interface{}{
		"sessionId": sessionID,
	})

	return map[string]interface{}{
		"success": true,
		"message": "Solicitud de control remoto aceptada",
	}
}

// RejectControlRequest rechaza una solicitud de control remoto
func (a *App) RejectControlRequest(sessionID string, reason string) map[string]interface{} {
	if a.apiClient == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "No hay conexión con el servidor",
		}
	}

	if reason == "" {
		reason = "Usuario rechazó la solicitud"
	}

	err := a.apiClient.RejectRemoteControlSession(sessionID, reason)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to reject control request: %v", err)
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	runtime.LogInfof(a.ctx, "Control request rejected for session: %s, reason: %s", sessionID, reason)

	// Emitir evento de sesión rechazada
	runtime.EventsEmit(a.ctx, "control_session_rejected", map[string]interface{}{
		"sessionId": sessionID,
		"reason":    reason,
	})

	return map[string]interface{}{
		"success": true,
		"message": "Solicitud de control remoto rechazada",
	}
}

// ===== CONFIGURACIÓN DE EVENTOS PARA UI =====

// setupUIEventBindings configura los eventos para la UI
func (a *App) setupUIEventBindings(uiObserver *WailsUIObserver) {
	// Eventos que deben notificarse a la UI
	uiEventTypes := []string{
		"login_successful",
		"logout_completed",
		"connection_established",
		"connection_terminated",
		"connection_error",
		"pc_registered",
		"app_initialized",
	}

	for _, eventType := range uiEventTypes {
		a.eventManager.Subscribe(eventType, uiObserver)
	}
}

// ===== OBSERVADOR PARA EVENTOS DE UI (Wails) =====

// WailsUIObserver maneja eventos para la UI de Wails
type WailsUIObserver struct {
	ctx context.Context
}

// OnEvent implementa Observer.OnEvent para eventos de UI
func (w *WailsUIObserver) OnEvent(event observer.Event) error {
	// Enviar evento a la UI usando Wails Events
	runtime.EventsEmit(w.ctx, event.Type, event.Data)

	// Log para debugging
	runtime.LogDebugf(w.ctx, "UI Event emitted: %s", event.Type)

	return nil
}

// GetID implementa Observer.GetID
func (w *WailsUIObserver) GetID() string {
	return "wails_ui_observer"
}

// ===== STREAMING DE PANTALLA =====

// startScreenStreaming inicia el streaming de pantalla durante una sesión activa
func (a *App) startScreenStreaming() {
	runtime.LogInfof(a.ctx, "📹 Starting screen streaming...")

	// Obtener canal de frames del RemoteControlAgent
	frameOutput := a.remoteControlAgent.GetFrameOutput()

	// Obtener el sessionID actual al inicio del streaming
	currentSessionID := a.remoteControlAgent.GetActiveSessionID()
	runtime.LogInfof(a.ctx, "📹 Screen streaming for session: %s", currentSessionID)

	for frame := range frameOutput {
		// Verificar si la sesión sigue activa Y es la misma sesión
		if !a.remoteControlAgent.IsActive() {
			runtime.LogInfof(a.ctx, "🔚 Screen streaming stopped - session no longer active")
			break
		}

		// Verificar que el frame pertenece a la sesión actual
		if frame.SessionID != currentSessionID {
			runtime.LogWarningf(a.ctx, "⚠️ Dropping frame for old session %s (current: %s)",
				frame.SessionID, currentSessionID)
			continue
		}

		// Verificar que aún coincide con la sesión activa del agente
		activeSessionID := a.remoteControlAgent.GetActiveSessionID()
		if frame.SessionID != activeSessionID {
			runtime.LogWarningf(a.ctx, "⚠️ Dropping frame for mismatched session %s (active: %s)",
				frame.SessionID, activeSessionID)
			continue
		}

		// Enviar frame al servidor de forma asíncrona
		if a.apiClient != nil {
			a.apiClient.SendScreenFrameAsync(frame)
		} else {
			runtime.LogWarningf(a.ctx, "⚠️ Cannot send screen frame: API client is nil")
			break
		}
	}

	runtime.LogInfof(a.ctx, "📹 Screen streaming ended for session: %s", currentSessionID)
}

// ===== MÉTODOS EXPUESTOS PARA CONTROL REMOTO =====

// GetRemoteControlStatus obtiene el estado del control remoto
func (a *App) GetRemoteControlStatus() map[string]interface{} {
	if a.remoteControlAgent == nil {
		return map[string]interface{}{
			"active":     false,
			"session_id": "",
			"error":      "Remote control agent not initialized",
		}
	}

	return map[string]interface{}{
		"active":       a.remoteControlAgent.IsActive(),
		"session_id":   a.remoteControlAgent.GetActiveSessionID(),
		"capabilities": a.remoteControlAgent.GetCapabilities(),
	}
}

// SetRemoteControlSettings configura los ajustes del control remoto
func (a *App) SetRemoteControlSettings(fps int, quality int) map[string]interface{} {
	if a.remoteControlAgent == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Remote control agent not initialized",
		}
	}

	// Configurar FPS
	if fps > 0 {
		a.remoteControlAgent.SetFrameRate(fps)
	}

	// Configurar calidad JPEG
	if quality > 0 {
		a.remoteControlAgent.SetJPEGQuality(quality)
	}

	runtime.LogInfof(a.ctx, "🎛️ Remote control settings updated: FPS=%d, Quality=%d", fps, quality)

	return map[string]interface{}{
		"success": true,
		"message": "Settings updated successfully",
		"settings": map[string]interface{}{
			"fps":     fps,
			"quality": quality,
		},
	}
}

// TestRemoteControlCapabilities realiza pruebas de las capacidades del control remoto
func (a *App) TestRemoteControlCapabilities() map[string]interface{} {
	if a.remoteControlAgent == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "Remote control agent not initialized",
		}
	}

	runtime.LogInfof(a.ctx, "🧪 Testing remote control capabilities...")

	results := map[string]interface{}{
		"success": true,
		"tests":   map[string]interface{}{},
	}

	// Test screen capture
	err := a.remoteControlAgent.TestScreenCapture()
	if err != nil {
		results["tests"].(map[string]interface{})["screen_capture"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		results["success"] = false
	} else {
		results["tests"].(map[string]interface{})["screen_capture"] = map[string]interface{}{
			"success": true,
			"message": "Screen capture test passed",
		}
	}

	// Test input simulation
	err = a.remoteControlAgent.TestInputSimulation()
	if err != nil {
		results["tests"].(map[string]interface{})["input_simulation"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		results["success"] = false
	} else {
		results["tests"].(map[string]interface{})["input_simulation"] = map[string]interface{}{
			"success": true,
			"message": "Input simulation test passed",
		}
	}

	runtime.LogInfof(a.ctx, "🧪 Remote control capabilities test completed: %v", results["success"])

	return results
}
