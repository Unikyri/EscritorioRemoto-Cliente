package factory

import (
	"EscritorioRemoto-Cliente/internal/controller"
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/singleton"
	"EscritorioRemoto-Cliente/internal/model/entities"
	"EscritorioRemoto-Cliente/pkg/api"
	"fmt"
	"os"
	"runtime"
	"time"
)

// ServiceFactory implementa el patrón Factory para crear servicios
type ServiceFactory struct {
	configManager *singleton.ConfigManager
}

// NewServiceFactory crea una nueva instancia del factory
func NewServiceFactory(configManager *singleton.ConfigManager) *ServiceFactory {
	return &ServiceFactory{
		configManager: configManager,
	}
}

// CreateAuthService crea un servicio de autenticación
func (sf *ServiceFactory) CreateAuthService() controller.AuthService {
	// Por ahora retornamos un mock service
	// En una implementación real, aquí se crearían las dependencias reales
	return &MockAuthService{}
}

// CreateConnectionService crea un servicio de conexión
func (sf *ServiceFactory) CreateConnectionService() controller.ConnectionService {
	// Crear servicio real con APIClient
	return &RealConnectionService{
		apiClient: nil, // Se inicializará en Connect
	}
}

// CreatePCService crea un servicio de PC que usa la conexión existente
func (sf *ServiceFactory) CreatePCService(connectionService controller.ConnectionService) controller.PCService {
	// Crear servicio real que usa la conexión existente
	return &RealPCService{
		connectionService: connectionService,
	}
}

// ===== REAL CONNECTION SERVICE =====

// RealConnectionService implementa ConnectionService usando APIClient
type RealConnectionService struct {
	apiClient   *api.APIClient
	connected   bool
	serverURL   string
	connectedAt *time.Time
}

func (r *RealConnectionService) Connect(serverURL string) error {
	r.apiClient = api.NewAPIClient(serverURL)
	err := r.apiClient.Connect()
	if err != nil {
		return err
	}

	r.connected = true
	r.serverURL = serverURL
	now := time.Now()
	r.connectedAt = &now
	return nil
}

func (r *RealConnectionService) Disconnect() error {
	if r.apiClient != nil {
		err := r.apiClient.Disconnect()
		r.apiClient = nil
		r.connected = false
		r.connectedAt = nil
		return err
	}
	return nil
}

func (r *RealConnectionService) IsConnected() bool {
	if r.apiClient != nil {
		return r.apiClient.IsConnected()
	}
	return false
}

func (r *RealConnectionService) GetServerURL() string {
	if r.apiClient != nil {
		return r.apiClient.GetServerURL()
	}
	return r.serverURL
}

func (r *RealConnectionService) SendHeartbeat() error {
	if r.apiClient != nil {
		return r.apiClient.SendHeartbeat()
	}
	return nil
}

func (r *RealConnectionService) GetConnectionInfo() *controller.ConnectionInfo {
	return &controller.ConnectionInfo{
		IsConnected:    r.IsConnected(),
		ServerURL:      r.GetServerURL(),
		ConnectedAt:    r.connectedAt,
		ConnectionTime: r.getConnectionTimeString(),
	}
}

func (r *RealConnectionService) getConnectionTimeString() string {
	if r.connectedAt != nil {
		return time.Since(*r.connectedAt).String()
	}
	return ""
}

// GetAPIClient retorna el cliente API para acceso directo
func (r *RealConnectionService) GetAPIClient() interface{} {
	return r.apiClient
}

// ===== REAL PC SERVICE =====

// RealPCService implementa PCService usando APIClient para comunicación real con el servidor
type RealPCService struct {
	connectionService controller.ConnectionService
	pcInfo            *entities.PCInfo
}

func (r *RealPCService) RegisterPC() (*entities.PCInfo, error) {
	// Obtener información del sistema
	hostname, _ := os.Hostname()
	osName := runtime.GOOS
	osVersion := "unknown"
	architecture := runtime.GOARCH

	// Generar identificador único para el PC basado en hostname
	pcIdentifier := fmt.Sprintf("%s-%s", hostname, osName)

	// IP que vamos a enviar (por ahora hardcoded)
	ipAddress := "127.0.0.1"

	// Debug logs
	fmt.Printf("DEBUG: RegisterPC called with:\n")
	fmt.Printf("  - Identifier: %s\n", pcIdentifier)
	fmt.Printf("  - IP Address: %s\n", ipAddress)
	fmt.Printf("  - OS: %s\n", osName)
	fmt.Printf("  - Hostname: %s\n", hostname)

	// Obtener APIClient de la conexión existente
	apiClientInterface := r.connectionService.GetAPIClient()
	var apiClient *api.APIClient

	if apiClientInterface != nil {
		if client, ok := apiClientInterface.(*api.APIClient); ok {
			apiClient = client
		}
	}

	// Si no hay conexión, crear una nueva (esto no debería pasar en producción)
	if apiClient == nil || !apiClient.IsConnected() {
		err := r.connectionService.Connect("ws://localhost:8080/ws/client")
		if err != nil {
			return nil, fmt.Errorf("failed to connect to server: %w", err)
		}

		// Obtener el APIClient después de conectar
		apiClientInterface = r.connectionService.GetAPIClient()
		if apiClientInterface != nil {
			if client, ok := apiClientInterface.(*api.APIClient); ok {
				apiClient = client
			}
		}

		if apiClient == nil {
			return nil, fmt.Errorf("failed to get API client after connection")
		}

		// Autenticar como cliente1 (hardcoded por ahora)
		authResp, err := apiClient.ConnectAndAuthenticate("cliente1", "cliente123")
		if err != nil {
			return nil, fmt.Errorf("failed to authenticate: %w", err)
		}

		if !authResp.Success {
			return nil, fmt.Errorf("authentication failed: %s", authResp.Error)
		}
	}

	fmt.Printf("DEBUG: About to call RegisterPC with identifier: %s\n", pcIdentifier)

	// Registrar el PC en el servidor
	regResp, err := apiClient.RegisterPC(pcIdentifier)
	if err != nil {
		fmt.Printf("DEBUG: RegisterPC failed with error: %v\n", err)
		return nil, fmt.Errorf("failed to register PC: %w", err)
	}

	if !regResp.Success {
		fmt.Printf("DEBUG: RegisterPC response not successful: %s\n", regResp.Error)
		return nil, fmt.Errorf("PC registration failed: %s", regResp.Error)
	}

	fmt.Printf("DEBUG: RegisterPC successful, PCID: %s\n", regResp.PCID)

	// Crear PCInfo con la información real
	pcInfo, err := entities.NewPCInfo(
		regResp.PCID, // Usar el ID que retornó el servidor
		hostname,
		osName,
		osVersion,
		architecture,
		ipAddress,           // Usar la IP que enviamos
		"00:00:00:00:00:00", // MAC address por determinar
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create PC info: %w", err)
	}

	r.pcInfo = pcInfo
	return pcInfo, nil
}

