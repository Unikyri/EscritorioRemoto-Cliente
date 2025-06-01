package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// SessionData contiene la información de la sesión del cliente
type SessionData struct {
	Token    string `json:"token"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
	PCID     string `json:"pcId,omitempty"`
}

// SessionManager maneja la persistencia de la sesión
type SessionManager struct {
	sessionFile string
	data        *SessionData
	mutex       sync.RWMutex
}

// NewSessionManager crea un nuevo gestor de sesión
func NewSessionManager() *SessionManager {
	// Obtener directorio de datos de usuario
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	sessionDir := filepath.Join(homeDir, ".escritorio-remoto")
	os.MkdirAll(sessionDir, 0755)

	sessionFile := filepath.Join(sessionDir, "session.json")

	sm := &SessionManager{
		sessionFile: sessionFile,
		data:        &SessionData{},
	}

	// Cargar sesión existente si existe
	sm.loadSession()

	return sm
}

// StoreToken guarda el token de autenticación
func (sm *SessionManager) StoreToken(token, userID, username string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.data.Token = token
	sm.data.UserID = userID
	sm.data.Username = username

	return sm.saveSession()
}

// StorePCID guarda el ID del PC registrado
func (sm *SessionManager) StorePCID(pcID string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.data.PCID = pcID

	return sm.saveSession()
}

// GetToken obtiene el token de autenticación
func (sm *SessionManager) GetToken() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.Token
}

// GetUserID obtiene el ID del usuario
func (sm *SessionManager) GetUserID() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.UserID
}

// GetUsername obtiene el nombre de usuario
func (sm *SessionManager) GetUsername() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.Username
}

// GetPCID obtiene el ID del PC
func (sm *SessionManager) GetPCID() string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.PCID
}

// IsAuthenticated verifica si hay una sesión válida
func (sm *SessionManager) IsAuthenticated() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.Token != "" && sm.data.UserID != ""
}

// IsRegistered verifica si el PC está registrado
func (sm *SessionManager) IsRegistered() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return sm.data.PCID != ""
}

// GetSessionData obtiene una copia de los datos de sesión
func (sm *SessionManager) GetSessionData() SessionData {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return *sm.data
}

// ClearSession limpia la sesión
func (sm *SessionManager) ClearSession() error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	sm.data = &SessionData{}

	// Eliminar archivo de sesión
	if err := os.Remove(sm.sessionFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// saveSession guarda la sesión en el archivo
func (sm *SessionManager) saveSession() error {
	data, err := json.MarshalIndent(sm.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sm.sessionFile, data, 0600)
}

// loadSession carga la sesión desde el archivo
func (sm *SessionManager) loadSession() {
	data, err := os.ReadFile(sm.sessionFile)
	if err != nil {
		return // Archivo no existe o error de lectura, usar datos vacíos
	}

	var sessionData SessionData
	if err := json.Unmarshal(data, &sessionData); err != nil {
		return // Error de parsing, usar datos vacíos
	}

	sm.data = &sessionData
}
