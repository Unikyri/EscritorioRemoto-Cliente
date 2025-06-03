package remotecontrol

import (
	"fmt"
	"log"
	"sync"
	"time"

	"EscritorioRemoto-Cliente/pkg/api"

	"github.com/google/uuid"
)

// VideoRecorder maneja la grabación de video durante sesiones de control remoto
type VideoRecorder struct {
	isRecording bool
	sessionID   string
	videoID     string // Identificador único para la secuencia de frames de esta grabación
	startTime   time.Time
	mutex       sync.RWMutex
	apiClient   APIClientInterface // interfaz para desacoplar
	// frameIndex int                // contador de frames enviados. Renombrado a currentFrameIndex
	currentFrameIndex int // Contador de frames para la grabación actual
	// config VideoEncoderConfig // Eliminado, ya no usamos VideoEncoder de la misma manera
	// encoder     *VideoEncoder // Eliminado, VideoRecorder manejará directamente los frames JPEG
	recordedFPS float64 // FPS calculado al finalizar la grabación
}

// APIClientInterface define el método necesario para enviar frames y metadatos
type APIClientInterface interface {
	SendVideoFrame(frame interface{}) error
	SendVideoRecordingComplete(videoID string, sessionID string, totalFrames int, fps float64, durationSeconds float64) error
}

// RecordingResult contiene el resultado de una grabación
type RecordingResult struct {
	VideoID       string
	SessionID     string
	FilePath      string  // Ya no es relevante para un MP4, podría eliminarse o reutilizarse
	Duration      int     // Duración en segundos
	FileSizeMB    float64 // Ya no es relevante para un MP4
	FrameCount    int
	CalculatedFPS float64 // FPS calculado
	Error         error
}

// VideoConfig contiene la configuración para el grabador
// Simplificado, ya que no hay codificación MP4 local compleja
type VideoConfig struct {
	// FrameRate int    // Podría ser informativo para el backend o UI, pero no se usa para codificación local
	// Quality string // Ya no es relevante para FFmpeg
	// MaxDuration int // Podría ser útil si se implementa un límite de tiempo de grabación
	// TempDir string // Ya no es necesario para guardar frames para FFmpeg
	// Resolution string // Ya no es relevante para FFmpeg
}

// DefaultVideoConfig retorna una configuración por defecto (muy simplificada)
func DefaultVideoConfig() VideoConfig {
	return VideoConfig{}
}

// NewVideoRecorder crea una nueva instancia del grabador de video
func NewVideoRecorder(_ VideoConfig) *VideoRecorder { // Config ya no se usa extensivamente
	// encoderConfig ya no es necesario
	return &VideoRecorder{
		isRecording:       false,
		currentFrameIndex: 0,
	}
}

// StartRecording inicia la grabación de video para una sesión
func (vr *VideoRecorder) StartRecording(sessionID string) error {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()

	if vr.isRecording {
		return fmt.Errorf("ya está grabando una sesión: %s", vr.sessionID)
	}

	vr.sessionID = sessionID
	vr.videoID = uuid.New().String() // ID único para esta secuencia de frames
	vr.startTime = time.Now()
	vr.isRecording = true
	vr.currentFrameIndex = 0
	vr.recordedFPS = 0

	// Ya no se crea ni inicia un VideoEncoder aquí

	log.Printf("🎬 Grabación de frames individuales iniciada - SessionID: %s, VideoID: %s", sessionID, vr.videoID)
	return nil
}

// StopRecording detiene la grabación y envía metadatos al backend
func (vr *VideoRecorder) StopRecording() (*RecordingResult, error) {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()

	if !vr.isRecording {
		// Devolver un resultado con error o nil si se prefiere no indicar error cuando no hay nada que detener.
		// Por consistencia con el comportamiento anterior, se puede retornar un error.
		return nil, fmt.Errorf("no hay grabación activa")
	}

	vr.isRecording = false
	durationSeconds := int(time.Since(vr.startTime).Seconds())
	finalFrameCount := vr.currentFrameIndex

	if durationSeconds > 0 {
		vr.recordedFPS = float64(finalFrameCount) / float64(durationSeconds)
	} else {
		vr.recordedFPS = 0 // Evitar división por cero si la duración es muy corta
	}

	log.Printf("🎬 Deteniendo grabación de frames. Total Frames: %d, Duración: %d s, FPS: %.2f",
		finalFrameCount, durationSeconds, vr.recordedFPS)

	// Enviar metadatos de finalización al backend
	if vr.apiClient != nil {
		err := vr.apiClient.SendVideoRecordingComplete(vr.videoID, vr.sessionID, finalFrameCount, vr.recordedFPS, float64(durationSeconds))
		if err != nil {
			// Loguear error pero continuar para devolver el resultado local
			log.Printf("❌ Error enviando metadatos de fin de grabación al backend: %v", err)
			// Podríamos incluir este error en el RecordingResult si es crítico
		} else {
			log.Printf("✅ Metadatos de fin de grabación enviados al backend para VideoID: %s", vr.videoID)
		}
	} else {
		log.Printf("⚠️ APIClient no disponible, no se pudieron enviar metadatos de fin de grabación.")
	}

	// Preparar resultado
	result := &RecordingResult{
		VideoID:       vr.videoID,
		SessionID:     vr.sessionID,
		FilePath:      "", // Ya no se genera un archivo MP4 local
		Duration:      durationSeconds,
		FileSizeMB:    0, // Ya no se genera un archivo MP4 local
		FrameCount:    finalFrameCount,
		CalculatedFPS: vr.recordedFPS,
		Error:         nil, // Asumir éxito a menos que algo específico falle aquí
	}

	log.Printf("🎬 Grabación de frames finalizada - VideoID: %s, Duración: %d segundos, Frames: %d, FPS: %.2f",
		result.VideoID, result.Duration, result.FrameCount, result.CalculatedFPS)

	// Resetear para la próxima grabación
	vr.videoID = ""
	vr.sessionID = ""
	vr.currentFrameIndex = 0
	vr.recordedFPS = 0

	return result, nil
}

