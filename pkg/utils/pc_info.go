package utils

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strings"
)

// GetPCIdentifier obtiene un identificador único para el PC
func GetPCIdentifier() (string, error) {
	// Intentar obtener hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-pc"
	}

	// Obtener información adicional del sistema
	osInfo := runtime.GOOS
	arch := runtime.GOARCH

	// Crear identificador combinando hostname, OS y arquitectura
	identifier := fmt.Sprintf("%s-%s-%s", hostname, osInfo, arch)

	// Limpiar caracteres no válidos
	identifier = strings.ReplaceAll(identifier, " ", "-")
	identifier = strings.ToLower(identifier)

	return identifier, nil
}

// GetMACAddress obtiene la dirección MAC de la primera interfaz de red activa
func GetMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Saltar interfaces loopback y down
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Verificar que tenga dirección MAC
		if len(iface.HardwareAddr) > 0 {
			return iface.HardwareAddr.String(), nil
		}
	}

	return "", fmt.Errorf("no active network interface found")
}

// GetLocalIP obtiene la IP local del PC
func GetLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}

// GetSystemInfo obtiene información general del sistema
func GetSystemInfo() map[string]string {
	info := make(map[string]string)

	info["os"] = runtime.GOOS
	info["arch"] = runtime.GOARCH
	info["num_cpu"] = fmt.Sprintf("%d", runtime.NumCPU())

	if hostname, err := os.Hostname(); err == nil {
		info["hostname"] = hostname
	}

	if ip, err := GetLocalIP(); err == nil {
		info["local_ip"] = ip
	}

	if mac, err := GetMACAddress(); err == nil {
		info["mac_address"] = mac
	}

	return info
}
