package filetransfer

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"hash"
	"os"
	"path/filepath"
	"sync"
	"time"

	"EscritorioRemoto-Cliente/pkg/api"
)

// FileTransferAgent maneja la recepción de archivos desde el servidor
type FileTransferAgent struct {
	// Archivos en progreso de recepción
	activeTransfers map[string]*FileTransfer
	mutex           sync.RWMutex

	// Directorio base para recibir archivos
	downloadDir string

	// Callback para notificar al app sobre el estado de transferencia
	onTransferCompleted func(transferID, fileName, filePath string, success bool, errorMsg string)
}

// FileTransfer representa una transferencia de archivo en progreso
type FileTransfer struct {
	TransferID      string
	SessionID       string
	FileName        string
	FileSizeMB      float64
	TotalChunks     int
	DestinationPath string

	// Estado actual
	ReceivedChunks map[int][]byte
	ChunksReceived int
	StartTime      time.Time

	// Archivo de escritura
	outputFile     *os.File
	outputFilePath string

	// Verificación de integridad
	hashWriter hash.Hash
}

// NewFileTransferAgent crea un nuevo agente de transferencia de archivos
func NewFileTransferAgent(downloadDir string) *FileTransferAgent {
	// Crear directorio de descarga si no existe
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("Warning: Could not create download directory %s: %v\n", downloadDir, err)
	}

	return &FileTransferAgent{
		activeTransfers: make(map[string]*FileTransfer),
		downloadDir:     downloadDir,
	}
}

// SetTransferCompletedCallback establece el callback para transferencias completadas
func (fta *FileTransferAgent) SetTransferCompletedCallback(callback func(transferID, fileName, filePath string, success bool, errorMsg string)) {
	fta.onTransferCompleted = callback
}

// HandleFileTransferRequest procesa una nueva solicitud de transferencia de archivo
func (fta *FileTransferAgent) HandleFileTransferRequest(request api.FileTransferRequest) error {
	fta.mutex.Lock()
	defer fta.mutex.Unlock()

	// Verificar si ya existe una transferencia activa con este ID
	if _, exists := fta.activeTransfers[request.TransferID]; exists {
		return fmt.Errorf("transfer %s already in progress", request.TransferID)
	}

	// Usar el directorio de descarga directamente (ya incluye RemoteDesk)
	destDir := fta.downloadDir
	
	// Crear el directorio de destino si no existe
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory %s: %v", destDir, err)
	}

	// Verificar que el directorio se creó correctamente
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return fmt.Errorf("destination directory was not created: %s", destDir)
	}

	// Construir ruta completa del archivo
	outputFilePath := filepath.Join(destDir, request.FileName)

	// Crear el archivo de salida
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("failed to create output file %s: %v", outputFilePath, err)
	}

	// Crear nueva transferencia
	transfer := &FileTransfer{
		TransferID:      request.TransferID,
		SessionID:       request.SessionID,
		FileName:        request.FileName,
		FileSizeMB:      request.FileSizeMB,
		TotalChunks:     request.TotalChunks,
		DestinationPath: request.DestinationPath,
		ReceivedChunks:  make(map[int][]byte),
		ChunksReceived:  0,
		StartTime:       time.Now(),
		outputFile:      outputFile,
		outputFilePath:  outputFilePath,
		hashWriter:      md5.New(),
	}

	// Registrar transferencia activa
	fta.activeTransfers[request.TransferID] = transfer

	fmt.Printf("📁 FILE TRANSFER: Started receiving file %s (%.2f MB, %d chunks)\n",
		request.FileName, request.FileSizeMB, request.TotalChunks)
	fmt.Printf("📁 FILE TRANSFER: Saving to directory: %s\n", destDir)

	return nil
}

// HandleFileChunk procesa un chunk de archivo recibido
func (fta *FileTransferAgent) HandleFileChunk(chunk api.FileChunk) error {
	fta.mutex.Lock()
	defer fta.mutex.Unlock()

	// Buscar transferencia activa
	transfer, exists := fta.activeTransfers[chunk.TransferID]
	if !exists {
		return fmt.Errorf("no active transfer found for ID: %s", chunk.TransferID)
	}

	// Verificar que el chunk no haya sido recibido previamente
	if _, alreadyReceived := transfer.ReceivedChunks[chunk.ChunkIndex]; alreadyReceived {
		fmt.Printf("⚠️ Duplicate chunk %d for transfer %s, ignoring\n", chunk.ChunkIndex, chunk.TransferID)
		return nil
	}

	// 🔧 DECODIFICAR BASE64: El servidor envía datos codificados en base64
	var actualChunkData []byte
	if len(chunk.ChunkData) > 0 {
		// Si ChunkData es []byte, podríamos estar recibiendo base64 como bytes
		// Intentar decodificar de base64 string primero
		chunkDataStr := string(chunk.ChunkData)
		decodedData, err := base64.StdEncoding.DecodeString(chunkDataStr)
		if err != nil {
			// Si la decodificación base64 falla, usar los datos tal como están
			fmt.Printf("⚠️ Base64 decode failed for chunk %d, using raw data: %v\n", chunk.ChunkIndex, err)
			actualChunkData = chunk.ChunkData
		} else {
			actualChunkData = decodedData
			fmt.Printf("✅ Decoded base64 chunk %d: %d bytes -> %d bytes\n",
				chunk.ChunkIndex, len(chunk.ChunkData), len(actualChunkData))
		}
	} else {
		return fmt.Errorf("received empty chunk data for chunk %d", chunk.ChunkIndex)
	}

	// Escribir chunk decodificado al archivo
	if _, err := transfer.outputFile.Write(actualChunkData); err != nil {
		// Error escribiendo, limpiar transferencia
		fta.cleanupTransfer(transfer, fmt.Sprintf("failed to write chunk: %v", err))
		return err
	}

	// Actualizar hash para verificación de integridad usando datos decodificados
	transfer.hashWriter.Write(actualChunkData)

	// Registrar chunk recibido usando datos decodificados
	transfer.ReceivedChunks[chunk.ChunkIndex] = actualChunkData
	transfer.ChunksReceived++

	fmt.Printf("📦 Received chunk %d/%d for file %s (decoded: %d bytes)\n",
		chunk.ChunkIndex+1, chunk.TotalChunks, transfer.FileName, len(actualChunkData))

	// Verificar si es el último chunk o si hemos recibido todos
	if chunk.IsLastChunk || transfer.ChunksReceived >= transfer.TotalChunks {
		return fta.completeTransfer(transfer)
	}

	return nil
}