// AddFrame procesa un frame (asume que frameData es JPEG) y lo envía al backend.
func (vr *VideoRecorder) AddFrame(frameData []byte) error {
	vr.mutex.RLock()
	// Guardar variables necesarias bajo RLock para evitar dataraces si vr.apiClient se modifica concurrentemente
	// o si vr.isRecording cambia durante la ejecución.
	apiClient := vr.apiClient
	isRecording := vr.isRecording
	currentVideoID := vr.videoID           // Usar el videoID de la grabación actual
	currentSessionID := vr.sessionID       // Usar el sessionID de la grabación actual
	frameIdxToSend := vr.currentFrameIndex // Capturar el índice actual para este frame
	vr.mutex.RUnlock()

	if !isRecording {
		return fmt.Errorf("no hay grabación activa para agregar frame")
	}

	// Asumimos que frameData ya es un JPEG listo para enviar.
	// Si se necesitara convertir de raw a JPEG, se haría aquí.

	if apiClient != nil {
		frameUpload := api.VideoFrameUpload{
			SessionID:  currentSessionID,
			VideoID:    currentVideoID,
			FrameIndex: frameIdxToSend, // Usar el índice capturado
			Timestamp:  time.Now().Unix(),
			FrameData:  frameData,
		}
		err := apiClient.SendVideoFrame(frameUpload) // Enviar el frame actual
		if err != nil {
			log.Printf("❌ Error enviando frame %d para VideoID %s: %v", frameIdxToSend, currentVideoID, err)
			// Decide si este error debe detener la grabación o solo loguearse.
			// Por ahora, solo loguear y continuar.
			// return fmt.Errorf("error enviando frame via API: %w", err)
		}
	} else {
		log.Printf("⚠️ APIClient no disponible, no se pudo enviar el frame %d para VideoID %s.", frameIdxToSend, currentVideoID)
		// return fmt.Errorf("APIClient no disponible para enviar frame")
	}

	// Incrementar el contador de frames después de intentar enviar el frame actual.
	// Esto se hace bajo un Lock para asegurar la atomicidad de la actualización.
	vr.mutex.Lock()
	// Solo incrementar si la grabación sigue siendo la misma (mismo videoID)
	// Esto es una doble verificación, aunque el chequeo de isRecording al inicio debería ser suficiente
	// si Start/Stop son los únicos que modifican videoID y isRecording.
	if vr.isRecording && vr.videoID == currentVideoID {
		vr.currentFrameIndex++
	}
	vr.mutex.Unlock()

	return nil // Retornar nil incluso si el envío falla para no detener la captura de pantalla
}

// IsRecording verifica si está grabando actualmente
func (vr *VideoRecorder) IsRecording() bool {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()
	return vr.isRecording
}

// GetCurrentVideoID obtiene el ID del video actual
func (vr *VideoRecorder) GetCurrentVideoID() string {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()
	return vr.videoID
}

// GetCurrentSessionID obtiene el ID de la sesión actual
func (vr *VideoRecorder) GetCurrentSessionID() string {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()
	return vr.sessionID
}

// GetFrameCount obtiene el número de frames capturados
func (vr *VideoRecorder) GetFrameCount() int {
	vr.mutex.RLock()
	defer vr.mutex.RUnlock()
	// ya no depende de vr.encoder
	return vr.currentFrameIndex
}

// SetAPIClient permite inyectar el cliente API
func (vr *VideoRecorder) SetAPIClient(client APIClientInterface) {
	vr.mutex.Lock()
	defer vr.mutex.Unlock()
	vr.apiClient = client
}
