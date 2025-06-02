package main

import (
	"context"

	"EscritorioRemoto-Cliente/internal/controller"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/factory"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/singleton"
	"EscritorioRemoto-Cliente/pkg/api"

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
		appController: appController,
		eventManager:  eventManager,
		configManager: configManager,
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

					runtime.LogInfof(a.ctx, "Remote control handler configured successfully")
				} else {
					runtime.LogInfof(a.ctx, "APIClient is nil, remote control handler will be configured after connection")
				}
			} else {
				runtime.LogInfof(a.ctx, "APIClient interface is nil, remote control handler will be configured after connection")
			}
		} else {
			runtime.LogInfof(a.ctx, "Connection service does not implement GetAPIClient interface")
		}
	} else {
		runtime.LogInfof(a.ctx, "Connection service is nil, remote control handler will be configured after connection")
	}
}

// shutdown es llamado cuando la app se cierra (Wails)
func (a *App) shutdown(ctx context.Context) {
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

	// 1. Desconectar del servidor
	disconnectResponse := a.appController.Disconnect()
	if !disconnectResponse.Success {
		runtime.LogWarningf(a.ctx, "Failed to disconnect cleanly: %s", disconnectResponse.Error)
	}

	// 2. Logout local
	logoutResponse := a.appController.Logout()

	// 3. Emitir evento de logout
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
