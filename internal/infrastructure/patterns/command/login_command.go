package command

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/model/entities"
	"fmt"
	"time"
)

// LoginCommand implementa el comando de login
type LoginCommand struct {
	BaseCommand
	username     string
	password     string
	authService  AuthServiceInterface
	user         *entities.User // Para poder hacer undo
}

// AuthServiceInterface define la interface del servicio de autenticación
type AuthServiceInterface interface {
	Login(username, password string) (*entities.User, error)
	Logout() error
	IsAuthenticated() bool
}

// NewLoginCommand crea un nuevo comando de login
func NewLoginCommand(username, password string, authService AuthServiceInterface) *LoginCommand {
	return &LoginCommand{
		BaseCommand: NewBaseCommand(
			fmt.Sprintf("Login user: %s", username),
			true, // El login se puede deshacer con logout
		),
		username:    username,
		password:    password,
		authService: authService,
	}
}

// Execute implementa Command.Execute
func (lc *LoginCommand) Execute() error {
	// Verificar si ya está autenticado
	if lc.authService.IsAuthenticated() {
		return fmt.Errorf("user is already logged in")
	}

	// Ejecutar login
	user, err := lc.authService.Login(lc.username, lc.password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	// Guardar usuario para poder hacer undo
	lc.user = user

	// Publicar evento de login exitoso
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "login_command_executed",
		Data: map[string]interface{}{
			"username":  lc.username,
			"user_id":   user.ID(),
			"timestamp": lc.timestamp,
		},
	})

	return nil
}

// Undo implementa Command.Undo
func (lc *LoginCommand) Undo() error {
	// Solo se puede deshacer si hay un usuario logueado
	if lc.user == nil {
		return fmt.Errorf("no login to undo")
	}

	// Verificar si está autenticado
	if !lc.authService.IsAuthenticated() {
		return fmt.Errorf("no user is currently logged in")
	}

	// Ejecutar logout para deshacer el login
	err := lc.authService.Logout()
	if err != nil {
		return fmt.Errorf("failed to undo login: %w", err)
	}

	// Publicar evento de undo
	eventManager := observer.GetInstance()
	eventManager.Publish(observer.Event{
		Type: "login_command_undone",
		Data: map[string]interface{}{
			"username":  lc.username,
			"timestamp": time.Now().UTC(),
		},
	})

	// Limpiar usuario
	lc.user = nil

	return nil
}

// GetUsername retorna el username usado en el comando
func (cmd *LoginCommand) GetUsername() string {
	return cmd.username
}

// GetUser retorna el usuario autenticado (si existe)
func (cmd *LoginCommand) GetUser() *entities.User {
	return cmd.user
} 