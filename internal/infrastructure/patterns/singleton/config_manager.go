package singleton

import (
	"sync"
)

// ConfigManager maneja la configuración global de la aplicación (Singleton)
type ConfigManager struct {
	serverURL    string
	timeout      int
	retryCount   int
	debugMode    bool
	initialized  bool
}

// Singleton instance
var (
	configInstance *ConfigManager
	configOnce     sync.Once
)

// GetConfigManager retorna la instancia singleton del ConfigManager
func GetConfigManager() *ConfigManager {
	configOnce.Do(func() {
		configInstance = &ConfigManager{
			serverURL:   "http://localhost:8080",
			timeout:     30,
			retryCount:  3,
			debugMode:   false,
			initialized: true,
		}
	})
	return configInstance
}

// GetServerURL retorna la URL del servidor
func (cm *ConfigManager) GetServerURL() string {
	return cm.serverURL
}

// SetServerURL establece la URL del servidor
func (cm *ConfigManager) SetServerURL(url string) {
	cm.serverURL = url
}

// GetTimeout retorna el timeout en segundos
func (cm *ConfigManager) GetTimeout() int {
	return cm.timeout
}

// SetTimeout establece el timeout en segundos
func (cm *ConfigManager) SetTimeout(timeout int) {
	cm.timeout = timeout
}

// GetRetryCount retorna el número de reintentos
func (cm *ConfigManager) GetRetryCount() int {
	return cm.retryCount
}

// SetRetryCount establece el número de reintentos
func (cm *ConfigManager) SetRetryCount(count int) {
	cm.retryCount = count
}

// IsDebugMode verifica si está en modo debug
func (cm *ConfigManager) IsDebugMode() bool {
	return cm.debugMode
}

// SetDebugMode establece el modo debug
func (cm *ConfigManager) SetDebugMode(debug bool) {
	cm.debugMode = debug
}

// IsInitialized verifica si está inicializado
func (cm *ConfigManager) IsInitialized() bool {
	return cm.initialized
} 