func (r *RealPCService) GetPCInfo() *entities.PCInfo {
	if r.pcInfo == nil {
		// Intentar obtener información básica del sistema
		hostname, _ := os.Hostname()
		pcInfo, _ := entities.NewPCInfo(
			"unknown",
			hostname,
			runtime.GOOS,
			"unknown",
			runtime.GOARCH,
			"127.0.0.1",
			"00:00:00:00:00:00",
		)
		r.pcInfo = pcInfo
	}
	return r.pcInfo
}

func (r *RealPCService) GetSystemInfo() map[string]string {
	hostname, _ := os.Hostname()
	return map[string]string{
		"hostname": hostname,
		"os":       runtime.GOOS,
		"arch":     runtime.GOARCH,
		"version":  "unknown",
	}
}

func (r *RealPCService) UpdatePCInfo() error {
	// Actualizar información del PC si es necesario
	if r.pcInfo != nil {
		hostname, _ := os.Hostname()
		r.pcInfo.Update(hostname, "unknown", runtime.GOARCH, "127.0.0.1", "00:00:00:00:00:00")
	}
	return nil
}

// ===== MOCK SERVICES PARA COMPILACIÓN =====

// MockAuthService implementa AuthService para testing/compilación
type MockAuthService struct {
	authenticated bool
	currentUser   *entities.User
}

func (m *MockAuthService) Login(username, password string) (*entities.User, error) {
	m.authenticated = true
	user, _ := entities.NewUser("mock-id", username, "client")
	m.currentUser = user
	return user, nil
}

func (m *MockAuthService) Logout() error {
	m.authenticated = false
	m.currentUser = nil
	return nil
}

func (m *MockAuthService) IsAuthenticated() bool {
	return m.authenticated
}

func (m *MockAuthService) GetCurrentUser() *entities.User {
	return m.currentUser
}

func (m *MockAuthService) ValidateCredentials(username, password string) error {
	return nil
}

// MockConnectionService implementa ConnectionService para testing/compilación
type MockConnectionService struct {
	connected bool
	serverURL string
}

func (m *MockConnectionService) Connect(serverURL string) error {
	m.connected = true
	m.serverURL = serverURL
	return nil
}

func (m *MockConnectionService) Disconnect() error {
	m.connected = false
	return nil
}

func (m *MockConnectionService) IsConnected() bool {
	return m.connected
}

func (m *MockConnectionService) GetServerURL() string {
	return m.serverURL
}

func (m *MockConnectionService) SendHeartbeat() error {
	return nil
}

func (m *MockConnectionService) GetConnectionInfo() *controller.ConnectionInfo {
	return &controller.ConnectionInfo{
		IsConnected: m.connected,
		ServerURL:   m.serverURL,
	}
}

// GetAPIClient retorna nil para el mock service
func (m *MockConnectionService) GetAPIClient() interface{} {
	return nil
}

// MockPCService implementa PCService para testing/compilación
type MockPCService struct {
	pcInfo *entities.PCInfo
}

func (m *MockPCService) RegisterPC() (*entities.PCInfo, error) {
	if m.pcInfo == nil {
		pcInfo, _ := entities.NewPCInfo("mock-pc-id", "mock-hostname", "Windows", "10", "x64", "192.168.1.100", "00:11:22:33:44:55")
		m.pcInfo = pcInfo
	}
	return m.pcInfo, nil
}

func (m *MockPCService) GetPCInfo() *entities.PCInfo {
	if m.pcInfo == nil {
		pcInfo, _ := entities.NewPCInfo("mock-pc-id", "mock-hostname", "Windows", "10", "x64", "192.168.1.100", "00:11:22:33:44:55")
		m.pcInfo = pcInfo
	}
	return m.pcInfo
}

func (m *MockPCService) GetSystemInfo() map[string]string {
	return map[string]string{
		"hostname": "mock-pc",
		"os":       "windows",
		"version":  "10",
		"arch":     "x64",
	}
}

func (m *MockPCService) UpdatePCInfo() error {
	return nil
}
