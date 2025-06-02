package entities

import (
	"errors"
	"time"
)

// PCInfo representa la información del PC cliente
type PCInfo struct {
	identifier   string
	hostname     string
	osName       string
	osVersion    string
	architecture string
	ipAddress    string
	macAddress   string
	registeredAt time.Time
	updatedAt    time.Time
}

// NewPCInfo crea una nueva instancia de PCInfo
func NewPCInfo(identifier, hostname, osName, osVersion, architecture, ipAddress, macAddress string) (*PCInfo, error) {
	if identifier == "" {
		return nil, errors.New("PC identifier cannot be empty")
	}
	if hostname == "" {
		return nil, errors.New("hostname cannot be empty")
	}
	if osName == "" {
		return nil, errors.New("OS name cannot be empty")
	}
	if ipAddress == "" {
		return nil, errors.New("IP address cannot be empty")
	}

	now := time.Now().UTC()

	return &PCInfo{
		identifier:   identifier,
		hostname:     hostname,
		osName:       osName,
		osVersion:    osVersion,
		architecture: architecture,
		ipAddress:    ipAddress,
		macAddress:   macAddress,
		registeredAt: now,
		updatedAt:    now,
	}, nil
}

// Getters
func (pc *PCInfo) Identifier() string {
	return pc.identifier
}

func (pc *PCInfo) Hostname() string {
	return pc.hostname
}

func (pc *PCInfo) OSName() string {
	return pc.osName
}

func (pc *PCInfo) OSVersion() string {
	return pc.osVersion
}

func (pc *PCInfo) Architecture() string {
	return pc.architecture
}

func (pc *PCInfo) IPAddress() string {
	return pc.ipAddress
}

func (pc *PCInfo) MACAddress() string {
	return pc.macAddress
}

func (pc *PCInfo) RegisteredAt() time.Time {
	return pc.registeredAt
}

func (pc *PCInfo) UpdatedAt() time.Time {
	return pc.updatedAt
}

// UpdateIPAddress actualiza la dirección IP
func (pc *PCInfo) UpdateIPAddress(newIP string) error {
	if newIP == "" {
		return errors.New("IP address cannot be empty")
	}

	pc.ipAddress = newIP
	pc.updatedAt = time.Now().UTC()
	return nil
}

// Update actualiza la información del PC
func (pc *PCInfo) Update(hostname, osVersion, architecture, ipAddress, macAddress string) {
	if hostname != "" {
		pc.hostname = hostname
	}
	if osVersion != "" {
		pc.osVersion = osVersion
	}
	if architecture != "" {
		pc.architecture = architecture
	}
	if ipAddress != "" {
		pc.ipAddress = ipAddress
	}
	if macAddress != "" {
		pc.macAddress = macAddress
	}

	pc.updatedAt = time.Now().UTC()
}

// GetDisplayName retorna un nombre para mostrar
func (pc *PCInfo) GetDisplayName() string {
	if pc.hostname != "" {
		return pc.hostname
	}
	return pc.identifier
}
