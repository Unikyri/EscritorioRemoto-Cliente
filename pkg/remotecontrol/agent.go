package remotecontrol

import (
	"fmt"
	"log"
	"sync"
	"time"

	"EscritorioRemoto-Cliente/pkg/api"
)

// RemoteControlAgent coordinates screen capture and input simulation
type RemoteControlAgent struct {
	screenCapture   *ScreenCapture
	inputSimulator  *InputSimulator
	isActive        bool
	activeSessionID string
	mutex           sync.RWMutex

	// Configuration
	frameRate    int           // Frames per second for screen capture
	jpegQuality  int           // JPEG compression quality (1-100)
	captureDelay time.Duration // Delay between captures

	// Channels for coordination
	stopCapture chan struct{}
	frameOutput chan api.ScreenFrame
}

// NewRemoteControlAgent creates a new RemoteControlAgent
func NewRemoteControlAgent() *RemoteControlAgent {
	return &RemoteControlAgent{
		screenCapture:  NewScreenCapture(),
		inputSimulator: NewInputSimulator(),
		isActive:       false,
		frameRate:      15,                             // 15 FPS by default
		jpegQuality:    75,                             // 75% quality by default
		captureDelay:   66 * time.Millisecond,          // ~15 FPS
		frameOutput:    make(chan api.ScreenFrame, 10), // Buffer for 10 frames
	}
}

// StartSession begins screen capture and prepares for input control
func (a *RemoteControlAgent) StartSession(sessionID string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.isActive {
		return fmt.Errorf("session already active: %s", a.activeSessionID)
	}

	log.Printf("🎬 Starting remote control session: %s", sessionID)

	a.activeSessionID = sessionID
	a.isActive = true
	a.stopCapture = make(chan struct{})

	// Start screen capture in goroutine
	go a.captureLoop()

	log.Printf("✅ Remote control session started successfully: %s", sessionID)
	return nil
}

// StopSession ends screen capture and input control
func (a *RemoteControlAgent) StopSession() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.isActive {
		return fmt.Errorf("no active session")
	}

	log.Printf("🛑 Stopping remote control session: %s", a.activeSessionID)

	// Signal capture loop to stop
	close(a.stopCapture)

	a.isActive = false
	a.activeSessionID = ""

	log.Printf("✅ Remote control session stopped successfully")
	return nil
}

// IsActive returns whether a session is currently active
func (a *RemoteControlAgent) IsActive() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.isActive
}

// GetActiveSessionID returns the current active session ID
func (a *RemoteControlAgent) GetActiveSessionID() string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.activeSessionID
}

// ProcessInputCommand processes an incoming input command
func (a *RemoteControlAgent) ProcessInputCommand(command api.InputCommand) error {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if !a.isActive {
		return fmt.Errorf("no active session")
	}

	if command.SessionID != a.activeSessionID {
		return fmt.Errorf("command session ID %s does not match active session %s",
			command.SessionID, a.activeSessionID)
	}

	log.Printf("🎮 Processing input command: type=%s, action=%s", command.EventType, command.Action)

	switch command.EventType {
	case "mouse":
		return a.inputSimulator.ProcessMouseCommand(command)
	case "keyboard":
		return a.inputSimulator.ProcessKeyboardCommand(command)
	default:
		return fmt.Errorf("unknown input event type: %s", command.EventType)
	}
}

// GetFrameOutput returns the channel for receiving captured frames
func (a *RemoteControlAgent) GetFrameOutput() <-chan api.ScreenFrame {
	return a.frameOutput
}

// SetFrameRate sets the capture frame rate (FPS)
func (a *RemoteControlAgent) SetFrameRate(fps int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if fps < 1 {
		fps = 1
	} else if fps > 30 {
		fps = 30
	}

	a.frameRate = fps
	a.captureDelay = time.Duration(1000/fps) * time.Millisecond

	log.Printf("📹 Frame rate set to %d FPS (delay: %v)", fps, a.captureDelay)
}

// SetJPEGQuality sets the JPEG compression quality (1-100)
func (a *RemoteControlAgent) SetJPEGQuality(quality int) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if quality < 1 {
		quality = 1
	} else if quality > 100 {
		quality = 100
	}

	a.jpegQuality = quality
	log.Printf("🖼️ JPEG quality set to %d%%", quality)
}

// captureLoop runs the screen capture loop
func (a *RemoteControlAgent) captureLoop() {
	log.Printf("📸 Starting screen capture loop (FPS: %d, Quality: %d%%)",
		a.frameRate, a.jpegQuality)

	ticker := time.NewTicker(a.captureDelay)
	defer ticker.Stop()

	sequenceNum := int64(0)

	for {
		select {
		case <-a.stopCapture:
			log.Printf("🔚 Screen capture loop stopped")
			return

		case <-ticker.C:
			// Capture screen frame
			frame, err := a.screenCapture.CaptureFrame()
			if err != nil {
				log.Printf("❌ Error capturing screen frame: %v", err)
				continue
			}

			// Compress to JPEG
			frameData, err := a.screenCapture.CompressToJPEG(frame, a.jpegQuality)
			if err != nil {
				log.Printf("❌ Error compressing frame: %v", err)
				continue
			}

			// Create frame message
			screenFrame := api.ScreenFrame{
				SessionID:   a.activeSessionID,
				Timestamp:   time.Now().Unix(),
				Width:       frame.Bounds().Dx(),
				Height:      frame.Bounds().Dy(),
				Format:      "jpeg",
				Quality:     a.jpegQuality,
				FrameData:   frameData,
				SequenceNum: sequenceNum,
			}

			// Send to output channel (non-blocking)
			select {
			case a.frameOutput <- screenFrame:
				sequenceNum++
			default:
				// Channel is full, skip this frame
				log.Printf("⚠️ Frame output channel full, skipping frame %d", sequenceNum)
			}
		}
	}
}

// GetCapabilities returns the capabilities of this agent
func (a *RemoteControlAgent) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"screen_capture": map[string]interface{}{
			"max_fps":           30,
			"min_fps":           1,
			"supported_formats": []string{"jpeg"},
			"compression":       true,
		},
		"input_control": map[string]interface{}{
			"mouse":    true,
			"keyboard": true,
			"scroll":   true,
		},
		"current_settings": map[string]interface{}{
			"fps":          a.frameRate,
			"jpeg_quality": a.jpegQuality,
		},
	}
}

// TestScreenCapture executes screen capture test
func (a *RemoteControlAgent) TestScreenCapture() error {
	return a.screenCapture.TestCapture()
}

// TestInputSimulation executes input simulation test
func (a *RemoteControlAgent) TestInputSimulation() error {
	return a.inputSimulator.TestInput()
}
