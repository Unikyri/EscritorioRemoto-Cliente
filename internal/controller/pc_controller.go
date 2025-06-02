package controller

import (
	"EscritorioRemoto-Cliente/internal/infrastructure/patterns/observer"
	"EscritorioRemoto-Cliente/internal/model/entities"
	"fmt"
)

// PCController maneja las operaciones relacionadas con el PC
type PCController struct {
	pcService    PCService
	eventManager *observer.EventManager
}

// PCService interface para el servicio de PC
type PCService interface {
	RegisterPC() (*entities.PCInfo, error)
	GetPCInfo() *entities.PCInfo
	GetSystemInfo() map[string]string
	UpdatePCInfo() error
}

// PCRegistrationResponse representa la respuesta de registro de PC
type PCRegistrationResponse struct {
	Success bool              `json:"success"`
	PCInfo  *entities.PCInfo  `json:"pc_info,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// PCInfoResponse representa la respuesta de información del PC
type PCInfoResponse struct {
	Success bool              `json:"success"`
	PCInfo  *entities.PCInfo  `json:"pc_info,omitempty"`
	Error   string            `json:"error,omitempty"`
}

// NewPCController crea un nuevo controlador de PC
func NewPCController(pcService PCService) *PCController {
	return &PCController{
		pcService:    pcService,
		eventManager: observer.GetInstance(),
	}
}

// RegisterPC maneja el registro del PC en el servidor
func (pc *PCController) RegisterPC() PCRegistrationResponse {
	// Intentar registrar el PC
	pcInfo, err := pc.pcService.RegisterPC()
	if err != nil {
		return PCRegistrationResponse{
			Success: false,
			Error:   fmt.Sprintf("PC registration failed: %v", err),
		}
	}

	// Publicar evento de registro exitoso
	pc.eventManager.Publish(observer.Event{
		Type: "pc_registered",
		Data: map[string]interface{}{
			"pc_id":      pcInfo.Identifier(),
			"hostname":   pcInfo.Hostname(),
			"os":         pcInfo.OSName(),
			"ip_address": pcInfo.IPAddress(),
		},
	})

	return PCRegistrationResponse{
		Success: true,
		PCInfo:  pcInfo,
	}
}

// GetPCInfo obtiene la información del PC
func (pc *PCController) GetPCInfo() PCInfoResponse {
	pcInfo := pc.pcService.GetPCInfo()
	if pcInfo == nil {
		return PCInfoResponse{
			Success: false,
			Error:   "Failed to retrieve PC information",
		}
	}

	return PCInfoResponse{
		Success: true,
		PCInfo:  pcInfo,
	}
}

// GetSystemInfo obtiene información del sistema
func (pc *PCController) GetSystemInfo() map[string]string {
	return pc.pcService.GetSystemInfo()
}

// UpdatePCInfo actualiza la información del PC
func (pc *PCController) UpdatePCInfo() error {
	err := pc.pcService.UpdatePCInfo()
	if err != nil {
		return fmt.Errorf("failed to update PC info: %w", err)
	}

	// Publicar evento de actualización
	pc.eventManager.Publish(observer.Event{
		Type: "pc_info_updated",
		Data: map[string]interface{}{
			"updated_at": "now",
		},
	})

	return nil
} 