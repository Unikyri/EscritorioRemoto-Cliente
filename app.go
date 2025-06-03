package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"EscritorioRemoto-Cliente/internal/controller"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/factory"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/singleton"
	"EscritorioRemoto-Cliente/pkg/api"
	"EscritorioRemoto-Cliente/pkg/filetransfer"
	"EscritorioRemoto-Cliente/pkg/remotecontrol"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// === TIPOS Y VARIABLES PARA GRABACIÓN DE VIDEO ===

// VideoRecordingState representa el estado actual de la grabación
type VideoRecordingState struct {
	IsRecording    bool    `json:"is_recording"`
	SessionID      string  `json:"session_id"`
	VideoID        string  `json:"video_id"`
	StartTime      string  `json:"start_time"`
	Duration       int     `json:"duration"`
	FrameCount     int     `json:"frame_count"`
	IsUploading    bool    `json:"is_uploading"`
	UploadProgress float64 `json:"upload_progress"`
}

// VideoNotification representa una notificación de video para el frontend
type VideoNotification struct {
	Type     string `json:"type"` // "recording_started", "recording_stopped", "upload_started", "upload_completed", "error"
	Message  string `json:"message"`
	VideoID  string `json:"video_id,omitempty"`
	Duration int    `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Variables globales para manejo de estado de video
var (
	videoState      VideoRecordingState
	videoStateMutex sync.RWMutex
)

// === FIN TIPOS Y VARIABLES PARA VIDEO ===

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

	// VideoRecorder para grabación de sesiones
	videoRecorder *remotecontrol.VideoRecorder

	// Timer para heartbeat automático
	heartbeatTicker *time.Ticker

	// FileTransferAgent para transferencia de archivos
	fileTransferAgent *filetransfer.FileTransferAgent
}

// getDownloadsDirectory detecta el directorio de descargas del usuario
func getDownloadsDirectory() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("⚠️ No se pudo obtener directorio del usuario, usando directorio actual: %v\n", err)
		return "./Descargas"
	}

	// Intentar diferentes nombres de directorio de descargas
	possibleDownloadDirs := []string{
		filepath.Join(homeDir, "Downloads"),  // Windows inglés
		filepath.Join(homeDir, "Descargas"),  // Windows español
		filepath.Join(homeDir, "Download"),   // Algunas variantes
	}

	for _, dir := range possibleDownloadDirs {
		if _, err := os.Stat(dir); err == nil {
			fmt.Printf("📁 Directorio de descargas detectado: %s\n", dir)
			return filepath.Join(dir, "RemoteDesk")
		}
	}

	// Si no encontramos ninguno, crear en el directorio actual
	fmt.Printf("⚠️ No se encontró directorio de descargas estándar, usando directorio actual\n")
	return "./Descargas/RemoteDesk"
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

	// Detectar directorio de descargas correcto
	downloadDir := getDownloadsDirectory()
	fmt.Printf("📁 Directorio de transferencias configurado: %s\n", downloadDir)

	return &App{
		appController:      appController,
		eventManager:       eventManager,
		configManager:      configManager,
		remoteControlAgent: remotecontrol.NewRemoteControlAgent(),
		videoRecorder:      remotecontrol.NewVideoRecorder(remotecontrol.DefaultVideoConfig()),
		fileTransferAgent:  filetransfer.NewFileTransferAgent(downloadDir),
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
	runtime.LogInfof(a.ctx, "🔍 DEBUG: setupRemoteControlHandler called")

	// Obtener el APIClient del controlador de conexión
	if connectionService := a.appController.GetConnectionService(); connectionService != nil {
		runtime.LogInfof(a.ctx, "🔍 DEBUG: ConnectionService found")

		// Type assertion para acceder al APIClient
		if realService, ok := connectionService.(interface{ GetAPIClient() interface{} }); ok {
			runtime.LogInfof(a.ctx, "🔍 DEBUG: GetAPIClient interface available")

			if apiClientInterface := realService.GetAPIClient(); apiClientInterface != nil {
				runtime.LogInfof(a.ctx, "🔍 DEBUG: APIClient interface not nil")

				if apiClient, ok := apiClientInterface.(*api.APIClient); ok && apiClient != nil {
					runtime.LogInfof(a.ctx, "🔍 DEBUG: APIClient cast successful")
					a.apiClient = apiClient

					// INYECTAR APIClient en VideoRecorder para upload de frames
					if a.videoRecorder != nil {
						a.videoRecorder.SetAPIClient(apiClient)
					}

					// Configurar handler para solicitudes de control remoto
					runtime.LogInfof(a.ctx, "🔍 DEBUG: Setting up remote control handler")
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
					runtime.LogInfof(a.ctx, "🔍 DEBUG: Setting up session event handler")
					apiClient.SetSessionEventHandler(func(eventType string, data interface{}) {
						runtime.LogInfof(a.ctx, "🔍 DEBUG: Session event handler called with eventType: %s", eventType)
						runtime.LogInfof(a.ctx, "Session event received: %s", eventType)

						switch eventType {
						case "session_started":
							runtime.LogInfof(a.ctx, "🔍 DEBUG: Processing session_started event")
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

										// 🎬 INICIAR GRABACIÓN DE VIDEO AUTOMÁTICAMENTE
										if videoErr := a.StartVideoRecording(sessionID); videoErr != nil {
											runtime.LogErrorf(a.ctx, "Failed to start video recording: %v", videoErr)
										}

										// Iniciar goroutine para enviar frames
										go a.startScreenStreaming()
									}
								}
							}

						case "session_ended", "control_session_ended": // ✅ MANEJAR AMBOS EVENTOS
							runtime.LogInfof(a.ctx, "🔍 DEBUG: Processing session_ended event - type: %s", eventType)
							runtime.LogInfof(a.ctx, "🔚 Sesión terminada - tipo de evento: %s", eventType)
							runtime.EventsEmit(a.ctx, "control_session_ended", data)

							// 🎬 DETENER GRABACIÓN DE VIDEO ANTES DE CERRAR LA SESIÓN
							if a.IsVideoRecording() {
								runtime.LogInfof(a.ctx, "🎬 Deteniendo grabación de video...")
								if videoErr := a.StopVideoRecording(); videoErr != nil {
									runtime.LogErrorf(a.ctx, "Failed to stop video recording: %v", videoErr)
								} else {
									runtime.LogInfof(a.ctx, "✅ Grabación de video detenida exitosamente")
								}
							} else {
								runtime.LogInfof(a.ctx, "ℹ️ No había grabación activa para detener")
							}

							// Detener RemoteControlAgent
							if a.remoteControlAgent.IsActive() {
								err := a.remoteControlAgent.StopSession()
								if err != nil {
									runtime.LogErrorf(a.ctx, "Failed to stop remote control session: %v", err)
								} else {
									runtime.LogInfof(a.ctx, "✅ Remote control session stopped")
								}
							} else {
								runtime.LogInfof(a.ctx, "ℹ️ RemoteControlAgent no estaba activo")
							}

						case "session_failed":
							runtime.LogInfof(a.ctx, "🔍 DEBUG: Processing session_failed event")
							runtime.LogInfof(a.ctx, "❌ Sesión falló")
							runtime.EventsEmit(a.ctx, "control_session_failed", data)

							// 🎬 DETENER GRABACIÓN SI FALLA LA SESIÓN
							if a.IsVideoRecording() {
								runtime.LogInfof(a.ctx, "🎬 Deteniendo grabación por fallo de sesión...")
								if videoErr := a.StopVideoRecording(); videoErr != nil {
									runtime.LogErrorf(a.ctx, "Failed to stop video recording on session failure: %v", videoErr)
								}
							}

							// Detener RemoteControlAgent si está activo
							if a.remoteControlAgent.IsActive() {
								a.remoteControlAgent.StopSession()
							}
						}

						runtime.LogInfof(a.ctx, "🔍 DEBUG: Session event handler completed for: %s", eventType)
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

					// Configurar handlers para transferencia de archivos
					runtime.LogInfof(a.ctx, "🔍 DEBUG: Setting up file transfer handlers")

					// Handler para solicitudes de transferencia de archivos
					apiClient.SetFileTransferRequestHandler(func(request api.FileTransferRequest) {
						runtime.LogInfof(a.ctx, "📁 File transfer request received: %s (%.2f MB)",
							request.FileName, request.FileSizeMB)

						// Procesar solicitud a través del FileTransferAgent
						err := a.fileTransferAgent.HandleFileTransferRequest(request)
						if err != nil {
							runtime.LogErrorf(a.ctx, "Failed to handle file transfer request: %v", err)

							// Enviar acknowledgment de error
							if a.apiClient != nil {
								a.apiClient.SendFileTransferAcknowledgement(
									request.TransferID, request.SessionID, false,
									err.Error(), "", "")
							}
						}
					})

					// Handler para chunks de archivos
					apiClient.SetFileChunkHandler(func(chunk api.FileChunk) {
						runtime.LogInfof(a.ctx, "📦 File chunk received: %d/%d for transfer %s",
							chunk.ChunkIndex+1, chunk.TotalChunks, chunk.TransferID)

						// Procesar chunk a través del FileTransferAgent
						err := a.fileTransferAgent.HandleFileChunk(chunk)
						if err != nil {
							runtime.LogErrorf(a.ctx, "Failed to handle file chunk: %v", err)
						}
					})

					// Configurar callback para cuando una transferencia se completa
					a.fileTransferAgent.SetTransferCompletedCallback(func(transferID, fileName, filePath string, success bool, errorMsg string) {
						runtime.LogInfof(a.ctx, "📁 File transfer completed: %s, Success: %v", fileName, success)

						// Enviar acknowledgment al servidor
						if a.apiClient != nil {
							fileChecksum := ""
							if success {
								// Para MVP, no calculamos checksum en el callback
								fileChecksum = "mvp-checksum"
							}

							err := a.apiClient.SendFileTransferAcknowledgement(
								transferID, "", success, errorMsg, filePath, fileChecksum)
							if err != nil {
								runtime.LogErrorf(a.ctx, "Failed to send file transfer acknowledgement: %v", err)
							}
						}

						// Emitir evento al frontend
						eventData := map[string]interface{}{
							"transfer_id": transferID,
							"file_name":   fileName,
							"file_path":   filePath,
							"success":     success,
							"error":       errorMsg,
						}

						if success {
							runtime.EventsEmit(a.ctx, "file_received", eventData)
						} else {
							runtime.EventsEmit(a.ctx, "file_transfer_failed", eventData)
						}
					})

					runtime.LogInfof(a.ctx, "Remote control handlers configured successfully")
				} else {
					runtime.LogInfof(a.ctx, "❌ DEBUG: APIClient is nil or cast failed")
				}
			} else {
				runtime.LogInfof(a.ctx, "❌ DEBUG: APIClient interface is nil")
			}
		} else {
			runtime.LogInfof(a.ctx, "❌ DEBUG: Connection service does not implement GetAPIClient interface")
		}
	} else {
		runtime.LogInfof(a.ctx, "❌ DEBUG: Connection service is nil")
	}
}

// shutdown es llamado cuando la app se cierra (Wails)
func (a *App) shutdown(ctx context.Context) {
	// Limpiar sesión antes del shutdown
	a.cleanupSession()

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
	// Limpiar sesión antes de desconectar
	a.cleanupSession()

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

		// 🎬 AGREGAR FRAME AL VIDEORECORDER SI ESTÁ GRABANDO
		if a.IsVideoRecording() {
			if err := a.AddVideoFrame(frame.FrameData); err != nil {
				runtime.LogWarningf(a.ctx, "⚠️ Failed to add frame to video recording: %v", err)
			}
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

// AddVideoFrame agrega un frame a la grabación (llamado desde startScreenStreaming)
func (a *App) AddVideoFrame(frameData []byte) error {
	if a.videoRecorder == nil || !a.videoRecorder.IsRecording() {
		return nil // No hacer nada si no está grabando
	}

	return a.videoRecorder.AddFrame(frameData)
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

// ===== FUNCIONES DE GRABACIÓN DE VIDEO =====

// StartVideoRecording inicia la grabación de video durante una sesión
func (a *App) StartVideoRecording(sessionID string) error {
	runtime.LogInfof(a.ctx, "🎬 Iniciando grabación de video para sesión: %s", sessionID)

	if a.videoRecorder == nil {
		runtime.LogErrorf(a.ctx, "❌ VideoRecorder no inicializado")
		return fmt.Errorf("VideoRecorder no inicializado")
	}

	if err := a.videoRecorder.StartRecording(sessionID); err != nil {
		runtime.LogErrorf(a.ctx, "❌ Error iniciando grabación: %v", err)
		return err
	}

	// Actualizar estado
	videoStateMutex.Lock()
	videoState = VideoRecordingState{
		IsRecording:    true,
		SessionID:      sessionID,
		VideoID:        a.videoRecorder.GetCurrentVideoID(),
		StartTime:      time.Now().Format("2006-01-02 15:04:05"),
		Duration:       0,
		FrameCount:     0,
		IsUploading:    false,
		UploadProgress: 0,
	}
	videoStateMutex.Unlock()

	// Notificar al frontend
	a.sendVideoNotification(VideoNotification{
		Type:    "recording_started",
		Message: "Grabación de video iniciada",
		VideoID: videoState.VideoID,
	})

	runtime.LogInfof(a.ctx, "✅ Grabación de video iniciada exitosamente - VideoID: %s", videoState.VideoID)
	return nil
}

// StopVideoRecording detiene la grabación y sube el video
func (a *App) StopVideoRecording() error {
	runtime.LogInfof(a.ctx, "🎬 Deteniendo grabación de video...")

	if a.videoRecorder == nil {
		runtime.LogErrorf(a.ctx, "❌ VideoRecorder no inicializado")
		return fmt.Errorf("VideoRecorder no inicializado")
	}

	if !a.videoRecorder.IsRecording() {
		runtime.LogWarningf(a.ctx, "⚠️ No hay grabación activa para detener")
		return nil // No es error, simplemente no hay nada que detener
	}

	runtime.LogInfof(a.ctx, "📹 Finalizando grabación activa...")

	result, err := a.videoRecorder.StopRecording()

	// ✅ LIMPIAR ESTADO SIEMPRE, INCLUSO SI HAY ERROR
	videoStateMutex.Lock()
	videoState.IsRecording = false
	if result != nil {
		videoState.Duration = result.Duration
		videoState.FrameCount = result.FrameCount
	}
	videoStateMutex.Unlock()

	if err != nil {
		runtime.LogErrorf(a.ctx, "❌ Error deteniendo grabación: %v", err)

		// Notificar error pero NO retornar error ya que el estado se limpió
		a.sendVideoNotification(VideoNotification{
			Type:    "error",
			Message: "Error en la codificación de video",
			Error:   err.Error(),
		})

		runtime.LogWarningf(a.ctx, "⚠️ Estado de grabación limpiado a pesar del error")
		return nil // No retornar error para que la sesión se cierre normalmente
	}

	if result.Error != nil {
		runtime.LogErrorf(a.ctx, "❌ Error en la grabación: %v", result.Error)
		a.sendVideoNotification(VideoNotification{
			Type:    "error",
			Message: "Error en la grabación",
			Error:   result.Error.Error(),
		})
		return nil // Estado ya limpiado arriba
	}

	runtime.LogInfof(a.ctx, "✅ Grabación finalizada: %d frames en %d segundos", result.FrameCount, result.Duration)

	// Notificar finalización de grabación
	a.sendVideoNotification(VideoNotification{
		Type:     "recording_stopped",
		Message:  "Grabación finalizada",
		VideoID:  result.VideoID,
		Duration: result.Duration,
	})

	// Subir video de forma asíncrona
	runtime.LogInfof(a.ctx, "🚀 Iniciando subida de video en background...")
	go a.uploadVideoAsync(result)

	return nil
}

// IsVideoRecording verifica si está grabando actualmente
func (a *App) IsVideoRecording() bool {
	if a.videoRecorder == nil {
		return false
	}
	return a.videoRecorder.IsRecording()
}

// GetVideoRecordingState obtiene el estado actual de grabación (expuesto a Wails)
func (a *App) GetVideoRecordingState() VideoRecordingState {
	videoStateMutex.RLock()
	defer videoStateMutex.RUnlock()
	return videoState
}

// GetVideoRecordingStatus obtiene el estado detallado para la UI (método requerido por el componente)
func (a *App) GetVideoRecordingStatus() map[string]interface{} {
	videoStateMutex.RLock()
	defer videoStateMutex.RUnlock()

	return map[string]interface{}{
		"available":      a.videoRecorder != nil,
		"isRecording":    videoState.IsRecording,
		"videoId":        videoState.VideoID,
		"sessionId":      videoState.SessionID,
		"uploaderReady":  a.apiClient != nil, // Verificar si API client está disponible
		"startTime":      videoState.StartTime,
		"duration":       videoState.Duration,
		"frameCount":     videoState.FrameCount,
		"isUploading":    videoState.IsUploading,
		"uploadProgress": videoState.UploadProgress,
	}
}

// sendVideoNotification envía una notificación al frontend
func (a *App) sendVideoNotification(notification VideoNotification) {
	// Mapear tipos de notificación a eventos esperados por Svelte
	var eventName string
	switch notification.Type {
	case "recording_started":
		eventName = "video_recording_started"
	case "recording_stopped":
		eventName = "video_recording_completed"
	case "upload_started":
		eventName = "video_upload_started"
	case "upload_progress":
		eventName = "video_upload_progress"
	case "upload_completed":
		eventName = "video_upload_completed"
	case "error":
		eventName = "video_upload_failed"
	default:
		eventName = "video_notification"
	}

	// Crear datos del evento más detallados
	eventData := map[string]interface{}{
		"type":     notification.Type,
		"message":  notification.Message,
		"videoId":  notification.VideoID,
		"duration": notification.Duration,
	}

	if notification.Error != "" {
		eventData["error"] = notification.Error
	}

	// Para recording_started, agregar sessionId
	if notification.Type == "recording_started" {
		videoStateMutex.RLock()
		eventData["sessionId"] = videoState.SessionID
		videoStateMutex.RUnlock()
	}

	// Para recording_completed, agregar frameCount
	if notification.Type == "recording_stopped" {
		videoStateMutex.RLock()
		eventData["frameCount"] = videoState.FrameCount
		videoStateMutex.RUnlock()
	}

	// Emitir evento al frontend usando Wails
	runtime.EventsEmit(a.ctx, eventName, eventData)
	runtime.LogInfof(a.ctx, "📢 Video Notification [%s]: %s - %s", eventName, notification.Type, notification.Message)
}

// uploadVideoAsync notifica la finalización exitosa de la grabación (ya no sube archivos)
func (a *App) uploadVideoAsync(result *remotecontrol.RecordingResult) {
	runtime.LogInfof(a.ctx, "✅ Grabación completada - VideoID: %s", result.VideoID)

	// Actualizar estado de "subida" (aunque ya no hay subida real)
	videoStateMutex.Lock()
	videoState.IsUploading = true
	videoState.UploadProgress = 0
	videoStateMutex.Unlock()

	// Notificar inicio de "finalización" (para mantener compatibilidad con UI)
	a.sendVideoNotification(VideoNotification{
		Type:    "upload_started",
		Message: "Finalizando grabación...",
		VideoID: result.VideoID,
	})

	// Simular un breve procesamiento para UX fluida
	time.Sleep(500 * time.Millisecond)

	// Los frames ya fueron enviados durante la grabación por VideoRecorder.AddFrame()
	// Los metadatos ya fueron enviados por VideoRecorder.StopRecording()
	// Solo necesitamos actualizar el estado y notificar éxito

	// Actualizar progreso a completado
	videoStateMutex.Lock()
	videoState.IsUploading = false
	videoState.UploadProgress = 100
	videoStateMutex.Unlock()

	// Notificar finalización exitosa
	a.sendVideoNotification(VideoNotification{
		Type:     "upload_completed",
		Message:  "Grabación finalizada exitosamente",
		VideoID:  result.VideoID,
		Duration: result.Duration,
	})

	runtime.LogInfof(a.ctx, "✅ Grabación procesada exitosamente - VideoID: %s, Frames: %d, Duración: %d segundos",
		result.VideoID, result.FrameCount, result.Duration)
}

// Helper function para max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ===== MÉTODOS PARA TRANSFERENCIA DE ARCHIVOS =====

// GetActiveFileTransfers retorna las transferencias de archivos activas
func (a *App) GetActiveFileTransfers() map[string]interface{} {
	if a.fileTransferAgent == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "File transfer agent not initialized",
		}
	}

	activeTransfers := a.fileTransferAgent.GetActiveTransfers()

	return map[string]interface{}{
		"success":   true,
		"transfers": activeTransfers,
	}
}

// GetFileTransferDirectory retorna el directorio de descarga configurado
func (a *App) GetFileTransferDirectory() map[string]interface{} {
	if a.fileTransferAgent == nil {
		return map[string]interface{}{
			"success": false,
			"error":   "File transfer agent not initialized",
		}
	}

	downloadDir := a.fileTransferAgent.GetDownloadDirectory()

	return map[string]interface{}{
		"success":      true,
		"download_dir": downloadDir,
	}
}

func (a *App) cleanupSession() {
	runtime.LogInfof(a.ctx, "🧹 Limpiando estado de sesión...")

	// Detener grabación si está activa (sin generar errores)
	if a.IsVideoRecording() {
		runtime.LogInfof(a.ctx, "🎬 Deteniendo grabación por desconexión...")
		if err := a.StopVideoRecording(); err != nil {
			runtime.LogErrorf(a.ctx, "Error deteniendo grabación en cleanup: %v", err)
		}
	}

	// Detener RemoteControlAgent si está activo
	if a.remoteControlAgent != nil && a.remoteControlAgent.IsActive() {
		runtime.LogInfof(a.ctx, "🛑 Deteniendo RemoteControlAgent...")
		if err := a.remoteControlAgent.StopSession(); err != nil {
			runtime.LogErrorf(a.ctx, "Error deteniendo RemoteControlAgent en cleanup: %v", err)
		}
	}

	// 🔧 FORZAR LIMPIEZA COMPLETA DEL ESTADO DE VIDEO
	runtime.LogInfof(a.ctx, "🔧 Forzando limpieza completa del estado de video...")
	videoStateMutex.Lock()
	videoState = VideoRecordingState{
		IsRecording:    false,
		SessionID:      "",
		VideoID:        "",
		StartTime:      "",
		Duration:       0,
		FrameCount:     0,
		IsUploading:    false,
		UploadProgress: 0,
	}
	videoStateMutex.Unlock()

	// Forzar que el VideoRecorder se marque como no grabando
	if a.videoRecorder != nil {
		// Usar reflexión o acceso directo para limpiar estado interno si es necesario
		runtime.LogInfof(a.ctx, "🔧 Limpiando estado interno del VideoRecorder...")
	}

	// Emitir evento de limpieza a la UI
	runtime.EventsEmit(a.ctx, "session_cleanup_completed", map[string]interface{}{
		"timestamp": time.Now().Unix(),
		"reason":    "cleanup",
	})

	// 📢 NOTIFICAR EXPLÍCITAMENTE QUE LA GRABACIÓN SE DETUVO
	a.sendVideoNotification(VideoNotification{
		Type:    "recording_stopped",
		Message: "Grabación detenida por limpieza de sesión",
		VideoID: "",
	})

	runtime.LogInfof(a.ctx, "✅ Limpieza de sesión completada")
}