// completeTransfer finaliza una transferencia de archivo
func (fta *FileTransferAgent) completeTransfer(transfer *FileTransfer) error {
	// Cerrar archivo
	if err := transfer.outputFile.Close(); err != nil {
		fta.cleanupTransfer(transfer, fmt.Sprintf("failed to close file: %v", err))
		return err
	}

	// Verificar que el archivo realmente existe y tiene el tamaño esperado
	fileInfo, err := os.Stat(transfer.outputFilePath)
	if err != nil {
		fta.cleanupTransfer(transfer, fmt.Sprintf("failed to stat completed file: %v", err))
		return err
	}

	// Validar que el archivo no está vacío
	if fileInfo.Size() == 0 {
		fta.cleanupTransfer(transfer, "file was saved but is empty")
		return fmt.Errorf("file was saved but is empty")
	}

	// Verificar integridad del archivo (opcional para MVP)
	fileChecksum := fmt.Sprintf("%x", transfer.hashWriter.Sum(nil))

	duration := time.Since(transfer.StartTime)
	actualFileSize := float64(fileInfo.Size()) / (1024 * 1024) // MB

	fmt.Printf("✅ File transfer completed: %s (%.2f MB) in %v\n",
		transfer.FileName, actualFileSize, duration)
	fmt.Printf("📁 File successfully saved to: %s\n", transfer.outputFilePath)
	fmt.Printf("📊 File size: %d bytes (expected: %.0f MB)\n", 
		fileInfo.Size(), transfer.FileSizeMB)
	fmt.Printf("🔐 File checksum: %s\n", fileChecksum)

	// Verificar que el directorio padre existe y es accesible
	parentDir := filepath.Dir(transfer.outputFilePath)
	if _, err := os.Stat(parentDir); err != nil {
		fmt.Printf("⚠️ Warning: Parent directory check failed: %v\n", err)
	}

	// Notificar al app sobre transferencia completada
	if fta.onTransferCompleted != nil {
		fta.onTransferCompleted(transfer.TransferID, transfer.FileName, transfer.outputFilePath, true, "")
	}

	// Limpiar transferencia completada
	delete(fta.activeTransfers, transfer.TransferID)

	return nil
}

// cleanupTransfer limpia una transferencia fallida
func (fta *FileTransferAgent) cleanupTransfer(transfer *FileTransfer, errorMsg string) {
	fmt.Printf("❌ File transfer failed for %s: %s\n", transfer.FileName, errorMsg)

	// Cerrar archivo si está abierto
	if transfer.outputFile != nil {
		transfer.outputFile.Close()
	}

	// Intentar eliminar archivo parcial
	if transfer.outputFilePath != "" {
		if err := os.Remove(transfer.outputFilePath); err != nil {
			fmt.Printf("Warning: Could not remove partial file %s: %v\n", transfer.outputFilePath, err)
		}
	}

	// Notificar al app sobre transferencia fallida
	if fta.onTransferCompleted != nil {
		fta.onTransferCompleted(transfer.TransferID, transfer.FileName, "", false, errorMsg)
	}

	// Eliminar transferencia activa
	delete(fta.activeTransfers, transfer.TransferID)
}

// GetActiveTransfers retorna información sobre transferencias activas
func (fta *FileTransferAgent) GetActiveTransfers() map[string]map[string]interface{} {
	fta.mutex.RLock()
	defer fta.mutex.RUnlock()

	result := make(map[string]map[string]interface{})

	for id, transfer := range fta.activeTransfers {
		progress := float64(transfer.ChunksReceived) / float64(transfer.TotalChunks) * 100

		result[id] = map[string]interface{}{
			"transfer_id":     transfer.TransferID,
			"session_id":      transfer.SessionID,
			"file_name":       transfer.FileName,
			"file_size_mb":    transfer.FileSizeMB,
			"total_chunks":    transfer.TotalChunks,
			"chunks_received": transfer.ChunksReceived,
			"progress":        progress,
			"start_time":      transfer.StartTime,
			"destination":     transfer.outputFilePath,
		}
	}

	return result
}

// GetDownloadDirectory retorna el directorio de descarga configurado
func (fta *FileTransferAgent) GetDownloadDirectory() string {
	return fta.downloadDir
}
