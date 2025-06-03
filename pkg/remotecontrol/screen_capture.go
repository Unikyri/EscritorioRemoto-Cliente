package remotecontrol

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"log"

	"github.com/kbinani/screenshot"
)

// ScreenCapture handles screen capture functionality
type ScreenCapture struct {
	displayNum int // Display number for multi-monitor support
}

// NewScreenCapture creates a new ScreenCapture instance
func NewScreenCapture() *ScreenCapture {
	return &ScreenCapture{
		displayNum: 0, // Primary display by default
	}
}

// CaptureFrame captures the current screen as an image
func (sc *ScreenCapture) CaptureFrame() (*image.RGBA, error) {
	// Get the bounds of the display
	bounds := screenshot.GetDisplayBounds(sc.displayNum)

	// Capture the screen
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %w", err)
	}

	return img, nil
}

// CompressToJPEG compresses an image to JPEG format
func (sc *ScreenCapture) CompressToJPEG(img *image.RGBA, quality int) ([]byte, error) {
	if quality < 1 || quality > 100 {
		quality = 75 // Default quality
	}

	// Create a buffer to store the JPEG data
	var buf bytes.Buffer

	// Configure JPEG options
	options := &jpeg.Options{Quality: quality}

	// Encode the image to JPEG
	err := jpeg.Encode(&buf, img, options)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return buf.Bytes(), nil
}

// GetScreenInfo returns information about the current screen
func (sc *ScreenCapture) GetScreenInfo() map[string]interface{} {
	bounds := screenshot.GetDisplayBounds(sc.displayNum)

	return map[string]interface{}{
		"display_num": sc.displayNum,
		"width":       bounds.Dx(),
		"height":      bounds.Dy(),
		"x":           bounds.Min.X,
		"y":           bounds.Min.Y,
	}
}

// GetAvailableDisplays returns the number of available displays
func (sc *ScreenCapture) GetAvailableDisplays() int {
	numDisplays := screenshot.NumActiveDisplays()
	return numDisplays
}

// SetDisplay sets the display to capture from
func (sc *ScreenCapture) SetDisplay(displayNum int) error {
	numDisplays := sc.GetAvailableDisplays()

	if displayNum < 0 || displayNum >= numDisplays {
		return fmt.Errorf("invalid display number %d, available displays: 0-%d",
			displayNum, numDisplays-1)
	}

	sc.displayNum = displayNum
	log.Printf("📺 Display set to %d", displayNum)

	return nil
}

// CaptureRegion captures a specific region of the screen
func (sc *ScreenCapture) CaptureRegion(x, y, width, height int) (*image.RGBA, error) {
	// Create bounds for the region
	bounds := image.Rect(x, y, x+width, y+height)

	// Capture the specified region
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture region: %w", err)
	}

	return img, nil
}

// TestCapture performs a test capture to verify functionality
func (sc *ScreenCapture) TestCapture() error {
	log.Printf("🧪 Testing screen capture...")

	// Test basic capture
	img, err := sc.CaptureFrame()
	if err != nil {
		return fmt.Errorf("test capture failed: %w", err)
	}

	// Test JPEG compression
	_, err = sc.CompressToJPEG(img, 75)
	if err != nil {
		return fmt.Errorf("test JPEG compression failed: %w", err)
	}

	log.Printf("✅ Screen capture test successful (size: %dx%d)",
		img.Bounds().Dx(), img.Bounds().Dy())

	return nil
}
