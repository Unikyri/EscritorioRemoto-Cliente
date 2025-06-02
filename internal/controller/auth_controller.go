package controller

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/command"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/model/entities"
	"fmt"
	"time"
)

// AuthController maneja las operaciones de autenticación
type AuthController struct {
	authService    AuthService
	commandHistory *command.CommandHistory
	eventManager   *observer.EventManager
}

// AuthService interface para el servicio de autenticación
type AuthService interface {
	Login(username, password string) (*entities.User, error)
	Logout() error
	IsAuthenticated() bool
	GetCurrentUser() *entities.User
	ValidateCredentials(username, password string) error
}

// LoginRequest representa la solicitud de login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse representa la respuesta de login
type LoginResponse struct {
	Success   bool            `json:"success"`
	User      *entities.User  `json:"user,omitempty"`
	Error     string          `json:"error,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
}

// LogoutResponse representa la respuesta de logout
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// NewAuthController crea un nuevo controlador de autenticación
func NewAuthController(authService AuthService) *AuthController {
	return &AuthController{
		authService:    authService,
		commandHistory: command.NewCommandHistory(50),
		eventManager:   observer.GetInstance(),
	}
}

// Login maneja la solicitud de login del usuario
func (ac *AuthController) Login(request LoginRequest) LoginResponse {
	// Validar request
	if err := ac.validateLoginRequest(request); err != nil {
		return LoginResponse{
			Success: false,
			Error:   err.Error(),
		}
	}

	// Verificar si ya está autenticado
	if ac.authService.IsAuthenticated() {
		return LoginResponse{
			Success: false,
			Error:   "User is already logged in",
		}
	}

	// Crear y ejecutar comando de login
	loginCmd := command.NewLoginCommand(
		request.Username, 
		request.Password, 
		ac.authService,
	)

	result := ac.commandHistory.ExecuteCommand(loginCmd)
	
	if !result.Success {
		return LoginResponse{
			Success: false,
			Error:   result.Error.Error(),
		}
	}

	// Obtener usuario autenticado
	user := ac.authService.GetCurrentUser()
	if user == nil {
		return LoginResponse{
			Success: false,
			Error:   "Failed to retrieve authenticated user",
		}
	}

	// Publicar evento de login completado
	ac.eventManager.Publish(observer.Event{
		Type: "auth_login_completed",
		Data: map[string]interface{}{
			"username": user.Username(),
			"user_id":  user.ID(),
			"role":     user.Role(),
		},
	})

	return LoginResponse{
		Success:   true,
		User:      user,
		SessionID: generateSessionID(),
	}
}

// Logout maneja la solicitud de logout del usuario
func (ac *AuthController) Logout() LogoutResponse {
	// Verificar si está autenticado
	if !ac.authService.IsAuthenticated() {
		return LogoutResponse{
			Success: false,
			Error:   "No user is currently logged in",
		}
	}

	// Obtener usuario actual antes del logout
	currentUser := ac.authService.GetCurrentUser()
	username := ""
	if currentUser != nil {
		username = currentUser.Username()
	}

	// Ejecutar logout
	err := ac.authService.Logout()
	if err != nil {
		return LogoutResponse{
			Success: false,
			Error:   fmt.Sprintf("Logout failed: %v", err),
		}
	}

	// Publicar evento de logout
	ac.eventManager.Publish(observer.Event{
		Type: "auth_logout_completed",
		Data: map[string]interface{}{
			"username": username,
			"reason":   "user_requested",
		},
	})

	return LogoutResponse{
		Success: true,
		Message: "Logout successful",
	}
}

// GetCurrentUser retorna el usuario actualmente autenticado
func (ac *AuthController) GetCurrentUser() *entities.User {
	return ac.authService.GetCurrentUser()
}

// IsAuthenticated verifica si hay un usuario autenticado
func (ac *AuthController) IsAuthenticated() bool {
	return ac.authService.IsAuthenticated()
}

// UndoLastAction deshace la última acción realizada
func (ac *AuthController) UndoLastAction() error {
	return ac.commandHistory.UndoLastCommand()
}

// GetCommandHistory retorna el historial de comandos
func (ac *AuthController) GetCommandHistory() []command.Command {
	return ac.commandHistory.GetHistory()
}

// validateLoginRequest valida la solicitud de login
func (ac *AuthController) validateLoginRequest(request LoginRequest) error {
	if request.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if request.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	if len(request.Username) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	if len(request.Password) < 6 {
		return fmt.Errorf("password must be at least 6 characters long")
	}

	// Validar credenciales con el servicio
	return ac.authService.ValidateCredentials(request.Username, request.Password)
}

// generateSessionID genera un ID de sesión simple
func generateSessionID() string {
	// En una implementación real, esto sería más robusto
	return fmt.Sprintf("session_%d", time.Now().Unix())
} 